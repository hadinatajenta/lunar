import { computed, ref } from "vue"
import { getStoredToken, http } from "../../../lib/http"
import {
  HELPER_TOKEN_MISSING_MESSAGE,
  REPOSITORY_ROOT_MISSING_MESSAGE,
  resolveErrorMessage,
  resolveHelperConnection,
  selectedNames,
  workspace
} from "../../microservices/composables/microservices-store"
import { useMicroservices } from "../../microservices/composables/useMicroservices"
import {
  AGENT_FALLBACK_ERROR,
  EMPTY_ANSWER_ERROR,
  HELPER_OFFLINE_HINT,
  INDEX_BUILDING_HINT,
  INDEX_BUILD_FALLBACK_ERROR,
  INDEX_EMPTY_HINT,
  INDEX_MISSING_HINT,
  INDEX_STATUS_FALLBACK_ERROR,
  SUPPORTED_TOOL_DOMAINS,
  TRANSCRIPT_FALLBACK_ERROR
} from "../hints"
import { fetchCodeIndexStatus, startCodeIndexBuild } from "../api/code-index-api"
import { resolveBackendURL } from "../api/backend-url"
import { AgentStreamAbortedError, streamAgentAnswer } from "../api/agent-stream"
import type { ActiveToolCall, CodeAgentAnswer, CodeAgentRequest } from "../agent-types"
import type {
  AgentAskRequest,
  AgentEvent,
  AgentSource,
  CodeIndexStatus,
  ToolDomain,
  TranscriptRequest
} from "../types"

const { connectHelper, loadWorkspace } = useMicroservices()

const streamingText = ref("")
const streamingReasoning = ref("")
const streamingSources = ref<AgentSource[]>([])
const activeToolCall = ref<ActiveToolCall | null>(null)
const streamError = ref<string | null>(null)
const isStreaming = ref(false)
const transcriptError = ref<string | null>(null)
const answerCitations = ref<Record<string, AgentSource[]>>({})

const indexStatus = ref<CodeIndexStatus | null>(null)
const indexStatusError = ref<string | null>(null)
const indexHint = ref<string | null>(null)
const isIndexStatusLoading = ref(false)
const isIndexBuildStarting = ref(false)
const isIndexBuilding = computed(() => indexStatus.value?.is_building ?? false)

export const resolveToolDomains = (domainIds: string[]): ToolDomain[] =>
  SUPPORTED_TOOL_DOMAINS.filter((domain) => domainIds.includes(domain))

const resolveSessionApiKey = (): string => getStoredToken() ?? ""

const hasConfiguredHelperUrl = (): boolean => (workspace.value?.helper_url.trim() ?? "").length > 0

const resolveConnectionWithWorkspace = async () => {
  if (workspace.value === null) await loadWorkspace()

  const storedConnection = resolveHelperConnection()
  if (storedConnection !== null && storedConnection.helperToken.length > 0) {
    return storedConnection
  }

  const isConnected = await connectHelper()
  if (!isConnected) return null
  return resolveHelperConnection()
}

const resolveIndexReadinessHint = (status: CodeIndexStatus): string | null => {
  if (status.is_building) return INDEX_BUILDING_HINT
  if (!status.indexed) return INDEX_MISSING_HINT
  if (status.chunk_count === 0) return INDEX_EMPTY_HINT
  return null
}

const refreshIndexStatus = async (): Promise<CodeIndexStatus | null> => {
  const connection = await resolveConnectionWithWorkspace()
  if (connection === null) {
    indexStatus.value = null
    indexStatusError.value = HELPER_TOKEN_MISSING_MESSAGE
    return null
  }

  isIndexStatusLoading.value = true
  try {
    const status = await fetchCodeIndexStatus(connection)
    indexStatus.value = status
    indexStatusError.value = null
    if (!status.is_building) {
      indexHint.value = resolveIndexReadinessHint(status)
    }
    return status
  } catch (thrownError) {
    indexStatus.value = null
    indexStatusError.value = resolveErrorMessage(thrownError, INDEX_STATUS_FALLBACK_ERROR)
    return null
  } finally {
    isIndexStatusLoading.value = false
  }
}

const startIndexBuild = async (): Promise<boolean> => {
  const connection = await resolveConnectionWithWorkspace()
  if (connection === null) {
    indexStatusError.value = HELPER_TOKEN_MISSING_MESSAGE
    return false
  }

  const rootPath = workspace.value?.root_path.trim() ?? ""
  if (rootPath.length === 0) {
    indexStatusError.value = REPOSITORY_ROOT_MISSING_MESSAGE
    return false
  }

  isIndexBuildStarting.value = true
  try {
    await startCodeIndexBuild(connection, {
      root: rootPath,
      repos: [...selectedNames.value],
      rebuild: false
    })
    indexHint.value = null
    await refreshIndexStatus()
    return true
  } catch (thrownError) {
    indexStatusError.value = resolveErrorMessage(thrownError, INDEX_BUILD_FALLBACK_ERROR)
    return false
  } finally {
    isIndexBuildStarting.value = false
  }
}

