//go:build embed_nomic

package nist

import (
	"encoding/json"
	_ "embed"
)

//go:embed data/nomic/chromem.db
var prebuiltVectorDB []byte

//go:embed data/nomic/chromem-meta.json
var prebuiltVectorMetaJSON []byte

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
