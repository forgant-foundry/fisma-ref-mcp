//go:build !no_embeddings

package store

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist"
	"github.com/philippgille/chromem-go"
)

const collectionName = "controls"

type vectorDB struct {
	col *chromem.Collection
	rel *relationalDB
}

func newVectorDB(ctx context.Context, cfg Config, controls []nist.Control, rel *relationalDB) (*vectorDB, error) {
	embFn, err := embeddingFunc(cfg)
	if err != nil {
		return nil, err
	}

	prebuiltData, meta, hasPrebuilt := nist.PrebuiltVector()
	if hasPrebuilt {
		return loadPrebuilt(ctx, prebuiltData, meta, cfg, embFn, rel)
	}
	return buildFromControls(ctx, controls, embFn, rel)
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

	// Associate the embedding function with the restored collection so that
	// query text is embedded using the same model as the stored vectors.
	col, err := db.GetOrCreateCollection(collectionName, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("attach embedding function to collection: %w", err)
	}

	return &vectorDB{col: col, rel: rel}, nil
}

// buildFromControls generates embeddings at startup. This is used when no
// pre-built index is embedded in the binary.
func buildFromControls(ctx context.Context, controls []nist.Control, embFn chromem.EmbeddingFunc, rel *relationalDB) (*vectorDB, error) {
	db := chromem.NewDB()
	col, err := db.GetOrCreateCollection(collectionName, nil, embFn)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}

	docs := make([]chromem.Document, 0, len(controls))
	for _, c := range controls {
		content := buildDocument(c)
		if content == "" {
			continue
		}
		docs = append(docs, chromem.Document{
			ID:      strings.ToUpper(c.ID),
			Content: content,
			Metadata: map[string]string{
				"family":         c.FamilyID,
				"is_enhancement": fmt.Sprintf("%v", c.IsEnhancement),
			},
		})
	}

	if err := col.AddDocuments(ctx, docs, 0); err != nil {
		return nil, fmt.Errorf("index controls: %w", err)
	}

	return &vectorDB{col: col, rel: rel}, nil
}

func (v *vectorDB) search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	hits, err := v.col.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	results := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		c, err := v.rel.getControl(ctx, h.ID)
		if err != nil {
			continue
		}
		results = append(results, SearchResult{Control: *c, Relevance: h.Similarity})
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

func buildDocument(c nist.Control) string {
	parts := []string{c.Title}
	if c.Statement != "" {
		parts = append(parts, c.Statement)
	}
	if c.Discussion != "" {
		parts = append(parts, c.Discussion)
	}
	return strings.Join(parts, "\n\n")
}
