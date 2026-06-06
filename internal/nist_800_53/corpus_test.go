package nist_800_53_test

import (
	"strings"
	"testing"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
)

// Published specification: SP 800-53 Rev 5.2.0
// 20 families, 324 base controls, 872 enhancements = 1,196 total
// SP 800-53B: Low 149, Moderate 287, High 370, Privacy 96

var expectedFamilies = []string{
	"AC", "AT", "AU", "CA", "CM", "CP", "IA", "IR", "MA", "MP",
	"PE", "PL", "PM", "PS", "PT", "RA", "SA", "SC", "SI", "SR",
}

func TestLoad_FamilyCount(t *testing.T) {
	families, _, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 20 {
		t.Errorf("got %d families, want 20", len(families))
	}
}

func TestLoad_FamilyIDs(t *testing.T) {
	families, _, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]bool, len(families))
	for _, f := range families {
		got[f.ID] = true
		if f.Title == "" {
			t.Errorf("family %s has empty title", f.ID)
		}
	}
	for _, id := range expectedFamilies {
		if !got[id] {
			t.Errorf("family %s missing from Load() result", id)
		}
	}
}

func TestLoad_ControlCounts(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(controls) != 1196 {
		t.Errorf("got %d controls, want 1196", len(controls))
	}
	var base, enhancements int
	for _, c := range controls {
		if c.IsEnhancement {
			enhancements++
		} else {
			base++
		}
	}
	if base != 324 {
		t.Errorf("got %d base controls, want 324", base)
	}
	if enhancements != 872 {
		t.Errorf("got %d enhancements, want 872", enhancements)
	}
}

func TestLoad_ControlContentIntegrity(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range controls {
		if c.ID == "" {
			t.Error("control has empty ID")
		}
		if c.Title == "" {
			t.Errorf("control %s has empty title", c.ID)
		}
		if c.FamilyID == "" {
			t.Errorf("control %s has empty FamilyID", c.ID)
		}
		if c.IsEnhancement && c.ParentID == "" {
			t.Errorf("enhancement %s has empty ParentID", c.ID)
		}
	}
}

func TestLoad_KnownControls(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]nist_800_53.Control, len(controls))
	for _, c := range controls {
		index[c.ID] = c
	}

	tests := []struct {
		id       string
		title    string
		familyID string
		isEnh    bool
		parentID string
	}{
		// Titles are stored verbatim from the OSCAL source, which uses full Title Case
		// (all words capitalised, including prepositions and articles).
		{"AC-1", "Policy And Procedures", "AC", false, ""},
		{"AC-2", "Account Management", "AC", false, ""},
		{"AC-2(1)", "Automated System Account Management", "AC", true, "AC-2"},
		{"IA-5", "Authenticator Management", "IA", false, ""},
		{"SI-3", "Malicious Code Protection", "SI", false, ""},
		{"SC-28", "Protection Of Information At Rest", "SC", false, ""},
		{"CM-6", "Configuration Settings", "CM", false, ""},
		{"AU-2", "Event Logging", "AU", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			c, ok := index[tt.id]
			if !ok {
				t.Fatalf("control %s not found", tt.id)
			}
			if c.Title != tt.title {
				t.Errorf("title = %q, want %q", c.Title, tt.title)
			}
			if c.FamilyID != tt.familyID {
				t.Errorf("FamilyID = %q, want %q", c.FamilyID, tt.familyID)
			}
			if c.IsEnhancement != tt.isEnh {
				t.Errorf("IsEnhancement = %v, want %v", c.IsEnhancement, tt.isEnh)
			}
			if c.ParentID != tt.parentID {
				t.Errorf("ParentID = %q, want %q", c.ParentID, tt.parentID)
			}
			if c.Statement == "" {
				t.Errorf("Statement is empty")
			}
		})
	}
}

func TestLoad_EnhancementParentExists(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]bool, len(controls))
	for _, c := range controls {
		index[c.ID] = true
	}
	for _, c := range controls {
		if c.IsEnhancement && !index[c.ParentID] {
			t.Errorf("enhancement %s references parent %s which does not exist", c.ID, c.ParentID)
		}
	}
}

