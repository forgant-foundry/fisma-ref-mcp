//go:build embed_nomic || embed_qwen3 || embed_openai_small

package vec_store

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/philippgille/chromem-go"
)

const (
	CollectionControls     = "controls"
	CollectionFismaMetrics = "fisma_metrics"
	CollectionCSF          = "csf_v2"
	CollectionFedRAMP      = "fedramp_20x"
)

// EmbedConfig holds the embedding provider settings needed to query the vector index.
type EmbedConfig struct {
	Provider  string // "openai" | "ollama"
	Model     string
	OpenAIKey string
	OllamaURL string
}

// RawHit is a single vector search result before relational resolution.
type RawHit struct {
	ID         string
	Similarity float32
	Source     string
	DocType    string // optional sub-type, e.g. "ksi" vs "requirement" in fedramp_20x
}

// VectorDB manages the chromem-go vector index and query routing.
type VectorDB struct {
	db   *chromem.DB
	cols map[string]*chromem.Collection
}

// NewVectorDB constructs a VectorDB. It loads the pre-built index if one is
// embedded in the binary; otherwise it builds the index at startup.
func NewVectorDB(ctx context.Context, cfg EmbedConfig, controls []nist_800_53.Control, metrics []fisma.Metric, subcategories []nist_csf.Subcategory, frmr *fedramp.Catalog) (*VectorDB, error) {
	embFn, err := embeddingFunc(cfg)
	if err != nil {
		return nil, err
	}

	prebuiltData, meta, hasPrebuilt := PrebuiltVector()
	if hasPrebuilt {
		return loadPrebuilt(ctx, prebuiltData, meta, cfg, embFn)
	}
	return buildFromDocuments(ctx, controls, metrics, subcategories, frmr, embFn)
}

func loadPrebuilt(ctx context.Context, data []byte, meta *VectorMeta, cfg EmbedConfig, embFn chromem.EmbeddingFunc) (*VectorDB, error) {
	effective := effectiveModel(cfg)
	if meta.Provider != cfg.Provider || meta.Model != effective {
		return nil, fmt.Errorf(
			"pre-built vector index uses %s/%s but runtime is configured for %s/%s: "+
				"rebuild the index with 'make embed-<model>'",
			meta.Provider, meta.Model, cfg.Provider, effective,
		)
	}

	db := chromem.NewDB()
	if err := db.ImportFromReader(bytes.NewReader(data), ""); err != nil {
		return nil, fmt.Errorf("import pre-built vector index: %w", err)
	}

	cols := make(map[string]*chromem.Collection)
	for _, name := range []string{CollectionControls, CollectionFismaMetrics, CollectionCSF, CollectionFedRAMP} {
		col, err := db.GetOrCreateCollection(name, nil, embFn)
		if err != nil {
			return nil, fmt.Errorf("attach embedding function to collection %q: %w", name, err)
		}
		cols[name] = col
	}
	return &VectorDB{db: db, cols: cols}, nil
}

