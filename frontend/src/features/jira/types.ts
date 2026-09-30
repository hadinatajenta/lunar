export type SubfilterCategory = "all" | "docs" | "bugs" | "subtasks"

export type DocTypeFilter = "all" | "ut" | "query" | "sop"

export type JiraItemKind =
  | "bug"
  | "subtask"
  | "ut"
  | "query"
  | "sop"
  | "story"
  | "task"

export type JiraItemStatus = "open" | "progress" | "done"

export type JiraItemPriority = "high" | "medium" | "low"

export interface JiraPriority {
  id?: string
  name: string
  icon_url?: string
}

export interface JiraStatus {
  id?: string
  name: string
  key?: string
  category?: string
}

export interface JiraUser {
  id?: string
  key?: string
  name?: string
  display_name?: string
  avatar_url?: string
  initials?: string
}

export interface JiraIssueFields {
  summary?: string
  description?: string
  status?: JiraStatus
  priority?: JiraPriority
  assignee?: JiraUser
  story_points?: number
}

export interface JiraIssue {
  id: string
  key: string
  title: string
  kind: JiraItemKind | string
  label: string
  status: JiraItemStatus | string
  priority: JiraItemPriority | string
  points: number
  sub?: string
  description?: string
  generated?: string
  assignee?: JiraUser | string
  sprint_id?: string
  sprint_name?: string
  fields?: JiraIssueFields
}

export interface JiraSprint {
  id: string | number
  name: string
  is_active?: boolean
  state?: string
  dates?: string
  startDate?: string
  endDate?: string
  issues: JiraIssue[]
}

export interface JiraBacklogResponse {
  sprints: JiraSprint[]
  total?: number
  total_issues?: number
  active_sprint_name?: string
  detected_squad?: string
  available_squads?: string[]
}

