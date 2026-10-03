import { computed, ref, watch } from "vue"
import { http } from "../../../lib/http"
import type { ChatMessage, ChatSession, ReasoningEffort, ToolDomain } from "../types"
import { allSupportedModels, getConfiguredModels, getDefaultModel, isModelConfigured } from "../utils/models"
import { deleteCopilotSession, sendCopilotChat } from "../api/copilot-api"
import { askCodeAgent, resetCodeAgentStream, storeAnswerCitations } from "../../code-context/composables/useCodeAgent"

const defaultDomains: ToolDomain[] = [
  { id: "jira", name: "Jira BRI", description: "Search issues, fetch active sprint status and assigned tickets", enabled: true },
  { id: "bitbucket", name: "Bitbucket BRI", description: "Inspect pull requests, fetch git diffs and review commits", enabled: true },
  { id: "confluence", name: "Confluence BRI", description: "Preview technical specifications and architecture documents", enabled: true },
  { id: "queryreview", name: "Query Review Assistant", description: "Extract SQL statements from query review documents", enabled: true },
  { id: "servicemap", name: "ServiceMap & Architecture", description: "Inspect workspace microservices, branches and changed files", enabled: true }
]

const loadStoredJson = <T>(key: string, fallback: T): T => {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

const loadEnabledDomains = (): string[] => loadStoredJson("copilot_enabled_domains", defaultDomains.map((d) => d.id))

const loadSavedSessions = (): ChatSession[] => loadStoredJson("copilot_user_sessions", [])

const loadSessionMessages = (sessionId: string): ChatMessage[] => loadStoredJson(`copilot_msgs_${sessionId}`, [])

const loadSelectedModel = (): string =>
  localStorage.getItem("copilot_selected_model") || "DeepSeek-V4 Pro (Thinking)"

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

watch(enabledDomains, (val) => { localStorage.setItem("copilot_enabled_domains", JSON.stringify(val)) }, { deep: true })
watch(selectedModel, (val) => { localStorage.setItem("copilot_selected_model", val) })
watch(thinkingMode, (val) => { localStorage.setItem("copilot_thinking_mode", String(val)) })
watch(reasoningEffort, (val) => { localStorage.setItem("copilot_reasoning_effort", val) })
watch(sessions, (val) => { localStorage.setItem("copilot_user_sessions", JSON.stringify(val)) }, { deep: true })

const collectProviderNames = (...sources: Array<string[] | undefined>): string[] =>
  Array.from(new Set(sources.flatMap((source) => source ?? []).map((name) => name.toLowerCase())))

const formatClockTime = (): string => new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })

const resolveProviderForModel = (modelName: string): string =>
  allSupportedModels.find((model) => model.name === modelName || model.id === modelName)?.provider ?? "deepseek"

const persistMessages = (sessionId: string): void => {
  localStorage.setItem(`copilot_msgs_${sessionId}`, JSON.stringify(messages.value))
}

const ensureActiveSession = (text: string): string => {
  if (activeSessionId.value !== null) {
    return activeSessionId.value
  }
  const sessionId = "session-" + Date.now()
  const title = text.length > 36 ? text.slice(0, 36) + "..." : text
  const newSession: ChatSession = {
    id: sessionId,
    title,
    model: selectedModel.value,
    thinkingMode: thinkingMode.value,
    reasoningEffort: reasoningEffort.value,
    updatedAt: "Just now"
  }
  sessions.value = [newSession, ...sessions.value]
  activeSessionId.value = sessionId
  return sessionId
}

export const useCopilot = () => {
  const refreshConfiguredProviders = async () => {
    isCheckingProviders.value = true
    try {
      const [secretsResp, configResp] = await Promise.all([
        http.get<{ configured_ai_providers?: string[]; has_ai_keys?: boolean }>("/api/auth/secrets").catch(() => null),
        http.get<{ system_ai_providers?: string[] }>("/api/config").catch(() => null)
      ])

      configuredProviders.value = collectProviderNames(secretsResp?.configured_ai_providers, configResp?.system_ai_providers)

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

  const configuredModels = computed(() => getConfiguredModels(configuredProviders.value))

  const toggleDomain = (domainId: string) => {
    const idx = enabledDomains.value.indexOf(domainId)
    if (idx >= 0) {
      enabledDomains.value = enabledDomains.value.filter((id) => id !== domainId)
    } else {
      enabledDomains.value = [...enabledDomains.value, domainId]
    }
  }

  const isDomainEnabled = (domainId: string) => enabledDomains.value.includes(domainId)

  const enableAllDomains = () => { enabledDomains.value = defaultDomains.map((d) => d.id) }

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
      if (sessions.value.length > 0) selectSession(sessions.value[0].id)
      else startNewChat()
    }
  }

  const requestBackendAnswer = async (sessionId: string, text: string) => {
    const activeTools = defaultDomains
      .filter((d) => enabledDomains.value.includes(d.id))
      .map((d) => d.name)

    const resp = await sendCopilotChat({
      session_id: sessionId,
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
      createdAt: formatClockTime()
    }

    messages.value.push(assistantMsg)
    persistMessages(sessionId)
  }

  const registerChatError = (sessionId: string, thrownError: unknown) => {
    const errText = thrownError instanceof Error ? thrownError.message : "Failed to obtain AI response"
    chatError.value = errText

    const errorAssistantMsg: ChatMessage = {
      id: "err-" + Date.now(),
      role: "assistant",
      content: `Error: ${errText}`,
      createdAt: formatClockTime()
    }
    messages.value.push(errorAssistantMsg)
    persistMessages(sessionId)
  }

  const sendMessage = async (prompt: string) => {
    const text = prompt.trim()
    if (!text || isLoading.value) return

    chatError.value = null

    if (!hasConfiguredAI.value) {
      chatError.value = "No AI provider configured: please set up an API key in Settings"
      return
    }

    const currentId = ensureActiveSession(text)

    const userMsg: ChatMessage = {
      id: "u-" + Date.now(),
      role: "user",
      content: text,
      createdAt: formatClockTime()
    }

    messages.value.push(userMsg)
    persistMessages(currentId)

    isLoading.value = true

    try {
      const agentAnswer = await askCodeAgent({
        sessionId: currentId,
        question: text,
        domainIds: enabledDomains.value,
        provider: resolveProviderForModel(selectedModel.value),
        model: selectedModel.value,
        thinkingMode: thinkingMode.value,
        reasoningEffort: reasoningEffort.value
      })

      if (agentAnswer !== null) {
        const assistantMsg: ChatMessage = {
          id: "a-" + Date.now(),
          role: "assistant",
          content: agentAnswer.answer,
          reasoning: agentAnswer.reasoning.length > 0 ? agentAnswer.reasoning : undefined,
          thinkingDurationMs: agentAnswer.durationMs,
          createdAt: formatClockTime()
        }

        messages.value.push(assistantMsg)
        storeAnswerCitations(assistantMsg.id, agentAnswer.sources)
        persistMessages(currentId)
        return
      }

      await requestBackendAnswer(currentId, text)
    } catch (err: unknown) {
      registerChatError(currentId, err)
    } finally {
      resetCodeAgentStream()
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
    enableAllDomains,
    startNewChat,
    selectSession,
    deleteSession,
    sendMessage
  }
}
