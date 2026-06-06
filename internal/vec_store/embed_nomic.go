//go:build embed_nomic

package vec_store

import _ "embed"

//go:embed data/nomic/chromem.db
var prebuiltVectorDB []byte

//go:embed data/nomic/chromem-meta.json
var prebuiltVectorMetaJSON []byte

// PrebuiltVector returns the serialised chromem-go DB and its metadata.
func PrebuiltVector() ([]byte, *VectorMeta, bool) {
	return decodePrebuilt(prebuiltVectorDB, prebuiltVectorMetaJSON)
}
