package fisma

// Metric represents one FY 2025 IG FISMA evaluation metric.
type Metric struct {
	ID            int            `json:"id"`
	Domain        string         `json:"domain"`
	Question      string         `json:"question"`
	ReviewCycle   string         `json:"review_cycle"`
	MaturityLevels []MaturityLevel `json:"maturity_levels"`
	Criteria      []Criterion    `json:"criteria"`
}

// MaturityLevel holds the assessment criteria for one of the five maturity levels
// (Ad Hoc → Defined → Consistently Implemented → Managed and Measurable → Optimized).
type MaturityLevel struct {
	Level          string `json:"level"`
	Description    string `json:"description"`
	Evidence       string `json:"evidence"`
	AssessorNotes  string `json:"assessor_notes"`
}

// Criterion represents a single reference in the Criteria column.
// Only NIST SP 800-53 references have control_ids populated; others are stubs
// for future traceability work.
type Criterion struct {
	RefType    string   `json:"ref_type"`    // "nist_800_53" | "nist_csf" | "omb" | "fisma" | ...
	RefText    string   `json:"ref_text"`
	ControlIDs []string `json:"control_ids"` // populated when ref_type == "nist_800_53"
}
