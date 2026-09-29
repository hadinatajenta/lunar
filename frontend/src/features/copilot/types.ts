export type ReasoningEffort = "low" | "medium" | "high"

export interface SourceChip {
  type: "jira" | "bitbucket" | "confluence" | "servicemap" | "queryreview"
  label: string
}

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  reasoning?: string
  thinkingDurationMs?: number
  sources?: SourceChip[]
  createdAt: string
}

export interface ChatSession {
  id: string
  title: string
  model: string
  thinkingMode?: boolean
  reasoningEffort?: ReasoningEffort
  updatedAt: string
}

export interface ToolDomain {
  id: "jira" | "bitbucket" | "confluence" | "queryreview" | "servicemap"
  name: string
  description: string
  enabled: boolean
}
