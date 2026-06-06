package nist_csf_test

import (
	"strings"
	"testing"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
)

// Published specification: NIST Cybersecurity Framework 2.0
// 6 functions, 45 categories, 185 subcategories

var expectedFunctions = []struct {
	id    string
	title string
}{
	{"GV", "GOVERN"},
	{"ID", "IDENTIFY"},
	{"PR", "PROTECT"},
	{"DE", "DETECT"},
	{"RS", "RESPOND"},
	{"RC", "RECOVER"},
}

func TestLoad_FunctionCount(t *testing.T) {
	fns, _, _, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(fns) != 6 {
		t.Errorf("got %d functions, want 6", len(fns))
	}
}

func TestLoad_FunctionIDs(t *testing.T) {
	fns, _, _, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]nist_csf.Function, len(fns))
	for _, f := range fns {
		index[f.ID] = f
	}
	for _, want := range expectedFunctions {
		f, ok := index[want.id]
		if !ok {
			t.Errorf("function %s not found", want.id)
			continue
		}
		if !strings.EqualFold(f.Title, want.title) {
			t.Errorf("function %s title = %q, want %q", want.id, f.Title, want.title)
		}
		if f.Text == "" {
			t.Errorf("function %s has empty Text", want.id)
		}
	}
}

func TestLoad_CategoryCount(t *testing.T) {
	_, cats, _, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 34 {
		t.Errorf("got %d categories, want 34", len(cats))
	}
}

func TestLoad_CategoryIntegrity(t *testing.T) {
	fns, cats, _, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	fnSet := make(map[string]bool, len(fns))
	for _, f := range fns {
		fnSet[f.ID] = true
	}
	for _, c := range cats {
		if c.ID == "" {
			t.Error("category has empty ID")
		}
		if c.Title == "" {
			t.Errorf("category %s has empty Title", c.ID)
		}
		if !fnSet[c.FunctionID] {
			t.Errorf("category %s has FunctionID %q which is not in the functions list", c.ID, c.FunctionID)
		}
		if !strings.HasPrefix(c.ID, c.FunctionID+".") {
			t.Errorf("category %s ID does not start with its FunctionID %q", c.ID, c.FunctionID)
		}
	}
}

func TestLoad_SubcategoryCount(t *testing.T) {
	_, _, subs, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 185 {
		t.Errorf("got %d subcategories, want 185", len(subs))
	}
}

func TestLoad_SubcategoryIntegrity(t *testing.T) {
	_, cats, subs, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	catSet := make(map[string]bool, len(cats))
	for _, c := range cats {
		catSet[c.ID] = true
	}
	for _, s := range subs {
		if s.ID == "" {
			t.Error("subcategory has empty ID")
		}
		if s.Text == "" {
			t.Errorf("subcategory %s has empty Text", s.ID)
		}
		if !catSet[s.CategoryID] {
			t.Errorf("subcategory %s has CategoryID %q which is not in the categories list", s.ID, s.CategoryID)
		}
		if !strings.HasPrefix(s.ID, s.FunctionID+".") {
			t.Errorf("subcategory %s ID does not start with its FunctionID %q", s.ID, s.FunctionID)
		}
	}
}

func TestLoad_KnownSubcategories(t *testing.T) {
	_, _, subs, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]nist_csf.Subcategory, len(subs))
	for _, s := range subs {
		index[s.ID] = s
	}

	// Spot-check well-known subcategories from the published CSF 2.0 document.
	tests := []struct {
		id         string
		functionID string
		categoryID string
	}{
		{"GV.OC-01", "GV", "GV.OC"},
		{"GV.OC-02", "GV", "GV.OC"},
		{"ID.AM-01", "ID", "ID.AM"},
		{"PR.AA-01", "PR", "PR.AA"},
		{"PR.AA-03", "PR", "PR.AA"},
		{"DE.CM-01", "DE", "DE.CM"},
		{"RS.MA-01", "RS", "RS.MA"},
		{"RC.RP-01", "RC", "RC.RP"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			s, ok := index[tt.id]
			if !ok {
				t.Fatalf("subcategory %s not found", tt.id)
			}
			if s.FunctionID != tt.functionID {
				t.Errorf("FunctionID = %q, want %q", s.FunctionID, tt.functionID)
			}
			if s.CategoryID != tt.categoryID {
				t.Errorf("CategoryID = %q, want %q", s.CategoryID, tt.categoryID)
			}
			if s.Text == "" {
				t.Error("Text is empty")
			}
		})
	}
}

func TestLoad_SubcategoriesHaveExamples(t *testing.T) {
	_, _, subs, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	var withExamples int
	for _, s := range subs {
		if len(s.Examples) > 0 {
			withExamples++
		}
	}
	// The CSF 2.0 source provides examples for a meaningful portion of subcategories
	// (not all — some are context-dependent). Verify at least 50% have examples.
	pct := float64(withExamples) / float64(len(subs)) * 100
	if pct < 50 {
		t.Errorf("only %.0f%% of subcategories have examples; expected >50%%", pct)
	}
}

// ── Crosswalk ─────────────────────────────────────────────────────────────────

func TestLoadCrosswalk_NotEmpty(t *testing.T) {
	cw, err := nist_csf.LoadCrosswalk()
	if err != nil {
		t.Fatal(err)
	}
	if len(cw) == 0 {
		t.Fatal("crosswalk is empty")
	}
}

func TestLoadCrosswalk_SubcategoriesExist(t *testing.T) {
	_, _, subs, err := nist_csf.Load()
	if err != nil {
		t.Fatal(err)
	}
	cw, err := nist_csf.LoadCrosswalk()
	if err != nil {
		t.Fatal(err)
	}
	subSet := make(map[string]bool, len(subs))
	for _, s := range subs {
		subSet[s.ID] = true
	}
	for subID := range cw {
		if !subSet[subID] {
			t.Errorf("crosswalk references subcategory %q which does not exist", subID)
		}
	}
}

func TestLoadCrosswalk_ControlIDFormat(t *testing.T) {
	cw, err := nist_csf.LoadCrosswalk()
	if err != nil {
		t.Fatal(err)
	}
	// All control IDs in the crosswalk should look like normalized SP 800-53 IDs.
	for subID, controls := range cw {
		for _, ctrlID := range controls {
			if ctrlID == "" {
				t.Errorf("crosswalk entry for %s contains empty control ID", subID)
			}
			if !strings.Contains(ctrlID, "-") {
				t.Errorf("crosswalk control ID %q does not look like a SP 800-53 ID", ctrlID)
			}
		}
	}
}
