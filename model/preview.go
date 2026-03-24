package model

// PreviewResult holds a resolved request snapshot that is safe to display.
// Secret values must already be redacted before this structure is rendered.
type PreviewResult struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Query        map[string]string `json:"query,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	Files        []FileUpload      `json:"files,omitempty"`
	Masked       bool              `json:"masked"`
	SecretKeys   []string          `json:"secret_keys,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	RequestName  string            `json:"request_name,omitempty"`
	CollectionID string            `json:"collection_id,omitempty"`
}
