import { http } from "../../../lib/http"
import type { ChatMessage, ChatSession, ModelOption, ReasoningEffort } from "../types"

export interface SendChatMessagePayload {
  session_id?: string
  model: string
  prompt: string
  thinking_mode?: boolean
  reasoning_effort?: ReasoningEffort
  active_tools?: string[]
}

export interface SendChatMessageResponse {
  session_id: string
  message: ChatMessage
}

export async function fetchCopilotModels(): Promise<ModelOption[]> {
  return http.get<ModelOption[]>("/api/copilot/models")
}

export async function sendCopilotChat(
  payload: SendChatMessagePayload
): Promise<SendChatMessageResponse> {
  return http.post<SendChatMessageResponse>("/api/copilot/chat", payload)
}

export async function fetchCopilotSessions(): Promise<ChatSession[]> {
  return http.get<ChatSession[]>("/api/copilot/sessions")
}

export async function deleteCopilotSession(
  sessionId: string
): Promise<{ status: string }> {
  return http.delete<{ status: string }>(`/api/copilot/sessions/${encodeURIComponent(sessionId)}`)
}

export async function fetchCopilotMessages(
  sessionId: string
): Promise<ChatMessage[]> {
  return http.get<ChatMessage[]>(`/api/copilot/sessions/${encodeURIComponent(sessionId)}/messages`)
}
