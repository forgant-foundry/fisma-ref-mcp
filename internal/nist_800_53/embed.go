package nist_800_53

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	_ "embed"
)

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)

//go:embed data/nist-800-53r5.json
var catalogJSON []byte

// Load parses the embedded NIST SP 800-53 catalog and returns all families
// and a flat list of controls (including enhancements).
func Load() ([]Family, []Control, error) {
	var raw rawCatalog
	if err := json.Unmarshal(catalogJSON, &raw); err != nil {
		return nil, nil, fmt.Errorf("parse catalog: %w", err)
	}

	byID := make(map[string]*rawElement, len(raw.Response.Elements.Elements))
	for i := range raw.Response.Elements.Elements {
		e := &raw.Response.Elements.Elements[i]
		byID[e.ElementIdentifier] = e
	}

	projFrom := make(map[string][]string)
	projTo := make(map[string][]string)
	for _, r := range raw.Response.Elements.Relationships {
		if r.RelationshipIdentifier != "projection" {
			continue
		}
		projFrom[r.SourceElementIdentifier] = append(projFrom[r.SourceElementIdentifier], r.DestElementIdentifier)
		projTo[r.DestElementIdentifier] = append(projTo[r.DestElementIdentifier], r.SourceElementIdentifier)
	}

	var families []Family
	var controls []Control

	for _, e := range raw.Response.Elements.Elements {
		if e.ElementType != "family" {
			continue
		}
		familyID := e.ElementIdentifier
		families = append(families, Family{
			ID:    familyID,
			Title: titleCase(e.Title),
		})

		for _, destID := range projFrom[familyID] {
			dest := byID[destID]
			if dest == nil {
				continue
			}
			if dest.ElementType == "control" {
				c := buildControl(dest, familyID, "", byID, projFrom)
				controls = append(controls, c)
				for _, enhID := range projFrom[destID] {
					enh := byID[enhID]
					if enh == nil || enh.ElementType != "control_enhancement" {
						continue
					}
					controls = append(controls, buildControl(enh, familyID, normalizeID(destID), byID, projFrom))
				}
			}
		}
	}

	sort.Slice(families, func(i, j int) bool { return families[i].ID < families[j].ID })
	sort.Slice(controls, func(i, j int) bool { return controls[i].ID < controls[j].ID })

	return families, controls, nil
}

func buildControl(e *rawElement, familyID, parentID string, byID map[string]*rawElement, projFrom map[string][]string) Control {
	id := normalizeID(e.ElementIdentifier)
	return Control{
		ID:            id,
		Title:         titleCase(e.Title),
		FamilyID:      familyID,
		Statement:     collectStatement(e.ElementIdentifier, byID, projFrom),
		Discussion:    collectDiscussion(e.ElementIdentifier, byID, projFrom),
		IsEnhancement: parentID != "",
		ParentID:      parentID,
	}
}

func collectStatement(controlID string, byID map[string]*rawElement, projFrom map[string][]string) string {
	var cstID string
	for _, destID := range projFrom[controlID] {
		if strings.HasPrefix(destID, "CST-") {
			cstID = destID
			break
		}
	}
	if cstID == "" {
		return ""
	}

	root := byID[cstID]
	var parts []string
	if root != nil && root.Text != "" {
		parts = append(parts, root.Text)
	}

	children := projFrom[cstID]
	sort.Strings(children)
	for _, childID := range children {
		child := byID[childID]
		if child == nil || child.ElementType != "control_statement" {
			continue
		}
		text := child.Text
		if child.Title != "" && text != "" {
			text = child.Title + ". " + text
		}
		if text != "" {
			parts = append(parts, text)
		}
	}

	return stripHTML(strings.Join(parts, " "))
}

func collectDiscussion(controlID string, byID map[string]*rawElement, projFrom map[string][]string) string {
	discID := "D-" + controlID
	if disc := byID[discID]; disc != nil {
		return stripHTML(disc.Text)
	}
	return ""
}

// NormalizeID converts zero-padded NIST identifiers to display form.
// "AC-01" → "AC-1", "AC-02(01)" → "AC-2(1)".
func NormalizeID(id string) string {
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

func normalizeID(id string) string { return NormalizeID(id) }

func stripLeadingZeros(s string) string {
	t := strings.TrimLeft(s, "0")
	if t == "" {
		return "0"
	}
	return t
}

func stripHTML(s string) string {
	s = htmlTagRE.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func titleCase(s string) string {
	if s != strings.ToUpper(s) || !strings.ContainsAny(s, " ") {
		return s
	}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
