package domain

type PushBranch struct {
	ID             string `json:"id"`
	Repo           string `json:"repo"`
	RepoMark       string `json:"repo_mark"`
	Branch         string `json:"branch"`
	CommitCount    int    `json:"commit_count"`
	RelativeTime   string `json:"relative_time"`
	Status         string `json:"status"`
	AIBadge        string `json:"ai_badge"`
	AISummary      string `json:"ai_summary"`
	SuggestedTitle string `json:"suggested_title"`
}

type PullRequest struct {
	ID              string `json:"id"`
	Number          int    `json:"number"`
	Repo            string `json:"repo"`
	Title           string `json:"title"`
	SourceBranch    string `json:"source_branch"`
	TargetBranch    string `json:"target_branch"`
	Status          string `json:"status"`
	UpdatedRelative string `json:"updated_relative"`
	Author          string `json:"author"`
	FilesCount      int    `json:"files_count"`
	LinesAdded      int    `json:"lines_added"`
	LinesDeleted    int    `json:"lines_deleted"`
	IsAssignedToMe  bool   `json:"is_assigned_to_me"`
	IsAIFlagged     bool   `json:"is_ai_flagged"`
}

type DiffHunk struct {
	Header    string   `json:"header"`
	OldStart  int      `json:"old_start"`
	OldLines  int      `json:"old_lines"`
	NewStart  int      `json:"new_start"`
	NewLines  int      `json:"new_lines"`
	Lines     []string `json:"lines"`
}

type DiffFile struct {
	OldPath   string     `json:"old_path"`
	NewPath   string     `json:"new_path"`
	Status    string     `json:"status"`
	Hunks     []DiffHunk `json:"hunks"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
}

type PRDiff struct {
	PRID         string     `json:"pr_id"`
	Repo         string     `json:"repo"`
	TotalAdded   int        `json:"total_added"`
	TotalDeleted int        `json:"total_deleted"`
	Files        []DiffFile `json:"files"`
}

type AICodeReview struct {
	PRID             string   `json:"pr_id"`
	Summary          string   `json:"summary"`
	Findings         []string `json:"findings"`
	GeneratedComment string   `json:"generated_comment"`
}

type PRComment struct {
	ID        string `json:"id"`
	PRID      string `json:"pr_id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type CreatePRRequest struct {
	Repo         string `json:"repo"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Title        string `json:"title"`
	Description  string `json:"description"`
}

type ReviewActionRequest struct {
	Action string `json:"action"`
}

type PRCommentRequest struct {
	Content string `json:"content"`
}
