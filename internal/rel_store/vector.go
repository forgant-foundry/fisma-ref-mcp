//go:build !no_embeddings

package rel_store

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/vec_store"
	"github.com/philippgille/chromem-go"
)

const (
	collectionControls     = "controls"
	collectionFismaMetrics = "fisma_metrics"
	collectionCSF          = "csf_v2"
	collectionFedRAMP      = "fedramp_20x"
)

type vectorDB struct {
	db   *chromem.DB
	cols map[string]*chromem.Collection
	rel  *relationalDB
}

func newVectorDB(ctx context.Context, cfg Config, controls []nist_800_53.Control, metrics []fisma.Metric, subcategories []nist_csf.Subcategory, frmr *fedramp.Catalog, rel *relationalDB) (*vectorDB, error) {
	embFn, err := embeddingFunc(cfg)
	if err != nil {
		return nil, err
	}

	prebuiltData, meta, hasPrebuilt := vec_store.PrebuiltVector()
	if hasPrebuilt {
		return loadPrebuilt(ctx, prebuiltData, meta, cfg, embFn, rel)
	}
	return buildFromDocuments(ctx, controls, metrics, subcategories, frmr, embFn, rel)
}

// loadPrebuilt imports the serialised chromem-go DB embedded at build time.
// It validates that the runtime embedding provider and model match those used
// during generation so that query vectors are in the same space as index vectors.
func loadPrebuilt(ctx context.Context, data []byte, meta *vec_store.VectorMeta, cfg Config, embFn chromem.EmbeddingFunc, rel *relationalDB) (*vectorDB, error) {
	effectiveModel := effectiveEmbeddingModel(cfg)
	if meta.Provider != cfg.EmbeddingProvider || meta.Model != effectiveModel {
		return nil, fmt.Errorf(
			"pre-built vector index uses %s/%s but runtime is configured for %s/%s: "+
				"rebuild the index with 'go generate ./internal/nist' using matching settings",
			meta.Provider, meta.Model,
			cfg.EmbeddingProvider, effectiveModel,
		)
	}

	db := chromem.NewDB()
	if err := db.ImportFromReader(bytes.NewReader(data), ""); err != nil {
		return nil, fmt.Errorf("import pre-built vector index: %w", err)
	}

	// Attach the embedding function to each known collection so that query text
	// is embedded with the same model as the stored vectors. GetOrCreateCollection
	// creates an empty collection if it wasn't present in the prebuilt index
	// (e.g., a collection added after the index was generated); in that case
	// the collection simply returns no results until the index is regenerated.
	cols := make(map[string]*chromem.Collection)
	for _, name := range []string{collectionControls, collectionFismaMetrics, collectionCSF, collectionFedRAMP} {
		col, err := db.GetOrCreateCollection(name, nil, embFn)
		if err != nil {
			return nil, fmt.Errorf("attach embedding function to collection %q: %w", name, err)
		}
		cols[name] = col
	}

	return &vectorDB{db: db, cols: cols, rel: rel}, nil
}

