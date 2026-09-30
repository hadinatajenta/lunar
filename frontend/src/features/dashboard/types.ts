export type DashboardSectionStatus = "ok" | "unconfigured" | "error"

export type DashboardCopilotStatus = "ok" | "error"

export interface DashboardJiraSection {
  status: DashboardSectionStatus
  is_configured: boolean
  assigned_tickets: number
  active_sprint: string | null
  technical_documents: number
}

export interface DashboardBitbucketSection {
  status: DashboardSectionStatus
  is_configured: boolean
  open_pull_requests: number
  review_requested: number
}

export interface DashboardCopilotSection {
  status: DashboardCopilotStatus
  active_tools: number
  total_tools: number
  configured_providers: string[]
}

export interface DashboardSummary {
  synced_at: string
  jira: DashboardJiraSection
  bitbucket: DashboardBitbucketSection
  copilot: DashboardCopilotSection
}
