export interface WorkspaceScope {
  local_repository_count: number
  remote_repository_count: number
  coverage_percent: number
  verified_remotely: boolean
  remote_repositories_checked: number
}

export interface IssueBranchMatch {
  repo: string
  branch: string
  source: "local" | "remote" | "both"
  is_current: boolean
  is_cloned: boolean
  bitbucket_project: string
  last_commit: string | null
  last_commit_relative: string | null
  ahead: number | null
  behind: number | null
}

export interface IssueBranches {
  key: string
  matches: IssueBranchMatch[]
  scope: WorkspaceScope
}

export interface SprintBranches {
  matches: Record<string, IssueBranchMatch[]>
  scope: WorkspaceScope
}

export interface BaseBranchOption {
  name: string
  last_commit_relative: string | null
  source: "remembered" | "bitbucket_default" | "trusted" | "other"
}

export interface RepositoryBaseBranchOptions {
  repo: string
  is_cloned: boolean
  remembered: string | null
  bitbucket_default: string | null
  options: BaseBranchOption[]
  other_branch_count: number
}

export type SuggestionSignal =
  | "epic_siblings"
  | "explicit_hint"
  | "code_index"
  | "confluence"
  | "model"

export interface SuggestionReason {
  signal: SuggestionSignal
  weight: number
  strength: number
  text: string
}

export type SuggestionConfidence = "high" | "medium" | "low"

export type EvidenceSource = "acceptance_criteria" | "summary" | "description"

export interface RepositorySuggestion {
  repo: string
  score: number
  is_cloned: boolean
  bitbucket_project: string
  reasons: SuggestionReason[]
}

export interface RepositoryCatalogue {
  local_repository_count: number
  remote_repository_count: number
}

export interface RepositorySuggestions {
  key: string
  confidence: SuggestionConfidence
  suggestions: RepositorySuggestion[]
  catalogue?: RepositoryCatalogue
  signals_unavailable: { signal: string; reason: string }[]
  scope: WorkspaceScope
}

export interface RepositoryOption {
  repo: string
  project: string
  is_cloned: boolean
}

export interface RepositoryPage {
  repositories: RepositoryOption[]
  local_repository_count: number
  remote_repository_count: number
}

export interface BranchNameSuggestion {
  key: string
  branch: string
  category?: string
  core?: string
  source: "model" | "fallback" | "existing"
  evidence?: string
  evidence_source?: EvidenceSource
}

export interface CreateBranchTarget {
  repo: string
  base_branch: string
  remember_base: boolean
}

export interface CreateBranchesRequest {
  key: string
  branch: string
  targets: CreateBranchTarget[]
}

export interface CreatedBranch {
  repo: string
  base_branch: string
  head_commit: string
}

export interface FailedBranch {
  repo: string
  reason: string
}

export interface SkippedBranch {
  repo: string
  reason: string
}

export interface CreateBranchesResponse {
  key: string
  branch: string
  created: CreatedBranch[]
  failed: FailedBranch[]
  skipped: SkippedBranch[]
}

export type RepositoryFeedbackAction = "added" | "removed" | "unticked" | "kept"

export interface RepositoryFeedbackRequest {
  repo: string
  action: RepositoryFeedbackAction
}

export interface RepositoryFeedbackResponse {
  recorded: boolean
}
