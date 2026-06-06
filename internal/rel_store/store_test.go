package rel_store_test

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/rel_store"
)

var testStore *rel_store.Store

func TestMain(m *testing.M) {
	var err error
	testStore, err = rel_store.New(context.Background(), rel_store.Config{})
	if err != nil {
		log.Fatalf("rel_store.New: %v", err)
	}
	defer testStore.Close()
	os.Exit(m.Run())
}

func bg() context.Context { return context.Background() }

// ── SP 800-53 ────────────────────────────────────────────────────────────────

func TestListFamilies(t *testing.T) {
	families, err := testStore.ListFamilies(bg())
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 20 {
		t.Errorf("got %d families, want 20", len(families))
	}
}

func TestGetFamily(t *testing.T) {
	controls, err := testStore.GetFamily(bg(), "AC")
	if err != nil {
		t.Fatal(err)
	}
	if len(controls) == 0 {
		t.Fatal("AC family returned no controls")
	}
	for _, c := range controls {
		if c.IsEnhancement {
			t.Errorf("GetFamily returned enhancement %s; want base controls only", c.ID)
		}
		if c.FamilyID != "AC" {
			t.Errorf("control %s has FamilyID %q, want AC", c.ID, c.FamilyID)
		}
	}
}

func TestGetControl(t *testing.T) {
	tests := []struct {
		input  string
		wantID string
	}{
		{"AC-1", "AC-1"},
		{"ac-1", "AC-1"},
		{"AC-2(1)", "AC-2(1)"},
		{"ac-2(1)", "AC-2(1)"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, err := testStore.GetControl(bg(), tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if c.ID != tt.wantID {
				t.Errorf("got ID %q, want %q", c.ID, tt.wantID)
			}
			if c.Title == "" {
				t.Error("Title is empty")
			}
		})
	}
}

func TestGetControl_NotFound(t *testing.T) {
	_, err := testStore.GetControl(bg(), "ZZ-999")
	if err == nil {
		t.Error("expected error for unknown control, got nil")
	}
}

func TestGetControl_BaselinesPopulated(t *testing.T) {
	// AC-2 is in all four baselines.
	c, err := testStore.GetControl(bg(), "AC-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Baselines) == 0 {
		t.Error("AC-2 Baselines field is empty; expected at least one baseline")
	}
}

// ── SP 800-53B baselines ──────────────────────────────────────────────────────

func TestGetBaseline(t *testing.T) {
	tests := []struct {
		name    string
		wantMin int
	}{
		{"low", 100},
		{"moderate", 200},
		{"high", 300},
		{"privacy", 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controls, err := testStore.GetBaseline(bg(), tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if len(controls) < tt.wantMin {
				t.Errorf("got %d controls, want at least %d", len(controls), tt.wantMin)
			}
		})
	}
}

func TestGetBaseline_Unknown(t *testing.T) {
	_, err := testStore.GetBaseline(bg(), "extreme")
	if err == nil {
		t.Error("expected error for unknown baseline, got nil")
	}
}

// ── FISMA metrics ─────────────────────────────────────────────────────────────

