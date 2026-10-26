import { ApiError } from "../../../lib/http"
import { buildHelperEndpoint, normalizeHelperUrl, resolveHelperErrorMessage } from "../../code-context/api/code-index-api"
import {
  HELPER_UNREACHABLE_MESSAGE,
  HelperConnectionError
} from "../../microservices/api/helper-api"
import type { HelperConnection } from "../../microservices/types"
import {
  createUtRunStreamState,
  drainUtRunFrames,
  dispatchUtRunFrame
} from "./ut-run-events"
import type { UtRunEvent, UtRunStreamHandlers } from "./ut-run-events"

const UT_RUN_PATH = "/ut-runs"
const UT_RUN_EMPTY_STREAM_MESSAGE = "The helper accepted the run but returned no stream."
const UT_RUN_INCOMPLETE_STREAM_MESSAGE =
  "The run stream ended before the result arrived. Check the helper and run the tests again."

export interface HelperUtRunRequest {
  root: string
  repo: string
  branch: string
  base_ref?: string
}

export interface UtRunStreamOptions {
  signal?: AbortSignal
  onEvent: (event: UtRunEvent) => void
}

const createUtRunHeaders = (connection: HelperConnection): Headers => {
  const headers = new Headers()
  headers.set("Accept", "text/event-stream")
  headers.set("Content-Type", "application/json")
  if (connection.helperToken.length > 0) {
    headers.set("Authorization", `Bearer ${connection.helperToken}`)
  }
  return headers
}

const readErrorPayload = async (response: Response): Promise<unknown> => {
  const contentType = response.headers.get("Content-Type") ?? ""
  if (contentType.includes("application/json")) {
    return response.json().catch(() => null)
  }
  return response.text().catch(() => null)
}

export const streamUtRun = async (
  connection: HelperConnection,
  request: HelperUtRunRequest,
  options: UtRunStreamOptions
): Promise<void> => {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)

  let response: Response
  try {
    response = await fetch(buildHelperEndpoint(connection, UT_RUN_PATH), {
      method: "POST",
      headers: createUtRunHeaders(connection),
      body: JSON.stringify(request),
      signal: options.signal
    })
  } catch (thrownError) {
    if (options.signal?.aborted === true) {
      throw thrownError
    }
    throw new HelperConnectionError(HELPER_UNREACHABLE_MESSAGE, helperUrl)
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
    throw new ApiError(UT_RUN_EMPTY_STREAM_MESSAGE, response.status)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  const state = createUtRunStreamState()
  const handlers: UtRunStreamHandlers = {
    onPhase: (phase) => options.onEvent({ kind: "phase", phase }),
    onOutput: (text) => options.onEvent({ kind: "output", text }),
    onDone: (result) => options.onEvent({ kind: "done", result })
  }

  try {
    while (!state.isDone) {
      const readResult = await reader.read()
      if (readResult.done) {
        break
      }
      state.buffer = (state.buffer + decoder.decode(readResult.value, { stream: true })).replace(/\r\n/g, "\n")
      drainUtRunFrames(state, handlers)
      if (options.signal?.aborted === true) {
        throw new ApiError(UT_RUN_INCOMPLETE_STREAM_MESSAGE, 0)
      }
    }

    if (!state.isDone) {
      state.buffer = (state.buffer + decoder.decode()).replace(/\r\n/g, "\n")
      drainUtRunFrames(state, handlers)
      if (!state.isDone && state.buffer.trim().length > 0) {
        state.isDone = dispatchUtRunFrame(state.buffer, handlers)
      }
    }
  } finally {
    reader.releaseLock()
  }

  if (!state.isDone) {
    throw new ApiError(UT_RUN_INCOMPLETE_STREAM_MESSAGE, 0)
  }
}

export const fetchUtScreenshotBlob = async (
  connection: HelperConnection,
  screenshotPath: string,
  signal?: AbortSignal
): Promise<Blob> => {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)
  const headers = new Headers()
  headers.set("Accept", "image/png")
  if (connection.helperToken.length > 0) {
    headers.set("Authorization", `Bearer ${connection.helperToken}`)
  }

  let response: Response
  try {
    response = await fetch(buildHelperEndpoint(connection, screenshotPath), {
      method: "GET",
      headers,
      signal
    })
  } catch (thrownError) {
    if (signal?.aborted === true) {
      throw thrownError
    }
    throw new HelperConnectionError(HELPER_UNREACHABLE_MESSAGE, helperUrl)
  }

  if (!response.ok) {
    const errorPayload = await readErrorPayload(response)
    throw new ApiError(
      resolveHelperErrorMessage(response.status, errorPayload, response.statusText),
      response.status,
      errorPayload
    )
  }

  return response.blob()
}

export const fetchUtScreenshotObjectUrl = async (
  connection: HelperConnection,
  screenshotPath: string,
  signal?: AbortSignal
): Promise<string> => {
  const screenshotBlob = await fetchUtScreenshotBlob(connection, screenshotPath, signal)
  return URL.createObjectURL(screenshotBlob)
}
