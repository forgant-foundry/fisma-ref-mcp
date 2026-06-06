package nist_800_53

// Family is a NIST SP 800-53 control family (e.g., Access Control).
type Family struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Control is a single NIST SP 800-53 control or control enhancement.
type Control struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	FamilyID      string   `json:"family_id"`
	Statement     string   `json:"statement"`
	Discussion    string   `json:"discussion"`
	IsEnhancement bool     `json:"is_enhancement"`
	ParentID      string   `json:"parent_id,omitempty"`
	Baselines     []string `json:"baselines,omitempty"` // populated from SP 800-53B at startup
}

// --- Raw JSON types (unexported) ---

type rawCatalog struct {
	Response struct {
		Elements struct {
			Documents     []rawDocument     `json:"documents"`
			Elements      []rawElement      `json:"elements"`
			Relationships []rawRelationship `json:"relationships"`
		} `json:"elements"`
	} `json:"response"`
}

type rawDocument struct {
	DocIdentifier string `json:"doc_identifier"`
	Name          string `json:"name"`
	Version       string `json:"version"`
}

type rawElement struct {
	ElementType       string `json:"element_type"`
	ElementIdentifier string `json:"element_identifier"`
	Title             string `json:"title"`
	Text              string `json:"text"`
}

type rawRelationship struct {
	SourceElementIdentifier string `json:"source_element_identifier"`
	DestElementIdentifier   string `json:"dest_element_identifier"`
	RelationshipIdentifier  string `json:"relationship_identifier"`
}