// buildFromDocuments generates embeddings at startup. This is used when no
// pre-built index is embedded in the binary.
func buildFromDocuments(ctx context.Context, controls []nist_800_53.Control, metrics []fisma.Metric, subcategories []nist_csf.Subcategory, frmr *fedramp.Catalog, embFn chromem.EmbeddingFunc, rel *relationalDB) (*vectorDB, error) {
	db := chromem.NewDB()
	cols := make(map[string]*chromem.Collection)

	controlCol, err := db.GetOrCreateCollection(collectionControls, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create controls collection: %w", err)
	}
	cols[collectionControls] = controlCol

	controlDocs := make([]chromem.Document, 0, len(controls))
	for _, c := range controls {
		content := buildControlDocument(c)
		if content == "" {
			continue
		}
		controlDocs = append(controlDocs, chromem.Document{
			ID:      strings.ToUpper(c.ID),
			Content: content,
			Metadata: map[string]string{
				"family":         c.FamilyID,
				"is_enhancement": fmt.Sprintf("%v", c.IsEnhancement),
			},
		})
	}
	if err := controlCol.AddDocuments(ctx, controlDocs, 0); err != nil {
		return nil, fmt.Errorf("index controls: %w", err)
	}

	fismaCol, err := db.GetOrCreateCollection(collectionFismaMetrics, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create fisma_metrics collection: %w", err)
	}
	cols[collectionFismaMetrics] = fismaCol

	fismaDocs := make([]chromem.Document, 0, len(metrics))
	for _, m := range metrics {
		content := buildMetricDocument(m)
		if content == "" {
			continue
		}
		fismaDocs = append(fismaDocs, chromem.Document{
			ID:      fmt.Sprintf("%d", m.ID),
			Content: content,
			Metadata: map[string]string{
				"domain":       m.Domain,
				"review_cycle": m.ReviewCycle,
			},
		})
	}
	if err := fismaCol.AddDocuments(ctx, fismaDocs, 0); err != nil {
		return nil, fmt.Errorf("index fisma metrics: %w", err)
	}

	csfCol, err := db.GetOrCreateCollection(collectionCSF, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create csf_v2 collection: %w", err)
	}
	cols[collectionCSF] = csfCol

	csfDocs := make([]chromem.Document, 0, len(subcategories))
	for _, s := range subcategories {
		content := buildSubcategoryDocument(s)
		if content == "" {
			continue
		}
		csfDocs = append(csfDocs, chromem.Document{
			ID:      s.ID,
			Content: content,
			Metadata: map[string]string{
				"category_id": s.CategoryID,
				"function_id": s.FunctionID,
			},
		})
	}
	if err := csfCol.AddDocuments(ctx, csfDocs, 0); err != nil {
		return nil, fmt.Errorf("index csf subcategories: %w", err)
	}

	fedCol, err := db.GetOrCreateCollection(collectionFedRAMP, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create fedramp_20x collection: %w", err)
	}
	cols[collectionFedRAMP] = fedCol

	var fedDocs []chromem.Document
	for _, theme := range frmr.KSIThemes {
		for _, ind := range theme.Indicators {
			content := buildKSIDocument(ind)
			if content == "" {
				continue
			}
			fedDocs = append(fedDocs, chromem.Document{
				ID:      ind.ID,
				Content: content,
				Metadata: map[string]string{"theme_id": ind.ThemeID, "type": "ksi"},
			})
		}
	}
	for _, rc := range frmr.Requirements {
		for _, req := range rc.Requirements {
			content := buildRequirementDocument(req)
			if content == "" {
				continue
			}
			fedDocs = append(fedDocs, chromem.Document{
				ID:      req.ID,
				Content: content,
				Metadata: map[string]string{"category": req.Category, "type": "requirement"},
			})
		}
	}
	if err := fedCol.AddDocuments(ctx, fedDocs, 0); err != nil {
		return nil, fmt.Errorf("index fedramp documents: %w", err)
	}

	return &vectorDB{db: db, cols: cols, rel: rel}, nil
}

type taggedHit struct {
	id       string
	similarity float32
	source   string
	docType  string // optional sub-type for disambiguation within a collection
}

func (v *vectorDB) search(ctx context.Context, query string, limit int, source string) ([]SearchResult, error) {
	var tagged []taggedHit

	if source == "" || source == "nist_800_53" {
		if col, ok := v.cols[collectionControls]; ok && col.Count() > 0 {
			hits, err := col.Query(ctx, query, limit, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("vector search controls: %w", err)
			}
			for _, h := range hits {
				tagged = append(tagged, taggedHit{id: h.ID, similarity: h.Similarity, source: "nist_800_53"})
			}
		}
	}

	if source == "" || source == "fisma_fy2025" {
		if col, ok := v.cols[collectionFismaMetrics]; ok && col.Count() > 0 {
			hits, err := col.Query(ctx, query, limit, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("vector search fisma metrics: %w", err)
			}
			for _, h := range hits {
				tagged = append(tagged, taggedHit{id: h.ID, similarity: h.Similarity, source: "fisma_fy2025"})
			}
		}
	}

	if source == "" || source == "nist_csf_v2" {
		if col, ok := v.cols[collectionCSF]; ok && col.Count() > 0 {
			hits, err := col.Query(ctx, query, limit, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("vector search csf subcategories: %w", err)
			}
			for _, h := range hits {
				tagged = append(tagged, taggedHit{id: h.ID, similarity: h.Similarity, source: "nist_csf_v2"})
			}
		}
	}

	if source == "" || source == "fedramp_20x" {
		if col, ok := v.cols[collectionFedRAMP]; ok && col.Count() > 0 {
			hits, err := col.Query(ctx, query, limit, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("vector search fedramp: %w", err)
			}
			for _, h := range hits {
				tagged = append(tagged, taggedHit{id: h.ID, similarity: h.Similarity, source: "fedramp_20x", docType: h.Metadata["type"]})
			}
		}
	}

	sort.Slice(tagged, func(i, j int) bool { return tagged[i].similarity > tagged[j].similarity })
	if len(tagged) > limit {
		tagged = tagged[:limit]
	}

	results := make([]SearchResult, 0, len(tagged))
	for _, h := range tagged {
		switch h.source {
		case "nist_800_53":
			c, err := v.rel.getControl(ctx, h.id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "nist_800_53",
				ID:        c.ID,
				Title:     c.ID + " " + c.Title,
				Body:      c.Statement,
				Relevance: h.similarity,
			})
		case "fisma_fy2025":
			id, _ := strconv.Atoi(h.id)
			m, err := v.rel.getFismaMetric(ctx, id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "fisma_fy2025",
				ID:        h.id,
				Title:     m.Domain,
				Body:      m.Question,
				Relevance: h.similarity,
			})
		case "nist_csf_v2":
			s, err := v.rel.getCSFSubcategory(ctx, h.id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "nist_csf_v2",
				ID:        s.ID,
				Title:     s.ID,
				Body:      s.Text,
				Relevance: h.similarity,
			})
		case "fedramp_20x":
			if h.docType == "ksi" {
				ind, err := v.rel.getKSI(ctx, h.id)
				if err != nil {
					continue
				}
				results = append(results, SearchResult{
					Source:    "fedramp_20x",
					ID:        ind.ID,
					Title:     ind.Name,
					Body:      ind.Statement,
					Relevance: h.similarity,
				})
			} else {
				var id, name, statement string
				row := v.rel.db.QueryRowContext(ctx,
					`SELECT id, name, statement FROM fedramp_requirements WHERE id = ?`, h.id)
				if err := row.Scan(&id, &name, &statement); err != nil {
					continue
				}
				results = append(results, SearchResult{
					Source:    "fedramp_20x",
					ID:        id,
					Title:     name,
					Body:      statement,
					Relevance: h.similarity,
				})
			}
		}
	}
	return results, nil
}

