package vec_store

import (
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
)

func BuildControlDocument(c nist_800_53.Control) string {
	parts := []string{c.Title}
	if c.Statement != "" {
		parts = append(parts, c.Statement)
	}
	if c.Discussion != "" {
		parts = append(parts, c.Discussion)
	}
	return strings.Join(parts, "\n\n")
}

func BuildMetricDocument(m fisma.Metric) string {
	parts := []string{m.Question}
	for _, lvl := range m.MaturityLevels {
		var lvlParts []string
		if lvl.Description != "" {
			lvlParts = append(lvlParts, lvl.Level+": "+lvl.Description)
		}
		if lvl.Evidence != "" {
			lvlParts = append(lvlParts, "Evidence: "+lvl.Evidence)
		}
		if lvl.AssessorNotes != "" {
			lvlParts = append(lvlParts, "Assessor notes: "+lvl.AssessorNotes)
		}
		if len(lvlParts) > 0 {
			parts = append(parts, strings.Join(lvlParts, "\n"))
		}
	}
	return strings.Join(parts, "\n\n")
}

func BuildSubcategoryDocument(s nist_csf.Subcategory) string {
	parts := []string{s.Text}
	for _, ex := range s.Examples {
		if ex != "" {
			parts = append(parts, ex)
		}
	}
	return strings.Join(parts, "\n\n")
}

func BuildKSIDocument(ind fedramp.KSIIndicator) string {
	if ind.Statement == "" {
		return ""
	}
	return ind.Name + "\n\n" + ind.Statement
}

func BuildRequirementDocument(req fedramp.Requirement) string {
	if req.Statement == "" {
		return ""
	}
	return req.Name + "\n\n" + req.Statement
}

func BuildTermDocument(t fedramp.Term) string {
	if t.Definition == "" {
		return ""
	}
	parts := []string{t.Term + ": " + t.Definition}
	if len(t.Alts) > 0 {
		parts = append(parts, "Also known as: "+strings.Join(t.Alts, ", "))
	}
	if t.Note != "" {
		parts = append(parts, t.Note)
	}
	return strings.Join(parts, "\n\n")
}
