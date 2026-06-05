package store

import (
	"context"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist"
)

// SearchResult pairs a control with its relevance score (0–1, higher is better).
type SearchResult struct {
	Control   nist.Control
	Relevance float32
}

// Store provides deterministic and semantic access to NIST SP 800-53 controls.
type Store struct {
	rel *relationalDB
	vec *vectorDB // nil when no embedding provider is configured
}

// Config holds embedding provider settings. Leave EmbeddingProvider empty to
// disable vector search (relational lookups remain fully functional).
type Config struct {
	EmbeddingProvider string // "openai" | "ollama" | "" (disabled)
	EmbeddingModel    string
	OpenAIKey         string
	OllamaBaseURL     string
}

// New loads the embedded NIST catalog, populates the in-memory relational DB,
// and optionally builds the chromem-go vector index.
func New(ctx context.Context, cfg Config) (*Store, error) {
	// Infer provider and model from the pre-built index when not explicitly set.
	// Each binary variant embeds exactly one index, so the meta is authoritative.
	if _, meta, ok := nist.PrebuiltVector(); ok {
		if cfg.EmbeddingProvider == "" {
			cfg.EmbeddingProvider = meta.Provider
		}
		if cfg.EmbeddingModel == "" && meta.Provider == cfg.EmbeddingProvider {
			cfg.EmbeddingModel = meta.Model
		}
	}

	families, controls, err := nist.Load()
	if err != nil {
		return nil, err
	}

	rel, err := newRelationalDB(families, controls)
	if err != nil {
		return nil, err
	}

	var vec *vectorDB
	if cfg.EmbeddingProvider != "" {
		vec, err = newVectorDB(ctx, cfg, controls, rel)
		if err != nil {
			return nil, err
		}
	}

	return &Store{rel: rel, vec: vec}, nil
}

// Close releases underlying resources.
func (s *Store) Close() error {
	return s.rel.close()
}

// GetControl returns a single control by its canonical ID (e.g., "AC-1" or "ac-1").
func (s *Store) GetControl(ctx context.Context, id string) (*nist.Control, error) {
	return s.rel.getControl(ctx, id)
}

// ListFamilies returns all control families.
func (s *Store) ListFamilies(ctx context.Context) ([]nist.Family, error) {
	return s.rel.listFamilies(ctx)
}

// GetFamily returns all controls (base controls only, not enhancements) for a family.
func (s *Store) GetFamily(ctx context.Context, familyID string) ([]nist.Control, error) {
	return s.rel.getFamily(ctx, familyID)
}

// SearchControls performs semantic vector search when an embedding provider is
// configured, or FTS5 full-text search otherwise.
func (s *Store) SearchControls(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if s.vec != nil {
		return s.vec.search(ctx, query, limit)
	}
	return s.rel.search(ctx, query, limit)
}
