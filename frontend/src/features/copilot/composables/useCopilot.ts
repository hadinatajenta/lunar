import { ref, watch } from "vue"
import type { ChatMessage, ChatSession, ReasoningEffort } from "../types"
import { allSupportedModels } from "../utils/models"
import { defaultDomains } from "../utils/domains"
import { deleteCopilotSession, sendCopilotChat } from "../api/copilot-api"
import { askCodeAgent, resetCodeAgentStream } from "../../code-context/composables/useCodeAgent"
import type { ChatHistoryTurn } from "../../code-context/types"
import { useProviderConfiguration } from "./useProviderConfiguration"
import {
  clearStoredKey,
  loadEnabledDomains,
  loadReasoningEffort,
  loadSavedSessions,
  loadSelectedModel,
  loadSessionMessages,
  loadSidebarCollapsed,
  loadThinkingMode,
  storeBoolean,
  storeJson,
  storeText
} from "../utils/copilot-storage"

const enabledDomains = ref<string[]>(loadEnabledDomains(defaultDomains.map((domain) => domain.id)))
const selectedModel = ref<string>(loadSelectedModel())
const thinkingMode = ref<boolean>(loadThinkingMode())
const reasoningEffort = ref<ReasoningEffort>(loadReasoningEffort())
const isSidebarCollapsed = ref<boolean>(loadSidebarCollapsed())
const isDomainModalOpen = ref(false)
const isLoading = ref(false)
const chatError = ref<string | null>(null)

const sessions = ref<ChatSession[]>(loadSavedSessions())
const activeSessionId = ref<string | null>(sessions.value.length > 0 ? sessions.value[0].id : null)
const historyTurnLimit = 10

const messages = ref<ChatMessage[]>(activeSessionId.value ? loadSessionMessages(activeSessionId.value) : [])

const { configuredProviders, isCheckingProviders, hasConfiguredAI, configuredModels, refreshConfiguredProviders } =
  useProviderConfiguration(selectedModel)

watch(enabledDomains, (storedDomains) => { storeJson("copilot_enabled_domains", storedDomains) }, { deep: true })
watch(selectedModel, (storedModel) => { storeText("copilot_selected_model", storedModel) })
watch(thinkingMode, (storedThinkingMode) => { storeBoolean("copilot_thinking_mode", storedThinkingMode) })
watch(reasoningEffort, (storedEffort) => { storeText("copilot_reasoning_effort", storedEffort) })
watch(isSidebarCollapsed, (storedCollapsed) => { storeBoolean("copilot_sidebar_collapsed", storedCollapsed) })
watch(sessions, (storedSessions) => { storeJson("copilot_user_sessions", storedSessions) }, { deep: true })

const formatClockTime = (): string => new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })

const resolveProviderForModel = (modelName: string): string =>
  allSupportedModels.find((model) => model.name === modelName || model.id === modelName)?.provider ?? "deepseek"

const persistMessages = (sessionId: string): void => {
  storeJson(`copilot_msgs_${sessionId}`, messages.value)
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
  const toggleSidebarCollapsed = () => {
    isSidebarCollapsed.value = !isSidebarCollapsed.value
  }

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

    sessions.value = sessions.value.filter((session) => session.id !== sessionId)
    clearStoredKey(`copilot_msgs_${sessionId}`)

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

  const buildHistory = (currentQuestion: string): ChatHistoryTurn[] => {
    const turns = messages.value
      .filter((message) => message.role === "user" || message.role === "assistant")
      .map((message) => ({ role: message.role as "user" | "assistant", content: message.content }))

    const lastTurn = turns[turns.length - 1]
    if (lastTurn !== undefined && lastTurn.role === "user" && lastTurn.content === currentQuestion) {
      turns.pop()
    }

    return turns.slice(-historyTurnLimit)
  }

  const requestAnswer = async (currentId: string, text: string): Promise<void> => {
    const agentAnswer = await askCodeAgent({
      sessionId: currentId,
      question: text,
      history: buildHistory(text),
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
      persistMessages(currentId)
      return
    }

    await requestBackendAnswer(currentId, text)
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
      await requestAnswer(currentId, text)
    } catch (err: unknown) {
      registerChatError(currentId, err)
    } finally {
      resetCodeAgentStream()
      isLoading.value = false
    }
  }

  const regenerateMessage = async (messageId: string): Promise<void> => {
    if (isLoading.value) return
    const index = messages.value.findIndex((message) => message.id === messageId)
    if (index < 1) return
    const question = messages.value[index - 1]
    if (question.role !== "user") return
    const currentId = activeSessionId.value
    if (currentId === null) return

    messages.value = messages.value.filter((message) => message.id !== messageId)
    isLoading.value = true

    try {
      await requestAnswer(currentId, question.content)
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
    regenerateMessage,
    activeSessionId,
    selectedModel,
    thinkingMode,
    reasoningEffort,
    isSidebarCollapsed,
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
    toggleSidebarCollapsed,
    toggleDomain,
    isDomainEnabled,
    enableAllDomains,
    startNewChat,
    selectSession,
    deleteSession,
    sendMessage
  }
}
