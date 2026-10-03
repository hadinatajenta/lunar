import type { AgentSource } from "./types"

export interface ActiveToolCall {
  tool: string
  action: string
  round: number
}

export interface CodeAgentRequest {
  sessionId: string
  question: string
  domainIds: string[]
  provider: string
  model: string
  thinkingMode: boolean
  reasoningEffort: string
}

export interface CodeAgentAnswer {
  answer: string
  reasoning: string
  sources: AgentSource[]
  durationMs: number
}
