package tools

const (
	ToolNameListMyIssues          = "list_my_jira_issues"
	ToolNameGetIssueDetail        = "get_jira_issue_detail"
	ToolNameGetActiveSprintIssues = "get_active_sprint_issues"
)

const listMyIssuesDescription = "List the Jira issues assigned to the signed-in user, with each issue's key, summary, status, type, priority, assignee, and sprint. Use it to answer what the user is working on."

const getIssueDetailDescription = "Read one Jira issue by its key, returning its summary, status, type, priority, assignee, sprint, parent, due date, and description. Use it after locating the issue key."

const getActiveSprintIssuesDescription = "List the issues in the active sprint of the signed-in user's own squad, grouped by sprint, so the backlog can be reviewed without opening each issue."

const listMyIssuesSchema = `{
  "type": "object",
  "properties": {}
}`

const getIssueDetailSchema = `{
  "type": "object",
  "properties": {
    "issue_key": {
      "type": "string",
      "description": "Jira issue key, for example 'CRMMS-6397'."
    }
  },
  "required": ["issue_key"]
}`

const getActiveSprintIssuesSchema = `{
  "type": "object",
  "properties": {}
}`
