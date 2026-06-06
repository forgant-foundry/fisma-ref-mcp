package nist_800_53

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/nist-800-53b.json
var baselineJSON []byte

// Baseline names returned by LoadBaselines.
const (
	BaselineLow      = "low"
	BaselineModerate = "moderate"
	BaselineHigh     = "high"
	BaselinePrivacy  = "privacy"
)

var baselineIDs = map[string]string{
	"SB-Low":      BaselineLow,
	"SB-Moderate": BaselineModerate,
	"SB-High":     BaselineHigh,
	"PB-Yes":      BaselinePrivacy,
}

// LoadBaselines parses the embedded SP 800-53B JSON and returns a map of
// normalized control ID → list of baseline names the control belongs to.
func LoadBaselines() (map[string][]string, error) {
	var raw struct {
		Response struct {
			Elements struct {
				Relationships []struct {
					SourceElementIdentifier string `json:"source_element_identifier"`
					DestElementIdentifier   string `json:"dest_element_identifier"`
					RelationshipIdentifier  string `json:"relationship_identifier"`
				} `json:"relationships"`
			} `json:"elements"`
		} `json:"response"`
	}

	if err := json.Unmarshal(baselineJSON, &raw); err != nil {
		return nil, fmt.Errorf("parse 800-53b json: %w", err)
	}

	out := make(map[string][]string)
	for _, r := range raw.Response.Elements.Relationships {
		if r.RelationshipIdentifier != "projection" {
			continue
		}
		name, ok := baselineIDs[r.DestElementIdentifier]
		if !ok {
			continue
		}
		id := NormalizeID(r.SourceElementIdentifier)
		out[id] = appendUnique(out[id], name)
	}
	return out, nil
}

// NormalizeBaseline accepts "low", "moderate", "high", or "privacy" in any
// case and returns the canonical lowercase form, or "" if unrecognized.
func NormalizeBaseline(s string) string {
	switch strings.ToLower(s) {
	case "low":
		return BaselineLow
	case "moderate":
		return BaselineModerate
	case "high":
		return BaselineHigh
	case "privacy":
		return BaselinePrivacy
	}
	return ""
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
