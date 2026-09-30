import { computed, ref, watch } from "vue"
import { http } from "../../../lib/http"
import type { ChatMessage, ChatSession, ReasoningEffort, ToolDomain } from "../types"
import { allSupportedModels, getConfiguredModels, getDefaultModel, isModelConfigured } from "../utils/models"
import { deleteCopilotSession, sendCopilotChat } from "../api/copilot-api"

const defaultDomains: ToolDomain[] = [
  {
    id: "jira",
    name: "Jira BRI",
    description: "Search issues, fetch active sprint status and assigned tickets",
    enabled: true
  },
  {
    id: "bitbucket",
    name: "Bitbucket BRI",
    description: "Inspect pull requests, fetch git diffs and review commits",
    enabled: true
  },
  {
    id: "confluence",
    name: "Confluence BRI",
    description: "Preview technical specifications and architecture documents",
    enabled: true
  },
  {
    id: "queryreview",
    name: "Query Review Assistant",
    description: "Extract and compare database migration SQL statements",
    enabled: true
  },
  {
    id: "servicemap",
    name: "ServiceMap & Architecture",
    description: "Search 33 BRI MMS microservices and code dependencies",
    enabled: true
  }
]

const loadEnabledDomains = (): string[] => {
  try {
    const raw = localStorage.getItem("copilot_enabled_domains")
    if (raw) return JSON.parse(raw)
  } catch {
    return defaultDomains.map((d) => d.id)
  }
  return defaultDomains.map((d) => d.id)
}

const loadSavedSessions = (): ChatSession[] => {
  try {
    const raw = localStorage.getItem("copilot_user_sessions")
    if (raw) return JSON.parse(raw)
  } catch {
    return []
  }
  return []
}

const loadSessionMessages = (sessionId: string): ChatMessage[] => {
  try {
    const raw = localStorage.getItem(`copilot_msgs_${sessionId}`)
    if (raw) return JSON.parse(raw)
  } catch {
    return []
  }
  return []
}

const loadSelectedModel = (): string => {
  return localStorage.getItem("copilot_selected_model") || "DeepSeek-V4 Pro (Thinking)"
}

const loadThinkingMode = (): boolean => {
  const val = localStorage.getItem("copilot_thinking_mode")
  return val === null ? true : val === "true"
}

const loadReasoningEffort = (): ReasoningEffort => {
  const val = localStorage.getItem("copilot_reasoning_effort") as ReasoningEffort
  return val === "low" || val === "high" ? val : "medium"
}

const enabledDomains = ref<string[]>(loadEnabledDomains())
const selectedModel = ref<string>(loadSelectedModel())
const thinkingMode = ref<boolean>(loadThinkingMode())
const reasoningEffort = ref<ReasoningEffort>(loadReasoningEffort())
const isDomainModalOpen = ref(false)
const isLoading = ref(false)
const chatError = ref<string | null>(null)

const configuredProviders = ref<string[]>([])
const isCheckingProviders = ref(true)

const sessions = ref<ChatSession[]>(loadSavedSessions())
const activeSessionId = ref<string | null>(sessions.value.length > 0 ? sessions.value[0].id : null)
const messages = ref<ChatMessage[]>(activeSessionId.value ? loadSessionMessages(activeSessionId.value) : [])

watch(enabledDomains, (val) => {
  localStorage.setItem("copilot_enabled_domains", JSON.stringify(val))
}, { deep: true })

watch(selectedModel, (val) => {
  localStorage.setItem("copilot_selected_model", val)
})

watch(thinkingMode, (val) => {
  localStorage.setItem("copilot_thinking_mode", String(val))
})

watch(reasoningEffort, (val) => {
  localStorage.setItem("copilot_reasoning_effort", val)
})

watch(sessions, (val) => {
  localStorage.setItem("copilot_user_sessions", JSON.stringify(val))
}, { deep: true })

