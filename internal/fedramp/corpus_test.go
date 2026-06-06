package fedramp_test

import (
	"strings"
	"testing"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
)

// Published specification: FedRAMP Machine-Readable Documentation (FRMR) 0.9.43-beta
// 49 glossary terms, 11 KSI themes, 60 KSI indicators, 163 process requirements

func TestLoad_Counts(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(cat.Terms) != 49 {
		t.Errorf("got %d glossary terms, want 49", len(cat.Terms))
	}
	if len(cat.KSIThemes) != 11 {
		t.Errorf("got %d KSI themes, want 11", len(cat.KSIThemes))
	}

	var indicators int
	for _, th := range cat.KSIThemes {
		indicators += len(th.Indicators)
	}
	if indicators != 60 {
		t.Errorf("got %d KSI indicators, want 60", indicators)
	}

	var requirements int
	for _, rc := range cat.Requirements {
		requirements += len(rc.Requirements)
	}
	if requirements != 163 {
		t.Errorf("got %d process requirements, want 163", requirements)
	}
}

// ── Glossary terms ────────────────────────────────────────────────────────────

func TestLoad_TermIntegrity(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool, len(cat.Terms))
	for _, term := range cat.Terms {
		if term.ID == "" {
			t.Error("term has empty ID")
		}
		if term.Term == "" {
			t.Errorf("term %s has empty Term name", term.ID)
		}
		if term.Definition == "" {
			t.Errorf("term %s has empty Definition", term.ID)
		}
		if seen[term.ID] {
			t.Errorf("duplicate term ID %s", term.ID)
		}
		seen[term.ID] = true
		if !strings.HasPrefix(term.ID, "FRD-") {
			t.Errorf("term ID %q does not follow FRD- prefix convention", term.ID)
		}
	}
}

func TestLoad_KnownTerms(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]fedramp.Term, len(cat.Terms))
	for _, term := range cat.Terms {
		index[term.ID] = term
	}

	// Spot-check terms confirmed present in the FRMR source document.
	known := []string{"FRD-ACV"}
	for _, id := range known {
		t.Run(id, func(t *testing.T) {
			term, ok := index[id]
			if !ok {
				t.Fatalf("term %s not found", id)
			}
			if term.Definition == "" {
				t.Errorf("term %s has empty definition", id)
			}
		})
	}
}

// ── KSI themes and indicators ─────────────────────────────────────────────────

func TestLoad_KSIThemeIntegrity(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool, len(cat.KSIThemes))
	for _, th := range cat.KSIThemes {
		if th.ID == "" {
			t.Error("KSI theme has empty ID")
		}
		if th.Name == "" {
			t.Errorf("KSI theme %s has empty Name", th.ID)
		}
		if th.ShortName == "" {
			t.Errorf("KSI theme %s has empty ShortName", th.ID)
		}
		if len(th.Indicators) == 0 {
			t.Errorf("KSI theme %s has no indicators", th.ID)
		}
		if seen[th.ID] {
			t.Errorf("duplicate KSI theme ID %s", th.ID)
		}
		seen[th.ID] = true
	}
}

func TestLoad_KSIIndicatorIntegrity(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	themeSet := make(map[string]bool, len(cat.KSIThemes))
	for _, th := range cat.KSIThemes {
		themeSet[th.ID] = true
	}
	seen := make(map[string]bool)
	for _, th := range cat.KSIThemes {
		for _, ind := range th.Indicators {
			if ind.ID == "" {
				t.Error("KSI indicator has empty ID")
			}
			if ind.Name == "" {
				t.Errorf("KSI indicator %s has empty Name", ind.ID)
			}
			if ind.Statement == "" {
				// Some indicators in the beta FRMR document do not yet have statement
				// text. Log rather than fail so we notice if the count grows.
				t.Logf("WARNING: KSI indicator %s has empty Statement (beta document gap)", ind.ID)
			}
			if ind.ThemeID != th.ID {
				t.Errorf("indicator %s has ThemeID %q, expected %q", ind.ID, ind.ThemeID, th.ID)
			}
			if seen[ind.ID] {
				t.Errorf("duplicate KSI indicator ID %s", ind.ID)
			}
			seen[ind.ID] = true
			if !strings.HasPrefix(ind.ID, "KSI-") {
				t.Errorf("indicator ID %q does not follow KSI- prefix convention", ind.ID)
			}
		}
	}
}

func TestLoad_KSIControlReferences(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	// At least some indicators must have SP 800-53 control references.
	var withControls int
	for _, th := range cat.KSIThemes {
		for _, ind := range th.Indicators {
			if len(ind.Controls) > 0 {
				withControls++
			}
		}
	}
	if withControls == 0 {
		t.Error("no KSI indicators have SP 800-53 control references")
	}
}

func TestLoad_KSIKnownThemes(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	shortNames := make(map[string]bool, len(cat.KSIThemes))
	for _, th := range cat.KSIThemes {
		shortNames[th.ShortName] = true
	}
	// These theme short names are documented in the FRMR and referenced in the README.
	for _, name := range []string{"IAM", "MLA", "SVC", "CNA"} {
		if !shortNames[name] {
			t.Errorf("expected KSI theme short name %q not found", name)
		}
	}
}

// ── Process requirements ──────────────────────────────────────────────────────

func TestLoad_RequirementIntegrity(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	validKeywords := map[string]bool{"MUST": true, "SHOULD": true, "MAY": true, "MUST NOT": true, "SHOULD NOT": true}
	validVersions := map[string]bool{"rev5": true, "20x": true, "both": true}
	for _, rc := range cat.Requirements {
		for _, req := range rc.Requirements {
			if req.ID == "" {
				t.Error("requirement has empty ID")
			}
			if req.Statement == "" {
				// Some entries in the beta FRMR lack statement text. Log rather than
				// fail so we notice if the count grows on the next document update.
				t.Logf("WARNING: requirement %s has empty Statement (beta document gap)", req.ID)
			}
			if req.Keyword != "" && !validKeywords[req.Keyword] {
				t.Errorf("requirement %s has unexpected keyword %q", req.ID, req.Keyword)
			}
			if !validVersions[req.Version] {
				t.Errorf("requirement %s has unexpected version %q", req.ID, req.Version)
			}
			if seen[req.ID] {
				t.Errorf("duplicate requirement ID %s", req.ID)
			}
			seen[req.ID] = true
		}
	}
}

func TestLoad_RequirementVersionCoverage(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	versions := make(map[string]bool)
	for _, rc := range cat.Requirements {
		for _, req := range rc.Requirements {
			versions[req.Version] = true
		}
	}
	for _, v := range []string{"rev5", "20x", "both"} {
		if !versions[v] {
			t.Errorf("no requirements found with version %q", v)
		}
	}
}

func TestLoad_RequirementCategoryIntegrity(t *testing.T) {
	cat, err := fedramp.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, rc := range cat.Requirements {
		if rc.ID == "" {
			t.Error("requirement category has empty ID")
		}
		if rc.Name == "" {
			t.Errorf("requirement category %s has empty Name", rc.ID)
		}
		for _, req := range rc.Requirements {
			if req.Category != rc.ID {
				t.Errorf("requirement %s has Category %q, expected %q", req.ID, req.Category, rc.ID)
			}
		}
	}
}
