package domain

const (
	SourceJiraLinks   = "jira-links"
	SourceCqlAll      = "cql-all"
	SourceNoLinks     = "no-links"
	SourceUnavailable = "unavailable"
)

type ConfluenceDocument struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TypeLabel   string `json:"type_label"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Owner       string `json:"owner"`
	LastEditor  string `json:"last_editor"`
	Updated     string `json:"updated"`
	Space       string `json:"space"`
	Description string `json:"description"`
	Body        string `json:"body"`
	URL         string `json:"url"`
}

type ConfluenceDocumentsResponse struct {
	Documents      []ConfluenceDocument `json:"documents"`
	Total          int                  `json:"total"`
	Source         string               `json:"source"`
	Degraded       bool                 `json:"degraded"`
	DegradedReason string               `json:"degraded_reason,omitempty"`
}
