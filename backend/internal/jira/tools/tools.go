package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	agentdomain "lunar/backend/internal/agent/domain"
	jiradomain "lunar/backend/internal/jira/domain"
)

const (
	maxResultCharacters  = 12000
	maxIssueRows         = 50
	maxSnippetCharacters = 240
	sourceTypeJira       = "jira"
	truncationNote       = "\n[result truncated]\n"
)

type issueStatus struct {
	Name           string `json:"name"`
	StatusCategory struct {
		Key string `json:"key"`
	} `json:"statusCategory"`
}

type issueType struct {
	Name string `json:"name"`
}

type issuePriority struct {
	Name string `json:"name"`
}

type issueUser struct {
	DisplayName string `json:"displayName"`
}

type issueParent struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

type issueProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type issueFields struct {
	Summary     string         `json:"summary"`
	Description string         `json:"description"`
	Updated     string         `json:"updated"`
	Project     issueProject   `json:"project"`
	Status      issueStatus    `json:"status"`
	IssueType   *issueType     `json:"issuetype"`
	Priority    *issuePriority `json:"priority"`
	Assignee    *issueUser     `json:"assignee"`
	Parent      *issueParent   `json:"parent"`
	DueDate     *string        `json:"duedate"`
}

type issueRecord struct {
	Key        string      `json:"key"`
	SprintName string      `json:"sprint_name"`
	Fields     issueFields `json:"fields"`
}

type sprintRecord struct {
	ID     int           `json:"id"`
	Name   string        `json:"name"`
	State  string        `json:"state"`
	Issues []issueRecord `json:"issues"`
}

type backlogResponse struct {
	Sprints          []sprintRecord `json:"sprints"`
	TotalIssues      int            `json:"total_issues"`
	ActiveSprintName string         `json:"active_sprint_name"`
	DetectedSquad    string         `json:"detected_squad"`
	AvailableSquads  []string       `json:"available_squads"`
}

type issueDetailArguments struct {
	IssueKey string `json:"issue_key"`
}

type toolDependencies struct {
	client *BackendClient
}

func NewTools(client *BackendClient) []agentdomain.Tool {
	dependencies := toolDependencies{client: client}
	return []agentdomain.Tool{
		listMyIssuesTool{dependencies: dependencies},
		getIssueDetailTool{dependencies: dependencies},
		getActiveSprintIssuesTool{dependencies: dependencies},
	}
}

func toolDefinition(name string, description string, parameters string) agentdomain.ToolDefinition {
	return agentdomain.ToolDefinition{
		Type: "function",
		Function: agentdomain.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  json.RawMessage(parameters),
			Strict:      true,
		},
	}
}

type listMyIssuesTool struct {
	dependencies toolDependencies
}

func (t listMyIssuesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameListMyIssues, listMyIssuesDescription, listMyIssuesSchema)
}

func (t listMyIssuesTool) Execute(ctx context.Context, _ json.RawMessage) (agentdomain.ToolResult, error) {
	body, err := t.dependencies.client.getJSON(ctx, "/api/jira/issues")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("list_my_jira_issues: %w", err)
	}

	var issues []issueRecord
	if err := json.Unmarshal(body, &issues); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("list_my_jira_issues: parse response: %w", err)
	}
	if len(issues) == 0 {
		return emptyToolResult("No Jira issues are assigned to you."), nil
	}

	return buildIssueListResult("Jira issues assigned to you", issues), nil
}

type getIssueDetailTool struct {
	dependencies toolDependencies
}

func (t getIssueDetailTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetIssueDetail, getIssueDetailDescription, getIssueDetailSchema)
}

