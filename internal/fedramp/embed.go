package fedramp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/FRMR.documentation.json
var frmrJSON []byte

// Load parses the embedded FedRAMP machine-readable documentation and returns
// the full catalog: glossary terms, KSI themes with indicators, and process
// requirement categories.
func Load() (*Catalog, error) {
	var raw struct {
		Info struct {
			Version     string `json:"version"`
			LastUpdated string `json:"last_updated"`
		} `json:"info"`

		FRD struct {
			Data map[string]map[string]struct {
				Term         string   `json:"term"`
				Definition   string   `json:"definition"`
				Alts         []string `json:"alts"`
				Note         string   `json:"note"`
				Notes        []string `json:"notes"`
				Reference    string   `json:"reference"`
				ReferenceURL string   `json:"reference_url"`
			} `json:"data"`
		} `json:"FRD"`

		FRR map[string]struct {
			Info struct {
				Name        string `json:"name"`
				FrontMatter struct {
					Purpose string `json:"purpose"`
				} `json:"front_matter"`
			} `json:"info"`
			// version → label → req_id → requirement
			Data map[string]map[string]map[string]struct {
				Name         string   `json:"name"`
				Statement    string   `json:"statement"`
				Keyword      string   `json:"primary_key_word"`
				Affects      []string `json:"affects"`
				Terms        []string `json:"terms"`
				Reference    string   `json:"reference"`
				ReferenceURL string   `json:"reference_url"`
			} `json:"data"`
		} `json:"FRR"`

		KSI map[string]struct {
			ID        string `json:"id"`
			ShortName string `json:"short_name"`
			Name      string `json:"name"`
			Theme     string `json:"theme"`
			Indicators map[string]struct {
				Name         string   `json:"name"`
				Statement    string   `json:"statement"`
				Controls     []string `json:"controls"`
				Terms        []string `json:"terms"`
				Reference    string   `json:"reference"`
				ReferenceURL string   `json:"reference_url"`
			} `json:"indicators"`
		} `json:"KSI"`
	}

	if err := json.Unmarshal(frmrJSON, &raw); err != nil {
		return nil, fmt.Errorf("parse fedramp frmr: %w", err)
	}

	cat := &Catalog{
		Version:     raw.Info.Version,
		LastUpdated: raw.Info.LastUpdated,
	}

	// FRD — glossary terms
	for _, section := range raw.FRD.Data {
		for id, t := range section {
			note := t.Note
			if note == "" && len(t.Notes) > 0 {
				note = strings.Join(t.Notes, " ")
			}
			cat.Terms = append(cat.Terms, Term{
				ID:         id,
				Term:       t.Term,
				Definition: t.Definition,
				Alts:       t.Alts,
				Note:       note,
			})
		}
	}

	// KSI — themes and indicators
	for shortName, rawTheme := range raw.KSI {
		theme := KSITheme{
			ID:        rawTheme.ID,
			ShortName: shortName,
			Name:      rawTheme.Name,
			Theme:     rawTheme.Theme,
		}
		for id, ind := range rawTheme.Indicators {
			controls := make([]string, 0, len(ind.Controls))
			for _, c := range ind.Controls {
				controls = append(controls, normControlID(c))
			}
			theme.Indicators = append(theme.Indicators, KSIIndicator{
				ID:           id,
				ThemeID:      rawTheme.ID,
				Name:         ind.Name,
				Statement:    ind.Statement,
				Controls:     controls,
				Terms:        ind.Terms,
				Reference:    ind.Reference,
				ReferenceURL: ind.ReferenceURL,
			})
		}
		cat.KSIThemes = append(cat.KSIThemes, theme)
	}

	// FRR — process requirements (flattened from version/label nesting)
	for catID, rawCat := range raw.FRR {
		rc := RequirementCategory{
			ID:      catID,
			Name:    rawCat.Info.Name,
			Purpose: rawCat.Info.FrontMatter.Purpose,
		}
		for version, labels := range rawCat.Data {
			for _, reqs := range labels {
				for id, req := range reqs {
					rc.Requirements = append(rc.Requirements, Requirement{
						ID:        id,
						Category:  catID,
						Name:      req.Name,
						Statement: req.Statement,
						Keyword:   req.Keyword,
						Version:   version,
						Affects:   req.Affects,
						Terms:     req.Terms,
						Reference: req.Reference,
					})
				}
			}
		}
		cat.Requirements = append(cat.Requirements, rc)
	}

	return cat, nil
}

// normControlID converts FedRAMP dot notation to SP 800-53 paren notation.
// "ac-2.2" → "AC-2(2)", "ia-12" → "IA-12".
func normControlID(id string) string {
	id = strings.ToUpper(id)
	if dot := strings.LastIndex(id, "."); dot >= 0 {
		return id[:dot] + "(" + id[dot+1:] + ")"
	}
	return id
}
