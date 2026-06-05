//go:build embed_openai_small

package nist

import (
	"encoding/json"
	_ "embed"
)

//go:embed data/openai-small/chromem.db
var prebuiltVectorDB []byte

//go:embed data/openai-small/chromem-meta.json
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
