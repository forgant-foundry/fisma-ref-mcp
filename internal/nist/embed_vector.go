//go:build !no_embeddings && !embed_nomic && !embed_openai_small && !embed_qwen3

package nist

import (
	"encoding/json"
	_ "embed"
)

//go:embed data/chromem.db
var prebuiltVectorDB []byte

//go:embed data/chromem-meta.json
var prebuiltVectorMetaJSON []byte

// PrebuiltVector returns the serialised chromem-go DB and its metadata.
// The bool is false when no pre-built index has been generated yet.
func PrebuiltVector() ([]byte, *VectorMeta, bool) {
	if len(prebuiltVectorDB) == 0 {
		return nil, nil, false
	}
	var m VectorMeta
	if err := json.Unmarshal(prebuiltVectorMetaJSON, &m); err != nil || m.Provider == "" {
		return nil, nil, false
	}
	return prebuiltVectorDB, &m, true
}
