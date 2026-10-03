export type WorkspaceSource = "helper" | "server"

export interface RepositorySummary {
  name: string
  is_git: boolean
  branch: string
  dirty_count: number
  updated_relative: string
  error: string
}

export interface ChangedFile {
  status: "modified" | "added" | "deleted" | "renamed" | "untracked"
  path: string
  added: number
  deleted: number
}

export interface RepositoryDetail {
  name: string
  branch: string
  updated_relative: string
  commit: { hash: string; author: string; relative_time: string; subject: string }
  ahead: number
  behind: number
  files: ChangedFile[]
  files_truncated: boolean
  error: string
}

export interface RepositoryListResponse {
  root_path: string
  repositories: RepositorySummary[]
}

export interface ServiceListResponse {
  root_path: string
  services: RepositoryDetail[]
}

export interface WorkspaceSettings {
  source: WorkspaceSource
  root_path: string
  helper_url: string
  has_helper_token: boolean
  selected: string[]
}

export interface WorkspaceSettingsInput {
  source: WorkspaceSource
  root_path: string
  helper_url: string
  helper_token?: string
}

export interface SelectionUpdateRequest {
  repos: string[]
}

export interface SelectionUpdateResponse {
  selected_count: number
}

export interface HelperHealth {
  status: string
  version: string
}

export interface HelperConnection {
  helperUrl: string
  helperToken: string
}

export interface ValidatePathRequest {
  root_path: string
}

export interface ValidatePathResponse {
  valid: boolean
  reason: string
}

export type HelperConnectionStatus = "unknown" | "checking" | "online" | "offline"
