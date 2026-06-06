//go:build !embed_nomic && !embed_qwen3 && !embed_openai_small

package vec_store

// PrebuiltVector returns false when no vector index has been embedded.
// Builds without an explicit embed_* tag fall back to FTS5 search.
func PrebuiltVector() ([]byte, *VectorMeta, bool) { return nil, nil, false }
