//go:build !no_embeddings

package store

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist"
	"github.com/philippgille/chromem-go"
)

const (
	collectionControls     = "controls"
	collectionFismaMetrics = "fisma_metrics"
)

type vectorDB struct {
	db   *chromem.DB
	cols map[string]*chromem.Collection
	rel  *relationalDB
}

func newVectorDB(ctx context.Context, cfg Config, controls []nist.Control, metrics []fisma.Metric, rel *relationalDB) (*vectorDB, error) {
	embFn, err := embeddingFunc(cfg)
	if err != nil {
		return nil, err
	}

	prebuiltData, meta, hasPrebuilt := nist.PrebuiltVector()
	if hasPrebuilt {
		return loadPrebuilt(ctx, prebuiltData, meta, cfg, embFn, rel)
	}
	return buildFromDocuments(ctx, controls, metrics, embFn, rel)
}

// loadPrebuilt imports the serialised chromem-go DB embedded at build time.
// It validates that the runtime embedding provider and model match those used
// during generation so that query vectors are in the same space as index vectors.
func loadPrebuilt(ctx context.Context, data []byte, meta *nist.VectorMeta, cfg Config, embFn chromem.EmbeddingFunc, rel *relationalDB) (*vectorDB, error) {
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
	// (e.g., fisma_metrics was added after the index was generated); in that case
	// the collection simply returns no results until the index is regenerated.
	cols := make(map[string]*chromem.Collection)
	for _, name := range []string{collectionControls, collectionFismaMetrics} {
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
func buildFromDocuments(ctx context.Context, controls []nist.Control, metrics []fisma.Metric, embFn chromem.EmbeddingFunc, rel *relationalDB) (*vectorDB, error) {
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

	return &vectorDB{db: db, cols: cols, rel: rel}, nil
}

type taggedHit struct {
	id         string
	similarity float32
	source     string
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

func buildControlDocument(c nist.Control) string {
	parts := []string{c.Title}
	if c.Statement != "" {
		parts = append(parts, c.Statement)
	}
	if c.Discussion != "" {
		parts = append(parts, c.Discussion)
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
