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

export interface BranchOption {
  name: string
  is_current: boolean
  updated_relative: string
}

export interface BranchListResponse {
  branches: BranchOption[]
}

export interface BranchCheckoutResponse {
  branch: string
}

export interface RepositoryFetchResponse {
  ahead: number
  behind: number
}

export interface RepositoryPullResponse {
  branch: string
  updated_relative: string
}

export interface RepositoryStashResponse {
  stash_count: number
}

export interface RepositoryDiscardResponse {
  restored_count: number
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

export type RepositoryActionKind = "checkout" | "fetch" | "pull" | "stash" | "discard"

export type RepositoryActionStatus = "idle" | "running" | "failed"

export interface RepositoryActionState {
  status: RepositoryActionStatus
  kind: RepositoryActionKind | null
  message: string
}

export type CollectionExportStage = "selecting" | "prompt" | "model" | "validating"

export interface CollectionExportRequest {
  root: string
  name: string
  model: string
  backend_url: string
  session_token: string
}

export interface CollectionExportStageEvent {
  stage: CollectionExportStage
  detail: string
}

export interface CollectionExportResult {
  service: string
  request_count: number
  folder_count: number
  collection: unknown
}

export interface CollectionExportHandlers {
  signal?: AbortSignal
  onStage: (stageEvent: CollectionExportStageEvent) => void
  onResult: (exportResult: CollectionExportResult) => void
}
