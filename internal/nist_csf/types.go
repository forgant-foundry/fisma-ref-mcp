package nist_csf

// Function is one of the six CSF 2.0 functions (GV, ID, PR, DE, RS, RC).
type Function struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Category is a group of related subcategories within a Function.
type Category struct {
	ID         string `json:"id"`
	FunctionID string `json:"function_id"`
	Title      string `json:"title"`
	Text       string `json:"text"`
}

// Subcategory is a specific outcome statement within a Category.
type Subcategory struct {
	ID         string   `json:"id"`
	CategoryID string   `json:"category_id"`
	FunctionID string   `json:"function_id"`
	Text       string   `json:"text"`
	Examples   []string `json:"examples"`
}
