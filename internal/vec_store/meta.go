package vec_store

import (
	"encoding/json"
	"time"
)

// VectorMeta records which provider and model were used to build the pre-built
// vector index so the runtime can validate that query embeddings are compatible.
type VectorMeta struct {
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	BuiltAt          time.Time `json:"built_at"`
	ControlCount     int       `json:"control_count"`
	MetricCount      int       `json:"metric_count"`
	SubcategoryCount int `json:"subcategory_count"`
}

func decodePrebuilt(db, metaJSON []byte) ([]byte, *VectorMeta, bool) {
	if len(db) == 0 {
		return nil, nil, false
	}
	var m VectorMeta
	if err := json.Unmarshal(metaJSON, &m); err != nil || m.Provider == "" {
		return nil, nil, false
	}
	return db, &m, true
}