export const resetCodeAgentStream = (): void => {
  streamingText.value = ""
  streamingReasoning.value = ""
  streamingSources.value = []
  activeToolCall.value = null
  streamError.value = null
}

export const storeAnswerCitations = (messageId: string, sources: AgentSource[]): void => {
  answerCitations.value = { ...answerCitations.value, [messageId]: sources }
}

const appendStreamingSource = (source: AgentSource): void => {
  const isKnown = streamingSources.value.some(
    (existing) =>
      existing.repo === source.repo && existing.file === source.file && existing.type === source.type
  )
  if (!isKnown) {
    streamingSources.value = [...streamingSources.value, source]
  }
}

const applyAgentEvent = (event: AgentEvent): void => {
  if (event.type === "text") {
    streamingText.value += event.text ?? ""
    return
  }
  if (event.type === "reasoning") {
    streamingReasoning.value += event.text ?? ""
    return
  }
  if (event.type === "sources") {
    for (const source of event.sources ?? []) {
      appendStreamingSource(source)
    }
    return
  }
  if (event.type === "tool_call") {
    activeToolCall.value = {
      tool: event.tool ?? "tool",
      action: event.action ?? "",
      round: event.round ?? 0
    }
    return
  }
  if (event.type === "tool_result") {
    activeToolCall.value = null
    return
  }
  if (event.type === "error") {
    streamError.value = event.message ?? AGENT_FALLBACK_ERROR
  }
}

const postTranscript = async (request: TranscriptRequest): Promise<void> => {
  try {
    await http.post<{ message_id: string }>("/api/copilot/transcript", request)
    transcriptError.value = null
  } catch (thrownError) {
    transcriptError.value = resolveErrorMessage(thrownError, TRANSCRIPT_FALLBACK_ERROR)
  }
}

export const askCodeAgent = async (request: CodeAgentRequest): Promise<CodeAgentAnswer | null> => {
  resetCodeAgentStream()

  const connection = await resolveConnectionWithWorkspace()
  if (connection === null) {
    indexHint.value = hasConfiguredHelperUrl() ? HELPER_OFFLINE_HINT : null
    return null
  }

  const status = await refreshIndexStatus()
  if (status === null) {
    indexHint.value = HELPER_OFFLINE_HINT
    return null
  }

  const readinessHint = resolveIndexReadinessHint(status)
  if (readinessHint !== null) {
    indexHint.value = readinessHint
    return null
  }
  indexHint.value = null

  const agentRequest: AgentAskRequest = {
    question: request.question,
    domains: resolveToolDomains(request.domainIds),
    provider: request.provider,
    model: request.model,
    root: workspace.value?.root_path.trim() ?? "",
    backend_url: resolveBackendURL(),
    session_token: resolveSessionApiKey(),
    thinking_mode: request.thinkingMode,
    reasoning_effort: request.reasoningEffort
  }

  isStreaming.value = true
  const startedAt = Date.now()
  try {
    await streamAgentAnswer(connection, agentRequest, { onEvent: applyAgentEvent })
    if (streamError.value !== null) {
      indexHint.value = streamError.value
      return null
    }
    const answer = streamingText.value.trim()
    if (answer.length === 0) {
      streamError.value = EMPTY_ANSWER_ERROR
      indexHint.value = EMPTY_ANSWER_ERROR
      return null
    }
    const durationMs = Date.now() - startedAt
    const sources = [...streamingSources.value]
    await postTranscript({
      session_id: request.sessionId,
      question: request.question,
      answer,
      reasoning: streamingReasoning.value,
      sources,
      duration_ms: durationMs
    })
    return { answer, reasoning: streamingReasoning.value, sources, durationMs }
  } catch (thrownError) {
    if (thrownError instanceof AgentStreamAbortedError) {
      return null
    }
    streamError.value = resolveErrorMessage(thrownError, AGENT_FALLBACK_ERROR)
    indexHint.value = streamError.value
    return null
  } finally {
    activeToolCall.value = null
    isStreaming.value = false
  }
}

export function useCodeAgent() {
  return {
    streamingText,
    streamingReasoning,
    streamingSources,
    activeToolCall,
    streamError,
    isStreaming,
    transcriptError,
    answerCitations,
    indexStatus,
    indexStatusError,
    indexHint,
    isIndexStatusLoading,
    isIndexBuildStarting,
    isIndexBuilding,
    refreshIndexStatus,
    startIndexBuild,
    askCodeAgent,
    resetCodeAgentStream,
    storeAnswerCitations
  }
}
