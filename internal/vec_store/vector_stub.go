//go:build !embed_nomic && !embed_qwen3 && !embed_openai_small

package vec_store

import (
	"context"
	"fmt"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
)

// EmbedConfig holds the embedding provider settings.
type EmbedConfig struct {
	Provider  string
	Model     string
	OpenAIKey string
	OllamaURL string
}

// RawHit is a single vector search result before relational resolution.
type RawHit struct {
	ID         string
	Similarity float32
	Source     string
	DocType    string
}

// VectorDB is a no-op stub in builds without an embedded vector index.
type VectorDB struct{}

// NewVectorDB always returns an error in stub builds.
func NewVectorDB(_ context.Context, _ EmbedConfig, _ []nist_800_53.Control, _ []fisma.Metric, _ []nist_csf.Subcategory, _ *fedramp.Catalog) (*VectorDB, error) {
	return nil, fmt.Errorf(
		"this binary was built without an embedded vector index; " +
			"use make build-nomic, build-qwen3, or build-openai-small for vector search",
	)
}

// Query always returns an error in stub builds.
func (v *VectorDB) Query(_ context.Context, _ string, _ int, _ string) ([]RawHit, error) {
	return nil, fmt.Errorf("vector search not available in this build")
}