export const useCopilot = () => {
  const refreshConfiguredProviders = async () => {
    isCheckingProviders.value = true
    try {
      const [secretsResp, configResp] = await Promise.all([
        http.get<{ configured_ai_providers?: string[]; has_ai_keys?: boolean }>("/api/auth/secrets").catch(() => null),
        http.get<{ system_ai_providers?: string[] }>("/api/config").catch(() => null)
      ])

      const list = new Set<string>()
      if (secretsResp?.configured_ai_providers) {
        for (const p of secretsResp.configured_ai_providers) {
          list.add(p.toLowerCase())
        }
      }
      if (configResp?.system_ai_providers) {
        for (const p of configResp.system_ai_providers) {
          list.add(p.toLowerCase())
        }
      }

      configuredProviders.value = Array.from(list)

      if (!isModelConfigured(selectedModel.value, configuredProviders.value)) {
        selectedModel.value = getDefaultModel(configuredProviders.value)
      }
    } catch {
      configuredProviders.value = []
    } finally {
      isCheckingProviders.value = false
    }
  }

  const hasConfiguredAI = computed(() => configuredProviders.value.length > 0)

  const configuredModels = computed(() => {
    return getConfiguredModels(configuredProviders.value)
  })

  const toggleDomain = (domainId: string) => {
    const idx = enabledDomains.value.indexOf(domainId)
    if (idx >= 0) {
      enabledDomains.value = enabledDomains.value.filter((id) => id !== domainId)
    } else {
      enabledDomains.value = [...enabledDomains.value, domainId]
    }
  }

  const isDomainEnabled = (domainId: string) => {
    return enabledDomains.value.includes(domainId)
  }

  const startNewChat = () => {
    activeSessionId.value = null
    messages.value = []
    chatError.value = null
  }

  const selectSession = (sessionId: string) => {
    activeSessionId.value = sessionId
    messages.value = loadSessionMessages(sessionId)
    chatError.value = null
  }

  const deleteSession = async (sessionId: string) => {
    try {
      await deleteCopilotSession(sessionId).catch(() => null)
    } catch {
      return
    }

    sessions.value = sessions.value.filter((s) => s.id !== sessionId)
    localStorage.removeItem(`copilot_msgs_${sessionId}`)

    if (activeSessionId.value === sessionId) {
      if (sessions.value.length > 0) {
        selectSession(sessions.value[0].id)
      } else {
        startNewChat()
      }
    }
  }

  const sendMessage = async (prompt: string) => {
    const text = prompt.trim()
    if (!text || isLoading.value) return

    chatError.value = null

    if (!hasConfiguredAI.value) {
      chatError.value = "No AI provider configured: please set up an API key in Settings"
      return
    }

    let currentId = activeSessionId.value
    if (!currentId) {
      currentId = "session-" + Date.now()
      const title = text.length > 36 ? text.slice(0, 36) + "..." : text
      const newSession: ChatSession = {
        id: currentId,
        title,
        model: selectedModel.value,
        thinkingMode: thinkingMode.value,
        reasoningEffort: reasoningEffort.value,
        updatedAt: "Just now"
      }
      sessions.value = [newSession, ...sessions.value]
      activeSessionId.value = currentId
    }

    const userMsg: ChatMessage = {
      id: "u-" + Date.now(),
      role: "user",
      content: text,
      createdAt: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    }

    messages.value.push(userMsg)
    localStorage.setItem(`copilot_msgs_${currentId}`, JSON.stringify(messages.value))

    isLoading.value = true

    const activeTools = defaultDomains
      .filter((d) => enabledDomains.value.includes(d.id))
      .map((d) => d.name)

    try {
      const resp = await sendCopilotChat({
        session_id: currentId,
        model: selectedModel.value,
        prompt: text,
        thinking_mode: thinkingMode.value,
        reasoning_effort: reasoningEffort.value,
        active_tools: activeTools
      })

      const assistantMsg: ChatMessage = {
        id: resp.message.id || "a-" + Date.now(),
        role: "assistant",
        content: resp.message.content,
        reasoning: resp.message.reasoning,
        thinkingDurationMs: resp.message.thinkingDurationMs,
        sources: resp.message.sources && resp.message.sources.length > 0
          ? resp.message.sources
          : [
              { type: "jira", label: "Jira BRI · Active Issues" },
              { type: "bitbucket", label: "Bitbucket BRI · PRs & Diffs" }
            ],
        createdAt: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
      }

      messages.value.push(assistantMsg)
      if (currentId) {
        localStorage.setItem(`copilot_msgs_${currentId}`, JSON.stringify(messages.value))
      }
    } catch (err: unknown) {
      const errText = err instanceof Error ? err.message : "Failed to obtain AI response"
      chatError.value = errText

      const errorAssistantMsg: ChatMessage = {
        id: "err-" + Date.now(),
        role: "assistant",
        content: `Error: ${errText}`,
        createdAt: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
      }
      messages.value.push(errorAssistantMsg)
      if (currentId) {
        localStorage.setItem(`copilot_msgs_${currentId}`, JSON.stringify(messages.value))
      }
    } finally {
      isLoading.value = false
    }
  }

  return {
    messages,
    sessions,
    activeSessionId,
    selectedModel,
    thinkingMode,
    reasoningEffort,
    enabledDomains,
    defaultDomains,
    isDomainModalOpen,
    isLoading,
    chatError,
    configuredProviders,
    isCheckingProviders,
    hasConfiguredAI,
    configuredModels,
    allSupportedModels,
    refreshConfiguredProviders,
    toggleDomain,
    isDomainEnabled,
    startNewChat,
    selectSession,
    deleteSession,
    sendMessage
  }
}
