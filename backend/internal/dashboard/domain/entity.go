package domain

const (
	StatusOK           = "ok"
	StatusUnconfigured = "unconfigured"
	StatusError        = "error"
)

type JiraSummary struct {
	Status             string `json:"status"`
	IsConfigured       bool   `json:"is_configured"`
	AssignedTickets    int    `json:"assigned_tickets"`
	ActiveSprint       string `json:"active_sprint"`
	TechnicalDocuments int    `json:"technical_documents"`
}

type BitbucketSummary struct {
	Status           string `json:"status"`
	IsConfigured     bool   `json:"is_configured"`
	OpenPullRequests int    `json:"open_pull_requests"`
	ReviewRequested  int    `json:"review_requested"`
}

type CopilotSummary struct {
	Status              string   `json:"status"`
	ActiveTools         int      `json:"active_tools"`
	TotalTools          int      `json:"total_tools"`
	ConfiguredProviders []string `json:"configured_providers"`
}

type Summary struct {
	SyncedAt  string           `json:"synced_at"`
	Jira      JiraSummary      `json:"jira"`
	Bitbucket BitbucketSummary `json:"bitbucket"`
	Copilot   CopilotSummary   `json:"copilot"`
}
