package nist_csf

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/csf-800-53-crosswalk.json
var crosswalkJSON []byte

// LoadCrosswalk parses the embedded CSF 2.0 → SP 800-53 Rev 5.2.0 crosswalk
// and returns a map of subcategory ID → sorted list of SP 800-53 control IDs.
func LoadCrosswalk() (map[string][]string, error) {
	var raw struct {
		Mappings map[string][]string `json:"mappings"`
	}
	if err := json.Unmarshal(crosswalkJSON, &raw); err != nil {
		return nil, fmt.Errorf("parse csf crosswalk: %w", err)
	}
	return raw.Mappings, nil
}
