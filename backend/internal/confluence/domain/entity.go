package domain

type ConfluenceDocument struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TypeLabel   string `json:"type_label"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Owner       string `json:"owner"`
	Updated     string `json:"updated"`
	Space       string `json:"space"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type ConfluenceDocumentsResponse struct {
	Documents []ConfluenceDocument `json:"documents"`
	Total     int                  `json:"total"`
}
