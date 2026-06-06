//go:build embed_qwen3

package vec_store

import _ "embed"

//go:embed data/qwen3/chromem.db
var prebuiltVectorDB []byte

//go:embed data/qwen3/chromem-meta.json
var prebuiltVectorMetaJSON []byte

// PrebuiltVector returns the serialised chromem-go DB and its metadata.
func PrebuiltVector() ([]byte, *VectorMeta, bool) {
	return decodePrebuilt(prebuiltVectorDB, prebuiltVectorMetaJSON)
}
