import type { ChatMessage, ChatSession, ReasoningEffort } from "../types"

const readStoredJson = <T>(key: string, fallback: T): T => {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

export const loadEnabledDomains = (fallbackIds: string[]): string[] =>
  readStoredJson("copilot_enabled_domains", fallbackIds)

export const loadSavedSessions = (): ChatSession[] => readStoredJson("copilot_user_sessions", [])

export const loadSessionMessages = (sessionId: string): ChatMessage[] =>
  readStoredJson(`copilot_msgs_${sessionId}`, [])

export const loadSelectedModel = (): string =>
  localStorage.getItem("copilot_selected_model") || "DeepSeek-V4 Pro (Thinking)"

export const loadReasoningEffort = (): ReasoningEffort => {
  const val = localStorage.getItem("copilot_reasoning_effort") as ReasoningEffort
  return val === "low" || val === "high" ? val : "medium"
}

export const loadSidebarCollapsed = (): boolean => localStorage.getItem("copilot_sidebar_collapsed") === "true"

export const storeBoolean = (key: string, value: boolean): void => {
  localStorage.setItem(key, String(value))
}

export const storeText = (key: string, value: string): void => {
  localStorage.setItem(key, value)
}

export const storeJson = (key: string, value: unknown): void => {
  localStorage.setItem(key, JSON.stringify(value))
}

export const clearStoredKey = (key: string): void => {
  localStorage.removeItem(key)
}