func TestLoad_FamilyMembership(t *testing.T) {
	families, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	familySet := make(map[string]bool, len(families))
	for _, f := range families {
		familySet[f.ID] = true
	}
	for _, c := range controls {
		if !familySet[c.FamilyID] {
			t.Errorf("control %s has FamilyID %q which is not in the families list", c.ID, c.FamilyID)
		}
	}
}

// ── SP 800-53B baselines ──────────────────────────────────────────────────────

func TestLoadBaselines_Counts(t *testing.T) {
	baselines, err := nist_800_53.LoadBaselines()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{"low": 0, "moderate": 0, "high": 0, "privacy": 0}
	for _, bls := range baselines {
		for _, bl := range bls {
			counts[bl]++
		}
	}
	want := map[string]int{
		"low":      149,
		"moderate": 287,
		"high":     370,
		"privacy":  96,
	}
	for name, wantN := range want {
		if counts[name] != wantN {
			t.Errorf("baseline %q: got %d controls, want %d", name, counts[name], wantN)
		}
	}
}

func TestLoadBaselines_SubsetRelation(t *testing.T) {
	// Every Low control must appear in Moderate; every Moderate must appear in High.
	baselines, err := nist_800_53.LoadBaselines()
	if err != nil {
		t.Fatal(err)
	}
	inBaseline := func(bl string) map[string]bool {
		s := make(map[string]bool)
		for id, bls := range baselines {
			for _, b := range bls {
				if b == bl {
					s[id] = true
				}
			}
		}
		return s
	}
	low := inBaseline("low")
	moderate := inBaseline("moderate")
	high := inBaseline("high")

	for id := range low {
		if !moderate[id] {
			t.Errorf("Low control %s not present in Moderate baseline", id)
		}
	}
	for id := range moderate {
		if !high[id] {
			t.Errorf("Moderate control %s not present in High baseline", id)
		}
	}
}

func TestLoadBaselines_AllControlsExist(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	baselines, err := nist_800_53.LoadBaselines()
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]bool, len(controls))
	for _, c := range controls {
		index[c.ID] = true
	}
	for id := range baselines {
		if !index[id] {
			t.Errorf("baseline references control %s which does not exist in the catalog", id)
		}
	}
}

// ── NormalizeID ───────────────────────────────────────────────────────────────

func TestNormalizeID(t *testing.T) {
	tests := []struct{ input, want string }{
		{"AC-1", "AC-1"},
		{"ac-1", "AC-1"},
		{"AC-01", "AC-1"},
		{"AC-2(1)", "AC-2(1)"},
		{"ac-2(1)", "AC-2(1)"},
		{"AC-02(01)", "AC-2(1)"},
		{"SI-3", "SI-3"},
		{"si-03", "SI-3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := nist_800_53.NormalizeID(tt.input); got != tt.want {
				t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeID_RoundTrip(t *testing.T) {
	_, controls, err := nist_800_53.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range controls {
		if got := nist_800_53.NormalizeID(c.ID); got != c.ID {
			t.Errorf("NormalizeID(%q) = %q; stored IDs should already be normalized", c.ID, got)
		}
		lower := strings.ToLower(c.ID)
		if got := nist_800_53.NormalizeID(lower); got != c.ID {
			t.Errorf("NormalizeID(%q) = %q, want %q", lower, got, c.ID)
		}
	}
}

// ── NormalizeBaseline ─────────────────────────────────────────────────────────

func TestNormalizeBaseline(t *testing.T) {
	tests := []struct{ input, want string }{
		{"low", "low"},
		{"Low", "low"},
		{"LOW", "low"},
		{"moderate", "moderate"},
		{"Moderate", "moderate"},
		{"high", "high"},
		{"HIGH", "high"},
		{"privacy", "privacy"},
		{"PRIVACY", "privacy"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := nist_800_53.NormalizeBaseline(tt.input); got != tt.want {
				t.Errorf("NormalizeBaseline(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