func (t getIssueDetailTool) Execute(ctx context.Context, args json.RawMessage) (agentdomain.ToolResult, error) {
	var arguments issueDetailArguments
	if err := json.Unmarshal(args, &arguments); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_jira_issue_detail: invalid arguments: %w", err)
	}

	issueKey, err := jiradomain.ValidateIssueKey(arguments.IssueKey)
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_jira_issue_detail: %w", err)
	}

	body, err := t.dependencies.client.getJSON(ctx, "/api/jira/issues/"+url.PathEscape(issueKey))
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_jira_issue_detail: %w", err)
	}

	var issue issueRecord
	if err := json.Unmarshal(body, &issue); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_jira_issue_detail: parse response: %w", err)
	}
	if strings.TrimSpace(issue.Key) == "" {
		return emptyToolResult(fmt.Sprintf("Jira issue %s was not found.", issueKey)), nil
	}

	return agentdomain.ToolResult{
		Content: summarizeIssueDetail(issue),
		Sources: []agentdomain.SourceReference{issueSource(issue)},
	}, nil
}

type getActiveSprintIssuesTool struct {
	dependencies toolDependencies
}

func (t getActiveSprintIssuesTool) Definition() agentdomain.ToolDefinition {
	return toolDefinition(ToolNameGetActiveSprintIssues, getActiveSprintIssuesDescription, getActiveSprintIssuesSchema)
}

func (t getActiveSprintIssuesTool) Execute(ctx context.Context, _ json.RawMessage) (agentdomain.ToolResult, error) {
	body, err := t.dependencies.client.getJSON(ctx, "/api/jira/backlog")
	if err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_active_sprint_issues: %w", err)
	}

	var backlog backlogResponse
	if err := json.Unmarshal(body, &backlog); err != nil {
		return agentdomain.ToolResult{}, fmt.Errorf("get_active_sprint_issues: parse response: %w", err)
	}

	return buildBacklogResult(backlog), nil
}

func buildIssueListResult(title string, issues []issueRecord) agentdomain.ToolResult {
	budget := newTextBudget(maxResultCharacters)
	budget.write(fmt.Sprintf("%s (%d):\n\n", title, len(issues)))

	sources := make([]agentdomain.SourceReference, 0, len(issues))
	written := 0
	for _, issue := range issues {
		if written >= maxIssueRows {
			budget.write(fmt.Sprintf("[%d more issues not shown]\n", len(issues)-written))
			break
		}
		if !budget.write("- " + summarizeIssue(issue) + "\n") {
			break
		}
		sources = append(sources, issueSource(issue))
		written++
	}

	return agentdomain.ToolResult{Content: budget.finish(), Sources: sources}
}

func buildBacklogResult(backlog backlogResponse) agentdomain.ToolResult {
	budget := newTextBudget(maxResultCharacters)

	header := "Active sprint backlog"
	if name := strings.TrimSpace(backlog.ActiveSprintName); name != "" {
		header = fmt.Sprintf("Active sprint backlog: %s", name)
	}
	budget.write(fmt.Sprintf("%s (%d issues):\n\n", header, backlog.TotalIssues))
	if squad := strings.TrimSpace(backlog.DetectedSquad); squad != "" {
		budget.write(fmt.Sprintf("Squad: %s\n\n", squad))
	}

	sources := make([]agentdomain.SourceReference, 0)
	written := 0
	isFull := false
	for _, sprint := range backlog.Sprints {
		budget.write(fmt.Sprintf("## %s (%s)\n", strings.TrimSpace(sprint.Name), strings.TrimSpace(sprint.State)))
		for _, issue := range sprint.Issues {
			if written >= maxIssueRows {
				isFull = true
				break
			}
			if issue.SprintName == "" {
				issue.SprintName = sprint.Name
			}
			if !budget.write("- " + summarizeIssue(issue) + "\n") {
				isFull = true
				break
			}
			sources = append(sources, issueSource(issue))
			written++
		}
		budget.write("\n")
		if isFull {
			break
		}
	}
	if isFull {
		budget.write("[more issues not shown]\n")
	}
	if written == 0 {
		budget.write("No active sprint issues were found.\n")
	}

	return agentdomain.ToolResult{Content: budget.finish(), Sources: sources}
}

func summarizeIssue(issue issueRecord) string {
	key := strings.TrimSpace(issue.Key)
	summary := strings.TrimSpace(issue.Fields.Summary)

	line := key
	if summary != "" {
		line = key + " " + summary
	}
	if attributes := issueAttributeList(issue); len(attributes) > 0 {
		line += " (" + strings.Join(attributes, ", ") + ")"
	}
	return line
}

