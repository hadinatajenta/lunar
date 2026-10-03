export type ToolDomain = "servicemap" | "jira" | "bitbucket" | "queryreview" | "confluence"

export interface CodeIndexStatus {
  indexed: boolean
  is_building: boolean
  repo_count: number
  file_count: number
  chunk_count: number
  updated_at: string
  progress_percent: number
}

export interface CodeSearchResult {
  repo_name: string
  file_path: string
  chunk_type: string
  raw_content: string
  start_line?: number
  end_line?: number
  score?: number
}

export interface CodeSearchResponse {
  query: string
  results: CodeSearchResult[]
}

export interface AgentSource {
  repo: string
  file: string
  type: string
  snippet: string
  start_line?: number
  end_line?: number
}

export interface AgentAskRequest {
  question: string
  domains: ToolDomain[]
  provider: string
  model: string
  root: string
  backend_url: string
  session_token: string
  thinking_mode: boolean
  reasoning_effort: string
}

export type AgentEventType =
  | "tool_call"
  | "tool_result"
  | "text"
  | "reasoning"
  | "sources"
  | "done"
  | "error"

export interface AgentEvent {
  type: AgentEventType
  text?: string
  tool?: string
  action?: string
  round?: number
  sources?: AgentSource[]
  rounds?: number
  message?: string
}

export interface TranscriptRequest {
  session_id: string
  question: string
  answer: string
  reasoning: string
  sources: AgentSource[]
  duration_ms: number
}
