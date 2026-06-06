//go:build embed_openai_small

package vec_store

import _ "embed"

//go:embed data/openai-small/chromem.db
var prebuiltVectorDB []byte

//go:embed data/openai-small/chromem-meta.json
var prebuiltVectorMetaJSON []byte

// PrebuiltVector returns the serialised chromem-go DB and its metadata.
func PrebuiltVector() ([]byte, *VectorMeta, bool) {
	return decodePrebuilt(prebuiltVectorDB, prebuiltVectorMetaJSON)
}