func buildFromDocuments(ctx context.Context, controls []nist_800_53.Control, metrics []fisma.Metric, subcategories []nist_csf.Subcategory, frmr *fedramp.Catalog, embFn chromem.EmbeddingFunc) (*VectorDB, error) {
	db := chromem.NewDB()
	cols := make(map[string]*chromem.Collection)

	controlCol, err := db.GetOrCreateCollection(CollectionControls, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create controls collection: %w", err)
	}
	cols[CollectionControls] = controlCol
	var controlDocs []chromem.Document
	for _, c := range controls {
		if content := BuildControlDocument(c); content != "" {
			controlDocs = append(controlDocs, chromem.Document{
				ID: strings.ToUpper(c.ID), Content: content,
				Metadata: map[string]string{"family": c.FamilyID, "is_enhancement": fmt.Sprintf("%v", c.IsEnhancement)},
			})
		}
	}
	if err := controlCol.AddDocuments(ctx, controlDocs, 0); err != nil {
		return nil, fmt.Errorf("index controls: %w", err)
	}

	fismaCol, err := db.GetOrCreateCollection(CollectionFismaMetrics, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create fisma_metrics collection: %w", err)
	}
	cols[CollectionFismaMetrics] = fismaCol
	var fismaDocs []chromem.Document
	for _, m := range metrics {
		if content := BuildMetricDocument(m); content != "" {
			fismaDocs = append(fismaDocs, chromem.Document{
				ID: fmt.Sprintf("%d", m.ID), Content: content,
				Metadata: map[string]string{"domain": m.Domain, "review_cycle": m.ReviewCycle},
			})
		}
	}
	if err := fismaCol.AddDocuments(ctx, fismaDocs, 0); err != nil {
		return nil, fmt.Errorf("index fisma metrics: %w", err)
	}

	csfCol, err := db.GetOrCreateCollection(CollectionCSF, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create csf_v2 collection: %w", err)
	}
	cols[CollectionCSF] = csfCol
	var csfDocs []chromem.Document
	for _, s := range subcategories {
		if content := BuildSubcategoryDocument(s); content != "" {
			csfDocs = append(csfDocs, chromem.Document{
				ID: s.ID, Content: content,
				Metadata: map[string]string{"category_id": s.CategoryID, "function_id": s.FunctionID},
			})
		}
	}
	if err := csfCol.AddDocuments(ctx, csfDocs, 0); err != nil {
		return nil, fmt.Errorf("index csf subcategories: %w", err)
	}

	fedCol, err := db.GetOrCreateCollection(CollectionFedRAMP, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create fedramp_20x collection: %w", err)
	}
	cols[CollectionFedRAMP] = fedCol
	var fedDocs []chromem.Document
	for _, theme := range frmr.KSIThemes {
		for _, ind := range theme.Indicators {
			if content := BuildKSIDocument(ind); content != "" {
				fedDocs = append(fedDocs, chromem.Document{
					ID: ind.ID, Content: content,
					Metadata: map[string]string{"theme_id": ind.ThemeID, "type": "ksi"},
				})
			}
		}
	}
	for _, rc := range frmr.Requirements {
		for _, req := range rc.Requirements {
			if content := BuildRequirementDocument(req); content != "" {
				fedDocs = append(fedDocs, chromem.Document{
					ID: req.ID, Content: content,
					Metadata: map[string]string{"category": req.Category, "type": "requirement"},
				})
			}
		}
	}
	for _, t := range frmr.Terms {
		if content := BuildTermDocument(t); content != "" {
			fedDocs = append(fedDocs, chromem.Document{
				ID: t.ID, Content: content,
				Metadata: map[string]string{"type": "term"},
			})
		}
	}
	if err := fedCol.AddDocuments(ctx, fedDocs, 0); err != nil {
		return nil, fmt.Errorf("index fedramp documents: %w", err)
	}

	return &VectorDB{db: db, cols: cols}, nil
}

// Query returns raw vector hits sorted by similarity. Resolution to full records
// is the caller's responsibility.
func (v *VectorDB) Query(ctx context.Context, query string, limit int, source string) ([]RawHit, error) {
	var hits []RawHit

	query_col := func(name, src string) error {
		col, ok := v.cols[name]
		if !ok || col.Count() == 0 {
			return nil
		}
		results, err := col.Query(ctx, query, limit, nil, nil)
		if err != nil {
			return fmt.Errorf("vector search %s: %w", src, err)
		}
		for _, h := range results {
			hits = append(hits, RawHit{ID: h.ID, Similarity: h.Similarity, Source: src, DocType: h.Metadata["type"]})
		}
		return nil
	}

	if source == "" || source == "nist_800_53" {
		if err := query_col(CollectionControls, "nist_800_53"); err != nil {
			return nil, err
		}
	}
	if source == "" || source == "fisma_fy2025" {
		if err := query_col(CollectionFismaMetrics, "fisma_fy2025"); err != nil {
			return nil, err
		}
	}
	if source == "" || source == "nist_csf_v2" {
		if err := query_col(CollectionCSF, "nist_csf_v2"); err != nil {
			return nil, err
		}
	}
	if source == "" || source == "fedramp_20x" {
		if err := query_col(CollectionFedRAMP, "fedramp_20x"); err != nil {
			return nil, err
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Similarity > hits[j].Similarity })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func embeddingFunc(cfg EmbedConfig) (chromem.EmbeddingFunc, error) {
	switch cfg.Provider {
	case "openai":
		model := cfg.Model
		if model == "" {
			model = string(chromem.EmbeddingModelOpenAI3Small)
		}
		return chromem.NewEmbeddingFuncOpenAI(cfg.OpenAIKey, chromem.EmbeddingModelOpenAI(model)), nil
	case "ollama":
		model := cfg.Model
		if model == "" {
			model = "nomic-embed-text"
		}
		base := cfg.OllamaURL
		if base == "" {
			base = "http://localhost:11434"
		}
		base = strings.TrimRight(base, "/")
		if !strings.HasSuffix(base, "/api") {
			base += "/api"
		}
		return chromem.NewEmbeddingFuncOllama(model, base), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q (use \"openai\" or \"ollama\")", cfg.Provider)
	}
}

func effectiveModel(cfg EmbedConfig) string {
	if cfg.Model != "" {
		return cfg.Model
	}
	switch cfg.Provider {
	case "openai":
		return string(chromem.EmbeddingModelOpenAI3Small)
	case "ollama":
		return "nomic-embed-text"
	}
	return ""
}

