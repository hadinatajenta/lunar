package domain

type JiraStatusCategory struct {
	ID        int    `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	ColorName string `json:"colorName,omitempty"`
}

type JiraStatus struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description,omitempty"`
	StatusCategory JiraStatusCategory `json:"statusCategory"`
}

type JiraProject struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type JiraIssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl,omitempty"`
	Subtask bool   `json:"subtask,omitempty"`
}

type JiraPriority struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl,omitempty"`
}

type JiraUser struct {
	Self         string            `json:"self,omitempty"`
	Name         string            `json:"name,omitempty"`
	Key          string            `json:"key,omitempty"`
	EmailAddress string            `json:"emailAddress,omitempty"`
	DisplayName  string            `json:"displayName,omitempty"`
	AvatarURLs   map[string]string `json:"avatarUrls,omitempty"`
}

type JiraParentFields struct {
	Summary   string         `json:"summary,omitempty"`
	Status    *JiraStatus    `json:"status,omitempty"`
	Priority  *JiraPriority  `json:"priority,omitempty"`
	IssueType *JiraIssueType `json:"issuetype,omitempty"`
}

type JiraParent struct {
	ID     string           `json:"id,omitempty"`
	Key    string           `json:"key"`
	Self   string           `json:"self,omitempty"`
	Fields JiraParentFields `json:"fields,omitempty"`
}

type JiraIssueFields struct {
	Summary            string         `json:"summary"`
	Description        string         `json:"description,omitempty"`
	Updated            string         `json:"updated"`
	Created            string         `json:"created,omitempty"`
	Project            JiraProject    `json:"project"`
	Status             JiraStatus     `json:"status"`
	IssueType          *JiraIssueType `json:"issuetype,omitempty"`
	Priority           *JiraPriority  `json:"priority,omitempty"`
	Assignee           *JiraUser      `json:"assignee,omitempty"`
	Parent             *JiraParent    `json:"parent,omitempty"`
	DueDate            *string        `json:"duedate,omitempty"`
	AcceptanceCriteria *string        `json:"customfield_15310,omitempty"`
}

type JiraIssue struct {
	ID         string          `json:"id"`
	Key        string          `json:"key"`
	Self       string          `json:"self"`
	SprintName string          `json:"sprint_name,omitempty"`
	Fields     JiraIssueFields `json:"fields"`
	Kind       string          `json:"kind,omitempty"`
	Points     int             `json:"points,omitempty"`
	SubLabel   string          `json:"sub_label,omitempty"`
}

type JiraSprint struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	State         string      `json:"state"`
	StartDate     string      `json:"startDate,omitempty"`
	EndDate       string      `json:"endDate,omitempty"`
	OriginBoardID int         `json:"originBoardId,omitempty"`
	Issues        []JiraIssue `json:"issues,omitempty"`
}

type JiraBacklogResponse struct {
	Sprints          []JiraSprint `json:"sprints"`
	TotalIssues      int          `json:"total_issues"`
	ActiveSprintName string       `json:"active_sprint_name,omitempty"`
	DetectedSquad    string       `json:"detected_squad,omitempty"`
	AvailableSquads  []string     `json:"available_squads,omitempty"`
}

type JiraSearchResponse struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Issues     []JiraIssue `json:"issues"`
}

type JiraRemoteLinkObject struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type JiraRemoteLink struct {
	ID           int                  `json:"id"`
	Relationship string               `json:"relationship"`
	Object       JiraRemoteLinkObject `json:"object"`
}
