package nist_csf

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/nist-csf-2.0.json
var csfJSON []byte

// Load parses the embedded NIST CSF 2.0 JSON and returns the structured
// functions, categories, and subcategories (with implementation examples).
func Load() (functions []Function, categories []Category, subcategories []Subcategory, err error) {
	var raw struct {
		Response struct {
			Elements struct {
				Elements []struct {
					ElementIdentifier string `json:"element_identifier"`
					ElementType       string `json:"element_type"`
					Text              string `json:"text"`
					Title             string `json:"title"`
				} `json:"elements"`
				Relationships []struct {
					SourceElementIdentifier string `json:"source_element_identifier"`
					DestElementIdentifier   string `json:"dest_element_identifier"`
					RelationshipIdentifier  string `json:"relationship_identifier"`
				} `json:"relationships"`
			} `json:"elements"`
		} `json:"response"`
	}

	if err = json.Unmarshal(csfJSON, &raw); err != nil {
		return nil, nil, nil, fmt.Errorf("parse csf json: %w", err)
	}

	elems := raw.Response.Elements

	byID := make(map[string]struct {
		text  string
		title string
		etype string
	}, len(elems.Elements))
	for _, e := range elems.Elements {
		byID[e.ElementIdentifier] = struct {
			text  string
			title string
			etype string
		}{e.Text, e.Title, e.ElementType}
	}

	children := make(map[string][]string)
	for _, r := range elems.Relationships {
		if r.RelationshipIdentifier == "projection" &&
			r.SourceElementIdentifier != r.DestElementIdentifier {
			src := r.SourceElementIdentifier
			dst := r.DestElementIdentifier
			if strings.HasPrefix(dst, src+".") || strings.HasPrefix(dst, src+"-") {
				children[src] = append(children[src], dst)
			}
		}
	}

	for _, e := range elems.Elements {
		if e.ElementType != "function" {
			continue
		}
		functions = append(functions, Function{
			ID:    e.ElementIdentifier,
			Title: e.Title,
			Text:  e.Text,
		})
	}

	// The JSON contains duplicate entries for some categories; keep the one with more text.
	catSeen := make(map[string]int)
	for _, e := range elems.Elements {
		if e.ElementType != "category" {
			continue
		}
		parts := strings.SplitN(e.ElementIdentifier, ".", 2)
		c := Category{
			ID:         e.ElementIdentifier,
			FunctionID: parts[0],
			Title:      e.Title,
			Text:       e.Text,
		}
		if idx, seen := catSeen[c.ID]; seen {
			if len(c.Text) > len(categories[idx].Text) {
				categories[idx] = c
			}
		} else {
			catSeen[c.ID] = len(categories)
			categories = append(categories, c)
		}
	}

	// The JSON contains duplicate entries for some subcategories; keep the one with more text.
	subSeen := make(map[string]int)
	for _, e := range elems.Elements {
		if e.ElementType != "subcategory" {
			continue
		}
		hyphen := strings.LastIndex(e.ElementIdentifier, "-")
		if hyphen < 0 {
			continue
		}
		catID := e.ElementIdentifier[:hyphen]
		dot := strings.Index(catID, ".")
		if dot < 0 {
			continue
		}
		fnID := catID[:dot]

		var examples []string
		for _, childID := range children[e.ElementIdentifier] {
			if c, ok := byID[childID]; ok && c.etype == "implementation_example" {
				examples = append(examples, c.text)
			}
		}

		s := Subcategory{
			ID:         e.ElementIdentifier,
			CategoryID: catID,
			FunctionID: fnID,
			Text:       e.Text,
			Examples:   examples,
		}
		if idx, seen := subSeen[s.ID]; seen {
			if len(s.Text) > len(subcategories[idx].Text) {
				subcategories[idx] = s
			}
		} else {
			subSeen[s.ID] = len(subcategories)
			subcategories = append(subcategories, s)
		}
	}

	return functions, categories, subcategories, nil
}
