//go:build no_embeddings

package nist

// PrebuiltVector always returns false in no_embeddings builds.
func PrebuiltVector() ([]byte, *VectorMeta, bool) { return nil, nil, false }
