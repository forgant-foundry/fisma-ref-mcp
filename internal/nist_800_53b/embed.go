package nist_800_53b

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/nist-800-53b.json
var baselineJSON []byte

// Baseline names returned by Load.
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

// Load parses the embedded SP 800-53B JSON and returns a map of
// normalized control ID → sorted list of baseline names the control belongs to.
func Load() (map[string][]string, error) {
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
		id := normalizeID(r.SourceElementIdentifier)
		out[id] = appendUnique(out[id], name)
	}
	return out, nil
}

// NormalizeBaseline accepts "low", "Low", "LOW", "moderate", "high", "privacy"
// and returns the canonical lowercase form, or "" if unrecognized.
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

// normalizeID converts zero-padded identifiers to display form, matching the
// nist_800_53 package convention. "AC-01" → "AC-1", "AC-02(01)" → "AC-2(1)".
func normalizeID(id string) string {
	id = strings.ToUpper(id)
	hyphen := strings.Index(id, "-")
	if hyphen < 0 {
		return id
	}
	family := id[:hyphen]
	rest := id[hyphen+1:]
	if paren := strings.Index(rest, "("); paren >= 0 {
		num := stripLeadingZeros(rest[:paren])
		enh := stripLeadingZeros(rest[paren+1 : len(rest)-1])
		return family + "-" + num + "(" + enh + ")"
	}
	return family + "-" + stripLeadingZeros(rest)
}

func stripLeadingZeros(s string) string {
	t := strings.TrimLeft(s, "0")
	if t == "" {
		return "0"
	}
	return t
}