func embeddingFunc(cfg Config) (chromem.EmbeddingFunc, error) {
	switch cfg.EmbeddingProvider {
	case "openai":
		model := cfg.EmbeddingModel
		if model == "" {
			model = string(chromem.EmbeddingModelOpenAI3Small)
		}
		return chromem.NewEmbeddingFuncOpenAI(cfg.OpenAIKey, chromem.EmbeddingModelOpenAI(model)), nil
	case "ollama":
		model := cfg.EmbeddingModel
		if model == "" {
			model = "nomic-embed-text"
		}
		baseURL := cfg.OllamaBaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return chromem.NewEmbeddingFuncOllama(model, ollamaAPIBase(baseURL)), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q (use \"openai\" or \"ollama\")", cfg.EmbeddingProvider)
	}
}

// ollamaAPIBase ensures the URL ends with /api, which chromem-go requires.
func ollamaAPIBase(u string) string {
	u = strings.TrimRight(u, "/")
	if !strings.HasSuffix(u, "/api") {
		u += "/api"
	}
	return u
}

// effectiveEmbeddingModel returns the model that will actually be used given cfg,
// matching the defaults applied in embeddingFunc.
func effectiveEmbeddingModel(cfg Config) string {
	if cfg.EmbeddingModel != "" {
		return cfg.EmbeddingModel
	}
	switch cfg.EmbeddingProvider {
	case "openai":
		return string(chromem.EmbeddingModelOpenAI3Small)
	case "ollama":
		return "nomic-embed-text"
	}
	return ""
}

func buildControlDocument(c nist_800_53.Control) string {
	parts := []string{c.Title}
	if c.Statement != "" {
		parts = append(parts, c.Statement)
	}
	if c.Discussion != "" {
		parts = append(parts, c.Discussion)
	}
	return strings.Join(parts, "\n\n")
}

func buildKSIDocument(ind fedramp.KSIIndicator) string {
	if ind.Statement == "" {
		return ""
	}
	return ind.Name + "\n\n" + ind.Statement
}

func buildRequirementDocument(req fedramp.Requirement) string {
	if req.Statement == "" {
		return ""
	}
	return req.Name + "\n\n" + req.Statement
}

func buildSubcategoryDocument(s nist_csf.Subcategory) string {
	parts := []string{s.Text}
	for _, ex := range s.Examples {
		if ex != "" {
			parts = append(parts, ex)
		}
	}
	return strings.Join(parts, "\n\n")
}

func buildMetricDocument(m fisma.Metric) string {
	parts := []string{m.Question}
	for _, lvl := range m.MaturityLevels {
		if lvl.Description != "" {
			parts = append(parts, lvl.Level+": "+lvl.Description)
		}
	}
	return strings.Join(parts, "\n\n")
}
