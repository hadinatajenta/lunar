import { ApiError, getStoredToken } from "../../../lib/http"
import type { CollectionExportHandlers, CollectionExportRequest, HelperConnection } from "../types"
import {
  createCollectionExportStreamState,
  dispatchExportFrame,
  drainExportFrames
} from "./collection-export-events"
import {
  HELPER_TOKEN_REJECTED_MESSAGE,
  HELPER_UNREACHABLE_MESSAGE,
  HelperConnectionError,
  buildHelperEndpoint,
  extractPayloadMessage,
  normalizeHelperUrl
} from "./helper-api"
import {
  COLLECTION_EXPORT_TIMEOUT_MS,
  RequestTimeoutError,
  withRequestSignalTimeout
} from "./request-timeout"

const EXPORT_PATH = "/export"
const WORKSPACE_EXPORT_PATH = "/api/workspace/export"

const BACKEND_SESSION_MESSAGE =
  "Your Lunar session is no longer valid. Sign in again, then retry the export."
const BACKEND_UNREACHABLE_MESSAGE =
  "The export request could not reach the Lunar backend. Check your connection and try again."
const EXPORT_EMPTY_STREAM_MESSAGE = "The export stream ended before the collection arrived. Try again."

const EXPORT_STATUS_MESSAGES = new Map<number, string>([
  [409, "No AI provider key is configured for this model. Add a provider key in Settings, then retry the export."],
  [422, "This repository cannot be exported as a Postman collection."],
  [504, "The AI provider did not answer in time. Try the export again."]
])

export class CollectionExportAbortedError extends Error {
  constructor() {
    super("The collection export was aborted")
    this.name = "CollectionExportAbortedError"
  }
}

const readExportStream = async (
  response: Response,
  handlers: CollectionExportHandlers
): Promise<void> => {
  if (response.body === null) {
    throw new ApiError(EXPORT_EMPTY_STREAM_MESSAGE, response.status)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  const state = createCollectionExportStreamState()

  try {
    while (!state.isDone) {
      const readResult = await reader.read()
      if (readResult.done) {
        break
      }
      state.buffer = (state.buffer + decoder.decode(readResult.value, { stream: true })).replace(/\r\n/g, "\n")
      drainExportFrames(state, handlers)
      if (handlers.signal?.aborted === true) {
        throw new CollectionExportAbortedError()
      }
    }

    if (!state.isDone) {
      state.buffer = (state.buffer + decoder.decode()).replace(/\r\n/g, "\n")
      drainExportFrames(state, handlers)
      if (!state.isDone && state.buffer.trim().length > 0) {
        state.isDone = dispatchExportFrame(state.buffer, handlers)
      }
    }
  } catch (thrownError) {
    if (thrownError instanceof CollectionExportAbortedError || thrownError instanceof ApiError) {
      throw thrownError
    }
    if (handlers.signal?.aborted === true) {
      throw new CollectionExportAbortedError()
    }
    throw thrownError
  } finally {
    reader.releaseLock()
  }

  if (!state.isDone) {
    throw new ApiError(EXPORT_EMPTY_STREAM_MESSAGE, 0)
  }
}

const readErrorPayload = async (response: Response): Promise<unknown> => {
  const contentType = response.headers.get("Content-Type") ?? ""
  if (contentType.includes("application/json")) {
    return response.json().catch(() => null)
  }
  return response.text().catch(() => null)
}

const resolveExportErrorMessage = (
  statusCode: number,
  payload: unknown,
  statusText: string,
  unauthorizedMessage: string
): string => {
  if (statusCode === 401) {
    return unauthorizedMessage
  }
  const payloadMessage = extractPayloadMessage(payload)
  if (payloadMessage !== null && payloadMessage.trim().length > 0) {
    return payloadMessage
  }
  const statusMessage = EXPORT_STATUS_MESSAGES.get(statusCode)
  if (statusMessage !== undefined) {
    return statusMessage
  }
  return statusText.trim().length > 0 ? statusText : `The export failed with status ${statusCode}`
}

const createExportHeaders = (authorizationToken: string): Headers => {
  const headers = new Headers()
  headers.set("Accept", "text/event-stream")
  headers.set("Content-Type", "application/json")
  if (authorizationToken.length > 0) {
    headers.set("Authorization", `Bearer ${authorizationToken}`)
  }
  return headers
}

const streamCollectionExport = async (
  endpoint: string,
  headers: Headers,
  request: CollectionExportRequest,
  handlers: CollectionExportHandlers,
  unauthorizedMessage: string
): Promise<void> => {
  if (handlers.signal?.aborted === true) {
    throw new CollectionExportAbortedError()
  }

  await withRequestSignalTimeout(COLLECTION_EXPORT_TIMEOUT_MS, handlers.signal, async (signal) => {
    let response: Response
    try {
      response = await fetch(endpoint, {
        method: "POST",
        headers,
        body: JSON.stringify(request),
        signal
      })
    } catch (thrownError) {
      if (handlers.signal?.aborted === true) {
        throw new CollectionExportAbortedError()
      }
      throw thrownError
    }

    if (!response.ok) {
      const errorPayload = await readErrorPayload(response)
      throw new ApiError(
        resolveExportErrorMessage(response.status, errorPayload, response.statusText, unauthorizedMessage),
        response.status,
        errorPayload
      )
    }

    await readExportStream(response, handlers)
  })
}

const isResolvedExportFailure = (thrownError: unknown): boolean =>
  thrownError instanceof ApiError ||
  thrownError instanceof CollectionExportAbortedError ||
  thrownError instanceof RequestTimeoutError

export const exportHelperCollection = async (
  connection: HelperConnection,
  request: CollectionExportRequest,
  handlers: CollectionExportHandlers
): Promise<void> => {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)
  try {
    await streamCollectionExport(
      buildHelperEndpoint(connection, EXPORT_PATH),
      createExportHeaders(connection.helperToken),
      request,
      handlers,
      HELPER_TOKEN_REJECTED_MESSAGE
    )
  } catch (thrownError) {
    if (isResolvedExportFailure(thrownError)) {
      throw thrownError
    }
    throw new HelperConnectionError(HELPER_UNREACHABLE_MESSAGE, helperUrl)
  }
}

export const exportWorkspaceCollection = async (
  request: CollectionExportRequest,
  handlers: CollectionExportHandlers
): Promise<void> => {
  try {
    await streamCollectionExport(
      WORKSPACE_EXPORT_PATH,
      createExportHeaders(getStoredToken() ?? ""),
      request,
      handlers,
      BACKEND_SESSION_MESSAGE
    )
  } catch (thrownError) {
    if (isResolvedExportFailure(thrownError)) {
      throw thrownError
    }
    throw new ApiError(BACKEND_UNREACHABLE_MESSAGE, 0, thrownError)
  }
}