func summarizeIssueDetail(issue issueRecord) string {
	budget := newTextBudget(maxResultCharacters)

	key := strings.TrimSpace(issue.Key)
	summary := strings.TrimSpace(issue.Fields.Summary)
	budget.write(fmt.Sprintf("%s: %s\n", key, summary))

	if attributes := issueAttributeList(issue); len(attributes) > 0 {
		budget.write(strings.Join(attributes, "\n") + "\n")
	}
	if project := strings.TrimSpace(issue.Fields.Project.Key); project != "" {
		budget.write("project=" + project + "\n")
	}
	if issue.Fields.Parent != nil && strings.TrimSpace(issue.Fields.Parent.Key) != "" {
		budget.write(fmt.Sprintf("parent=%s %s\n", strings.TrimSpace(issue.Fields.Parent.Key), strings.TrimSpace(issue.Fields.Parent.Fields.Summary)))
	}
	if issue.Fields.DueDate != nil && strings.TrimSpace(*issue.Fields.DueDate) != "" {
		budget.write("due=" + strings.TrimSpace(*issue.Fields.DueDate) + "\n")
	}
	if updated := strings.TrimSpace(issue.Fields.Updated); updated != "" {
		budget.write("updated=" + updated + "\n")
	}
	if description := strings.TrimSpace(issue.Fields.Description); description != "" {
		budget.write("\nDescription:\n" + description + "\n")
	}

	return budget.finish()
}

func issueAttributeList(issue issueRecord) []string {
	attributes := make([]string, 0, 5)

	if name := strings.TrimSpace(issue.Fields.Status.Name); name != "" {
		attributes = append(attributes, "status="+name)
	}
	if issue.Fields.IssueType != nil {
		if name := strings.TrimSpace(issue.Fields.IssueType.Name); name != "" {
			attributes = append(attributes, "type="+name)
		}
	}
	if issue.Fields.Priority != nil {
		if name := strings.TrimSpace(issue.Fields.Priority.Name); name != "" {
			attributes = append(attributes, "priority="+name)
		}
	}
	if issue.Fields.Assignee != nil {
		if name := strings.TrimSpace(issue.Fields.Assignee.DisplayName); name != "" {
			attributes = append(attributes, "assignee="+name)
		}
	}
	if sprint := strings.TrimSpace(issue.SprintName); sprint != "" {
		attributes = append(attributes, "sprint="+sprint)
	}

	return attributes
}

func issueSource(issue issueRecord) agentdomain.SourceReference {
	return agentdomain.SourceReference{
		Repo:    strings.TrimSpace(issue.Fields.Project.Key),
		File:    strings.TrimSpace(issue.Key),
		Type:    sourceTypeJira,
		Snippet: clipText(summarizeIssue(issue), maxSnippetCharacters),
	}
}

func emptyToolResult(message string) agentdomain.ToolResult {
	return agentdomain.ToolResult{Content: message}
}

type textBudget struct {
	builder   strings.Builder
	maximum   int
	truncated bool
}

func newTextBudget(maximum int) *textBudget {
	return &textBudget{maximum: maximum}
}

func (b *textBudget) write(text string) bool {
	if b.truncated {
		return false
	}

	remaining := b.maximum - b.builder.Len()
	if remaining <= 0 {
		b.truncated = true
		return false
	}
	if len(text) <= remaining {
		b.builder.WriteString(text)
		return true
	}

	b.builder.WriteString(clipText(text, remaining))
	b.truncated = true
	return false
}

func (b *textBudget) finish() string {
	if !b.truncated {
		return b.builder.String()
	}
	return clipText(b.builder.String(), b.maximum-len(truncationNote)) + truncationNote
}

func clipText(text string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if len(text) <= maximum {
		return text
	}

	clipped := text[:maximum]
	for len(clipped) > 0 && !utf8.ValidString(clipped) {
		clipped = clipped[:len(clipped)-1]
	}
	return clipped
}
