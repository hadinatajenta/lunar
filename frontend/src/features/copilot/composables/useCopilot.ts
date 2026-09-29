import { ref, watch } from "vue"
import type { ChatMessage, ChatSession, ReasoningEffort, ToolDomain } from "../types"

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
  return localStorage.getItem("copilot_selected_model") || "Claude Opus 5.5 (Adaptive Thinking)"
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
  }

  const selectSession = (sessionId: string) => {
    activeSessionId.value = sessionId
    messages.value = loadSessionMessages(sessionId)
  }

  const deleteSession = (sessionId: string) => {
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

  const formatThinkingTrace = (prompt: string, effort: ReasoningEffort, tools: string[]): string => {
    const effortPrefix = effort === "high"
      ? "Deep analysis initiated with high token allocation.\n"
      : effort === "low"
      ? "Fast reasoning pass selected for low latency response.\n"
      : "Standard chain-of-thought reasoning activated.\n"

    return `${effortPrefix}1. Parsing intent from user prompt: "${prompt.slice(0, 80)}${prompt.length > 80 ? "..." : ""}".
2. Checking active tool integrations: [${tools.join(", ")}].
3. Querying local knowledge graphs for repository schema and linked Atlassian entities.
4. Synthesizing cross-system evidence and validating data consistency across active endpoints.
5. Formulating structured, verified response.`
  }

  const sendMessage = async (prompt: string) => {
    const text = prompt.trim()
    if (!text || isLoading.value) return

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

    const reasoningTrace = thinkingMode.value
      ? formatThinkingTrace(text, reasoningEffort.value, activeTools)
      : undefined

    const latency = thinkingMode.value ? (reasoningEffort.value === "high" ? 1100 : 700) : 400

    setTimeout(() => {
      const assistantMsg: ChatMessage = {
        id: "a-" + Date.now(),
        role: "assistant",
        content: `Processed your inquiry using ${selectedModel.value}.\n\nActive workspace tools queried: ${activeTools.join(", ")}.\n\nAll requested contexts have been verified and integrated into your current session.`,
        reasoning: reasoningTrace,
        thinkingDurationMs: thinkingMode.value ? (reasoningEffort.value === "high" ? 2840 : 1420) : undefined,
        sources: [
          { type: "jira", label: "Jira · Sprint Scope" },
          { type: "bitbucket", label: "Bitbucket · Active PRs" }
        ],
        createdAt: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
      }

      messages.value.push(assistantMsg)
      if (currentId) {
        localStorage.setItem(`copilot_msgs_${currentId}`, JSON.stringify(messages.value))
      }
      isLoading.value = false
    }, latency)
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
    toggleDomain,
    isDomainEnabled,
    startNewChat,
    selectSession,
    deleteSession,
    sendMessage
  }
}
