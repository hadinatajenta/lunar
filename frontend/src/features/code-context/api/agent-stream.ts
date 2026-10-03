import { ApiError } from "../../../lib/http"
import { HelperConnectionError } from "../../microservices/api/helper-api"
import type { HelperConnection } from "../../microservices/types"
import type { AgentAskRequest, AgentEvent, AgentEventType, AgentSource } from "../types"
import {
  buildHelperEndpoint,
  createHelperHeaders,
  normalizeHelperUrl,
  resolveHelperErrorMessage
} from "./code-index-api"

const AGENT_ASK_PATH = "/agent/ask"
const AGENT_STREAM_DONE_MARKER = "[DONE]"
const HELPER_UNREACHABLE_MESSAGE =
  "The Lunar helper is not reachable. Start the helper on this machine and try again."
const HELPER_EMPTY_STREAM_MESSAGE = "The helper accepted the question but returned no stream."

const AGENT_EVENT_TYPES: AgentEventType[] = [
  "tool_call",
  "tool_result",
  "text",
  "reasoning",
  "sources",
  "done",
  "error"
]

export interface AgentStreamOptions {
  signal?: AbortSignal
  onEvent: (event: AgentEvent) => void
}

export class AgentStreamAbortedError extends Error {
  constructor() {
    super("The agent stream was aborted")
    this.name = "AgentStreamAbortedError"
  }
}

interface AgentStreamState {
  buffer: string
  isDone: boolean
}

const isAgentEventType = (value: unknown): value is AgentEventType =>
  typeof value === "string" && AGENT_EVENT_TYPES.some((eventType) => eventType === value)

const isAgentSource = (value: unknown): value is AgentSource => {
  if (typeof value !== "object" || value === null) {
    return false
  }
  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.repo === "string" &&
    typeof candidate.file === "string" &&
    typeof candidate.type === "string" &&
    typeof candidate.snippet === "string"
  )
}

const readStringField = (source: Record<string, unknown>, key: string): string | undefined => {
  const value = source[key]
  return typeof value === "string" ? value : undefined
}

const readNumberField = (source: Record<string, unknown>, key: string): number | undefined => {
  const value = source[key]
  return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

const readSourcesField = (source: Record<string, unknown>): AgentSource[] | undefined => {
  const value = source.sources
  if (!Array.isArray(value)) {
    return undefined
  }
  const sources = value.filter(isAgentSource)
  return sources.length > 0 ? sources : undefined
}

export const parseAgentEvent = (payload: string): AgentEvent | null => {
  let parsedPayload: unknown
  try {
    parsedPayload = JSON.parse(payload)
  } catch {
    return null
  }
  if (typeof parsedPayload !== "object" || parsedPayload === null) {
    return null
  }
  const candidate = parsedPayload as Record<string, unknown>
  if (!isAgentEventType(candidate.type)) {
    return null
  }
  return {
    type: candidate.type,
    text: readStringField(candidate, "text"),
    tool: readStringField(candidate, "tool"),
    action: readStringField(candidate, "action"),
    round: readNumberField(candidate, "round"),
    sources: readSourcesField(candidate),
    rounds: readNumberField(candidate, "rounds"),
    message: readStringField(candidate, "message")
  }
}

const consumeFrame = (frame: string, options: AgentStreamOptions): boolean => {
  const dataLines: string[] = []
  for (const line of frame.split("\n")) {
    if (line.startsWith("data:")) {
      dataLines.push(line.slice("data:".length).replace(/^ /, ""))
    }
  }
  if (dataLines.length === 0) {
    return false
  }
  const payload = dataLines.join("\n").trim()
  if (payload === AGENT_STREAM_DONE_MARKER) {
    return true
  }
  const agentEvent = parseAgentEvent(payload)
  if (agentEvent !== null) {
    options.onEvent(agentEvent)
  }
  return false
}

const drainStreamFrames = (state: AgentStreamState, options: AgentStreamOptions): void => {
  let separatorIndex = state.buffer.indexOf("\n\n")
  while (separatorIndex >= 0 && !state.isDone) {
    const frame = state.buffer.slice(0, separatorIndex)
    state.buffer = state.buffer.slice(separatorIndex + 2)
    state.isDone = consumeFrame(frame, options)
    separatorIndex = state.buffer.indexOf("\n\n")
  }
}

const readErrorPayload = async (response: Response): Promise<unknown> => {
  const contentType = response.headers.get("Content-Type") ?? ""
  if (contentType.includes("application/json")) {
    return response.json().catch(() => null)
  }
  return response.text().catch(() => null)
}

export const streamAgentAnswer = async (
  connection: HelperConnection,
  request: AgentAskRequest,
  options: AgentStreamOptions
): Promise<void> => {
  if (options.signal?.aborted) {
    throw new AgentStreamAbortedError()
  }

  const headers = createHelperHeaders(connection, "text/event-stream")
  headers.set("Content-Type", "application/json")

  let response: Response
  try {
    response = await fetch(buildHelperEndpoint(connection, AGENT_ASK_PATH), {
      method: "POST",
      headers,
      body: JSON.stringify(request),
      signal: options.signal
    })
  } catch (thrownError) {
    if (options.signal?.aborted || thrownError instanceof AgentStreamAbortedError) {
      throw new AgentStreamAbortedError()
    }
    throw new HelperConnectionError(HELPER_UNREACHABLE_MESSAGE, normalizeHelperUrl(connection.helperUrl))
  }

  if (!response.ok) {
    const errorPayload = await readErrorPayload(response)
    throw new ApiError(
      resolveHelperErrorMessage(response.status, errorPayload, response.statusText),
      response.status,
      errorPayload
    )
  }

  if (response.body === null) {
    throw new ApiError(HELPER_EMPTY_STREAM_MESSAGE, response.status)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  const state: AgentStreamState = { buffer: "", isDone: false }

  try {
    while (!state.isDone) {
      let readResult: Awaited<ReturnType<typeof reader.read>>
      try {
        readResult = await reader.read()
      } catch (readError) {
        if (options.signal?.aborted) {
          throw new AgentStreamAbortedError()
        }
        throw readError
      }
      if (readResult.done) {
        break
      }
      state.buffer = (state.buffer + decoder.decode(readResult.value, { stream: true })).replace(/\r\n/g, "\n")
      drainStreamFrames(state, options)
      if (options.signal?.aborted) {
        throw new AgentStreamAbortedError()
      }
    }

    if (!state.isDone) {
      state.buffer = (state.buffer + decoder.decode()).replace(/\r\n/g, "\n")
      drainStreamFrames(state, options)
      if (!state.isDone && state.buffer.trim().length > 0) {
        consumeFrame(state.buffer, options)
      }
    }
  } finally {
    reader.releaseLock()
  }
}
