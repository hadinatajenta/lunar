package domain

const (
	PromptVersion = "queryreview-v1"

	ErrorCodeNoCompareLink = "NO_COMPARE_LINK"

	RepositoryStatusSuccess   = "success"
	RepositoryStatusNoQueries = "no_queries"
	RepositoryStatusError     = "error"
)

type QueryReviewRequest struct {
	Model  string         `json:"model"`
	Source *CompareSource `json:"source"`
}

type CompareSource struct {
	CompareURL   string `json:"compare_url"`
	ProjectKey   string `json:"project_key"`
	RepoSlug     string `json:"repo_slug"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
}

type QueryReviewResult struct {
	DocumentID          string                  `json:"document_id"`
	Model               string                  `json:"model"`
	PromptVersion       string                  `json:"prompt_version"`
	MigPageID           string                  `json:"mig_page_id"`
	MigPageTitle        string                  `json:"mig_page_title"`
	SkippedRepositories []SkippedRepository     `json:"skipped_repositories"`
	Repositories        []RepositoryQueryReview `json:"repositories"`
}

type SkippedRepository struct {
	Repository string `json:"repository"`
	Reason     string `json:"reason"`
}

type MigTarget struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	BaseRef    string `json:"base_ref"`
}

type MigTargetsResult struct {
	MigPageTitle        string              `json:"mig_page_title"`
	Targets             []MigTarget         `json:"targets"`
	SkippedRepositories []SkippedRepository `json:"skipped_repositories"`
}

type MigRepository struct {
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	TargetBranch string `json:"target_branch"`
	CompareURL   string `json:"compare_url"`
}

type RepositoryQueryReview struct {
	Repository    string         `json:"repository"`
	ProjectKey    string         `json:"project_key"`
	SourceBranch  string         `json:"source_branch"`
	TargetBranch  string         `json:"target_branch"`
	CompareURL    string         `json:"compare_url"`
	Status        string         `json:"status"`
	AnalyzedFiles []AnalyzedFile `json:"analyzed_files"`
	SkippedFiles  []SkippedFile  `json:"skipped_files"`
	Queries       []QueryItem    `json:"queries"`
	Summary       string         `json:"summary"`
	Truncated     bool           `json:"truncated"`
	Error         string         `json:"error"`
}

type AnalyzedFile struct {
	Path       string `json:"path"`
	AddedLines int    `json:"added_lines"`
	Language   string `json:"language"`
}

type SkippedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type QueryItem struct {
	No             int    `json:"no"`
	Feature        string `json:"feature"`
	Repository     string `json:"repository"`
	Table          string `json:"table"`
	ActionType     string `json:"action_type"`
	QueryStatement string `json:"query_statement"`
	Index          string `json:"index"`
	CreateIndex    string `json:"create_index"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Notes          string `json:"notes"`
}

type QueryExtraction struct {
	Queries       []QueryItem
	Summary       string
	PromptVersion string
}

type DiffAnalysis struct {
	Files     []PromptFile
	Analyzed  []AnalyzedFile
	Skipped   []SkippedFile
	Truncated bool
}

type PromptFile struct {
	Path       string
	Language   string
	AddedLines []PromptLine
}

type PromptLine struct {
	Number  int
	Content string
}

type PullRequestBranches struct {
	SourceBranch string
	TargetBranch string
}

type ResolvedSource struct {
	CompareURL   string
	ProjectKey   string
	RepoSlug     string
	SourceBranch string
	TargetBranch string
}