func TestListFismaMetrics(t *testing.T) {
	all, err := testStore.ListFismaMetrics(bg(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 35 {
		t.Errorf("got %d metrics, want 35", len(all))
	}
}

func TestListFismaMetrics_DomainFilter(t *testing.T) {
	all, err := testStore.ListFismaMetrics(bg(), "")
	if err != nil {
		t.Fatal(err)
	}
	domain := all[0].Domain
	filtered, err := testStore.ListFismaMetrics(bg(), domain)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) == 0 {
		t.Fatalf("domain filter %q returned no metrics", domain)
	}
	if len(filtered) >= len(all) {
		t.Errorf("domain filter returned %d metrics (same as unfiltered %d); filter may be broken", len(filtered), len(all))
	}
	for _, m := range filtered {
		if m.Domain != domain {
			t.Errorf("metric %d has domain %q, want %q", m.ID, m.Domain, domain)
		}
	}
}

func TestGetFismaMetric(t *testing.T) {
	m, err := testStore.GetFismaMetric(bg(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != 1 {
		t.Errorf("got ID %d, want 1", m.ID)
	}
	if m.Question == "" {
		t.Error("Question is empty")
	}
	if len(m.MaturityLevels) == 0 {
		t.Error("MaturityLevels is empty")
	}
}

func TestGetFismaMetric_MaturityLevelEvidence(t *testing.T) {
	// At least one metric should have Evidence populated in at least one level.
	all, err := testStore.ListFismaMetrics(bg(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, stub := range all {
		m, err := testStore.GetFismaMetric(bg(), stub.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, lvl := range m.MaturityLevels {
			if lvl.Evidence != "" {
				return // found one — pass
			}
		}
	}
	t.Error("no maturity level across all metrics has Evidence text; data may not be loading correctly")
}

func TestGetFismaMetric_NotFound(t *testing.T) {
	_, err := testStore.GetFismaMetric(bg(), 9999)
	if err == nil {
		t.Error("expected error for unknown metric, got nil")
	}
}

// ── Cross-corpus: SP 800-53 ↔ FISMA ──────────────────────────────────────────

func TestGetMetricsByControl(t *testing.T) {
	metrics, err := testStore.GetMetricsByControl(bg(), "AC-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) == 0 {
		t.Error("AC-2 returned no FISMA metrics")
	}
}

func TestGetMetricsByControl_CaseInsensitive(t *testing.T) {
	upper, err := testStore.GetMetricsByControl(bg(), "AC-2")
	if err != nil {
		t.Fatal(err)
	}
	lower, err := testStore.GetMetricsByControl(bg(), "ac-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(upper) != len(lower) {
		t.Errorf("AC-2 returned %d metrics, ac-2 returned %d; case normalization broken", len(upper), len(lower))
	}
}

// ── NIST CSF 2.0 ─────────────────────────────────────────────────────────────

func TestListCSFFunctions(t *testing.T) {
	fns, err := testStore.ListCSFFunctions(bg())
	if err != nil {
		t.Fatal(err)
	}
	if len(fns) != 6 {
		t.Errorf("got %d CSF functions, want 6", len(fns))
	}
}

func TestListCSFCategories_FunctionFilter(t *testing.T) {
	all, err := testStore.ListCSFCategories(bg(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("ListCSFCategories returned no categories")
	}
	filtered, err := testStore.ListCSFCategories(bg(), "GV")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) == 0 {
		t.Fatal("GV function filter returned no categories")
	}
	if len(filtered) >= len(all) {
		t.Errorf("function filter returned %d categories (same as unfiltered %d)", len(filtered), len(all))
	}
	for _, c := range filtered {
		if c.FunctionID != "GV" {
			t.Errorf("category %s has FunctionID %q, want GV", c.ID, c.FunctionID)
		}
	}
}

func TestGetCSFSubcategory(t *testing.T) {
	tests := []struct {
		id string
	}{
		{"GV.OC-01"},
		{"gv.oc-01"}, // case insensitive
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			sub, err := testStore.GetCSFSubcategory(bg(), tt.id)
			if err != nil {
				t.Fatal(err)
			}
			if sub.ID != strings.ToUpper(tt.id) {
				t.Errorf("got ID %q, want %q", sub.ID, strings.ToUpper(tt.id))
			}
			if sub.Text == "" {
				t.Error("subcategory Text is empty")
			}
		})
	}
}

func TestGetCSFSubcategory_NotFound(t *testing.T) {
	_, err := testStore.GetCSFSubcategory(bg(), "ZZ.ZZ-99")
	if err == nil {
		t.Error("expected error for unknown subcategory, got nil")
	}
}

// ── Cross-corpus: SP 800-53 ↔ CSF ────────────────────────────────────────────

func TestGetCSFSubcategoriesByControl(t *testing.T) {
	subs, err := testStore.GetCSFSubcategoriesByControl(bg(), "AC-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) == 0 {
		t.Error("AC-2 returned no CSF subcategories from crosswalk")
	}
}

func TestGetCSFSubcategoriesByControl_CaseInsensitive(t *testing.T) {
	upper, err := testStore.GetCSFSubcategoriesByControl(bg(), "AC-2")
	if err != nil {
		t.Fatal(err)
	}
	lower, err := testStore.GetCSFSubcategoriesByControl(bg(), "ac-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(upper) != len(lower) {
		t.Errorf("AC-2 returned %d subcategories, ac-2 returned %d; case normalization broken", len(upper), len(lower))
	}
}

// ── Cross-corpus: CSF ↔ FISMA ────────────────────────────────────────────────

func TestGetMetricsByCSFSubcategory(t *testing.T) {
	// Find a CSF subcategory ID that's actually referenced by a FISMA criterion.
	all, err := testStore.ListFismaMetrics(bg(), "")
	if err != nil {
		t.Fatal(err)
	}
	var subID string
	for _, stub := range all {
		m, err := testStore.GetFismaMetric(bg(), stub.ID)
		if err != nil {
			continue
		}
		for _, cr := range m.Criteria {
			if cr.CSFSubcategoryID != "" {
				subID = cr.CSFSubcategoryID
				break
			}
		}
		if subID != "" {
			break
		}
	}
	if subID == "" {
		t.Skip("no CSF subcategory criteria found in any metric")
	}
	results, err := testStore.GetMetricsByCSFSubcategory(bg(), subID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Errorf("GetMetricsByCSFSubcategory(%q) returned no metrics", subID)
	}
}

// ── FedRAMP ───────────────────────────────────────────────────────────────────

func TestListKSIThemes(t *testing.T) {
	themes, err := testStore.ListKSIThemes(bg())
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 11 {
		t.Errorf("got %d KSI themes, want 11", len(themes))
	}
	for _, th := range themes {
		if len(th.Indicators) == 0 {
			t.Errorf("theme %s has no indicators", th.ID)
		}
	}
}

func TestGetKSI(t *testing.T) {
	themes, err := testStore.ListKSIThemes(bg())
	if err != nil {
		t.Fatal(err)
	}
	ind := themes[0].Indicators[0]
	got, err := testStore.GetKSI(bg(), ind.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != ind.ID {
		t.Errorf("got ID %q, want %q", got.ID, ind.ID)
	}
	if got.Statement == "" {
		t.Errorf("KSI %s has empty statement", got.ID)
	}
}

func TestGetKSI_NotFound(t *testing.T) {
	_, err := testStore.GetKSI(bg(), "KSI-DOESNOTEXIST")
	if err == nil {
		t.Error("expected error for unknown KSI, got nil")
	}
}

func TestListKSIThemes_ControlsPopulated(t *testing.T) {
	themes, err := testStore.ListKSIThemes(bg())
	if err != nil {
		t.Fatal(err)
	}
	for _, th := range themes {
		for _, ind := range th.Indicators {
			if len(ind.Controls) > 0 {
				return // at least one indicator has controls — pass
			}
		}
	}
	t.Error("no KSI indicator across all themes has Controls populated; listKSIThemes may be missing the ksi_controls join")
}

func TestGetKSIsByControl(t *testing.T) {
	// Controls are now populated by ListKSIThemes directly.
	themes, err := testStore.ListKSIThemes(bg())
	if err != nil {
		t.Fatal(err)
	}
	var controlID string
	for _, th := range themes {
		for _, ind := range th.Indicators {
			if len(ind.Controls) > 0 {
				controlID = ind.Controls[0]
				break
			}
		}
		if controlID != "" {
			break
		}
	}
	if controlID == "" {
		t.Skip("no KSI indicators have SP 800-53 control references")
	}
	inds, err := testStore.GetKSIsByControl(bg(), controlID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inds) == 0 {
		t.Errorf("GetKSIsByControl(%q) returned no indicators", controlID)
	}
}

func TestListFedRAMPRequirements(t *testing.T) {
	all, err := testStore.ListFedRAMPRequirements(bg(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("ListFedRAMPRequirements returned no requirements")
	}

	rev5, err := testStore.ListFedRAMPRequirements(bg(), "", "rev5")
	if err != nil {
		t.Fatal(err)
	}
	if len(rev5) == 0 {
		t.Error("rev5 filter returned no requirements")
	}
	if len(rev5) >= len(all) {
		t.Errorf("rev5 filter returned %d (same as unfiltered %d); version filter may be broken", len(rev5), len(all))
	}

	// Requirements with version="both" must appear in rev5 and 20x filtered results.
	var bothCount int
	for _, r := range all {
		if r.Version == "both" {
			bothCount++
		}
	}
	if bothCount > 0 {
		for _, label := range []string{"rev5", "20x"} {
			filtered, err := testStore.ListFedRAMPRequirements(bg(), "", label)
			if err != nil {
				t.Fatal(err)
			}
			var gotBoth int
			for _, r := range filtered {
				if r.Version == "both" {
					gotBoth++
				}
			}
			if gotBoth != bothCount {
				t.Errorf("%s filter: got %d 'both' requirements, want %d", label, gotBoth, bothCount)
			}
		}
	}
}

func TestGetFedRAMPTerm(t *testing.T) {
	term, err := testStore.GetFedRAMPTerm(bg(), "FRD-ACV")
	if err != nil {
		t.Fatal(err)
	}
	if term.Term == "" {
		t.Error("Term name is empty")
	}
	if term.Definition == "" {
		t.Error("Definition is empty")
	}
}

func TestGetFedRAMPTerm_NotFound(t *testing.T) {
	_, err := testStore.GetFedRAMPTerm(bg(), "FRD-DOESNOTEXIST")
	if err == nil {
		t.Error("expected error for unknown term, got nil")
	}
}

func TestGetFedRAMPRequirement(t *testing.T) {
	// Derive a real requirement ID from the list rather than hardcoding.
	all, err := testStore.ListFedRAMPRequirements(bg(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("no requirements to test against")
	}
	req := all[0]
	got, err := testStore.GetFedRAMPRequirement(bg(), req.ID)
	if err != nil {
		t.Fatalf("GetFedRAMPRequirement(%q): %v", req.ID, err)
	}
	if got.ID != req.ID {
		t.Errorf("got ID %q, want %q", got.ID, req.ID)
	}
	if got.Category != req.Category {
		t.Errorf("got Category %q, want %q", got.Category, req.Category)
	}
}

func TestGetFedRAMPRequirement_NotFound(t *testing.T) {
	_, err := testStore.GetFedRAMPRequirement(bg(), "DOESNOTEXIST-XXX")
	if err == nil {
		t.Error("expected error for unknown requirement, got nil")
	}
}

// ── FTS5 search ───────────────────────────────────────────────────────────────

func TestSearch_EachSource(t *testing.T) {
	tests := []struct {
		source string
		query  string
	}{
		{"nist_800_53", "access control"},
		{"fisma_fy2025", "authentication"},
		{"nist_csf_v2", "risk"},
		{"fedramp_20x", "authorization"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			results, err := testStore.Search(bg(), tt.query, 5, tt.source)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) == 0 {
				t.Errorf("search %q in %s returned no results", tt.query, tt.source)
			}
			for _, r := range results {
				if r.Source != tt.source {
					t.Errorf("result source %q != requested source %q", r.Source, tt.source)
				}
				if r.ID == "" {
					t.Error("result has empty ID")
				}
			}
		})
	}
}

func TestSearch_CrossCorpus(t *testing.T) {
	results, err := testStore.Search(bg(), "authentication", 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("cross-corpus search returned no results")
	}
	seen := make(map[string]bool)
	for _, r := range results {
		seen[r.Source] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected results from multiple corpora, got only: %v", seen)
	}
}

func TestSearch_FedrampTerms(t *testing.T) {
	// FTS5 search within fedramp_20x should now reach glossary terms.
	// Use a term that's unlikely to appear in KSI/requirement text alone.
	results, err := testStore.Search(bg(), "cloud service provider", 10, "fedramp_20x")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("fedramp_20x search for glossary term returned no results")
	}
}

func TestSearch_LimitRespected(t *testing.T) {
	results, err := testStore.Search(bg(), "control", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 3 {
		t.Errorf("got %d results with limit 3", len(results))
	}
}
