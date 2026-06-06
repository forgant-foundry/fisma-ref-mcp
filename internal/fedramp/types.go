package fedramp

// Catalog is the full FedRAMP machine-readable documentation (FRMR).
type Catalog struct {
	Version     string
	LastUpdated string
	Terms       []Term
	KSIThemes   []KSITheme
	Requirements []RequirementCategory
}

// Term is a FedRAMP glossary definition from the FRD section.
type Term struct {
	ID         string   `json:"id"`
	Term       string   `json:"term"`
	Definition string   `json:"definition"`
	Alts       []string `json:"alts,omitempty"`
	Note       string   `json:"note,omitempty"`
}

// KSITheme is one of the 11 FedRAMP 20x Key Security Indicator themes.
type KSITheme struct {
	ID         string         `json:"id"`
	ShortName  string         `json:"short_name"`
	Name       string         `json:"name"`
	Theme      string         `json:"theme"`
	Indicators []KSIIndicator `json:"indicators"`
}

// KSIIndicator is a single outcome-based security indicator within a theme.
type KSIIndicator struct {
	ID           string   `json:"id"`
	ThemeID      string   `json:"theme_id"`
	Name         string   `json:"name"`
	Statement    string   `json:"statement"`
	Controls     []string `json:"controls"`              // normalized SP 800-53 IDs
	Terms        []string `json:"terms,omitempty"`
	Reference    string   `json:"reference,omitempty"`
	ReferenceURL string   `json:"reference_url,omitempty"`
}

// RequirementCategory is one of the FedRAMP process requirement areas (ADS, CCM, VDR, etc.).
type RequirementCategory struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Purpose      string        `json:"purpose,omitempty"`
	Requirements []Requirement `json:"requirements"`
}

// Requirement is a single FedRAMP MUST/SHOULD statement.
type Requirement struct {
	ID        string   `json:"id"`
	Category  string   `json:"category"`
	Name      string   `json:"name"`
	Statement string   `json:"statement"`
	Keyword   string   `json:"keyword"` // MUST | SHOULD | MAY | MUST NOT
	Version   string   `json:"version"` // rev5 | 20x | both
	Affects   []string `json:"affects,omitempty"`
	Terms     []string `json:"terms,omitempty"`
	Reference string   `json:"reference,omitempty"`
}
