//go:build no_embeddings

package rel_store

import (
	"context"
	"fmt"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
)

type vectorDB struct{}

func newVectorDB(_ context.Context, _ Config, _ []nist_800_53.Control, _ []fisma.Metric, _ []nist_csf.Subcategory, _ *relationalDB) (*vectorDB, error) {
	return nil, fmt.Errorf(
		"this binary was built without embedding support (no_embeddings tag); " +
			"use 'make embed-nomic', 'make embed-qwen3', or 'make embed-openai-small' " +
			"to build a version with vector search, or remove the embedding provider configuration",
	)
}

func (v *vectorDB) search(_ context.Context, _ string, _ int, _ string) ([]SearchResult, error) {
	return nil, fmt.Errorf("vector search not available in this build")
}
