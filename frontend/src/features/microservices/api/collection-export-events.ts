import { ApiError } from "../../../lib/http"
import type {
  CollectionExportHandlers,
  CollectionExportResult,
  CollectionExportStage,
  CollectionExportStageEvent
} from "../types"
import { extractPayloadMessage } from "./helper-api"

const STAGE_EVENT_NAME = "stage"
const RESULT_EVENT_NAME = "result"
const ERROR_EVENT_NAME = "error"
const EXPORT_STAGES: CollectionExportStage[] = ["selecting", "prompt", "model", "validating"]

const EXPORT_UNREADABLE_RESULT_MESSAGE = "The export result could not be read. Try again."
const EXPORT_FAILED_MESSAGE = "The export failed before it produced a collection."

export interface CollectionExportStreamState {
  buffer: string
  isDone: boolean
}

interface ExportFrame {
  eventName: string
  payload: string
}

export const createCollectionExportStreamState = (): CollectionExportStreamState => ({
  buffer: "",
  isDone: false
})

const parseJsonObject = (payload: string): Record<string, unknown> | null => {
  try {
    const parsedPayload: unknown = JSON.parse(payload)
    if (typeof parsedPayload !== "object" || parsedPayload === null || Array.isArray(parsedPayload)) {
      return null
    }
    return parsedPayload as Record<string, unknown>
  } catch {
    return null
  }
}

const parseExportFrame = (frame: string): ExportFrame | null => {
  let eventName = ""
  const payloadLines: string[] = []
  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) {
      eventName = line.slice("event:".length).trim()
      continue
    }
    if (line.startsWith("data:")) {
      payloadLines.push(line.slice("data:".length).replace(/^ /, ""))
    }
  }
  if (payloadLines.length === 0) {
    return null
  }
  return { eventName, payload: payloadLines.join("\n").trim() }
}

const isCollectionExportStage = (candidate: unknown): candidate is CollectionExportStage =>
  typeof candidate === "string" && EXPORT_STAGES.some((stage) => stage === candidate)

const parseStageEvent = (payload: string): CollectionExportStageEvent | null => {
  const parsedPayload = parseJsonObject(payload)
  if (parsedPayload === null) {
    return null
  }
  const stageCandidate = parsedPayload.stage
  if (!isCollectionExportStage(stageCandidate)) {
    return null
  }
  return {
    stage: stageCandidate,
    detail: typeof parsedPayload.detail === "string" ? parsedPayload.detail : ""
  }
}

const parseResultEvent = (payload: string): CollectionExportResult | null => {
  const parsedPayload = parseJsonObject(payload)
  if (parsedPayload === null) {
    return null
  }
  const requestCount = parsedPayload.request_count
  const folderCount = parsedPayload.folder_count
  if (typeof requestCount !== "number" || typeof folderCount !== "number") {
    return null
  }
  if (parsedPayload.collection === undefined) {
    return null
  }
  return {
    service: typeof parsedPayload.service === "string" ? parsedPayload.service : "",
    request_count: requestCount,
    folder_count: folderCount,
    collection: parsedPayload.collection
  }
}

const readPayloadMessage = (payload: string, fallbackMessage: string): string => {
  const parsedPayload = parseJsonObject(payload)
  const payloadMessage = parsedPayload === null ? null : extractPayloadMessage(parsedPayload)
  if (payloadMessage === null || payloadMessage.trim().length === 0) {
    return fallbackMessage
  }
  return payloadMessage
}

export const dispatchExportFrame = (
  frame: string,
  handlers: CollectionExportHandlers
): boolean => {
  const exportFrame = parseExportFrame(frame)
  if (exportFrame === null) {
    return false
  }

  if (exportFrame.eventName === STAGE_EVENT_NAME) {
    const stageEvent = parseStageEvent(exportFrame.payload)
    if (stageEvent !== null) {
      handlers.onStage(stageEvent)
    }
    return false
  }

  if (exportFrame.eventName === RESULT_EVENT_NAME) {
    const exportResult = parseResultEvent(exportFrame.payload)
    if (exportResult === null) {
      throw new ApiError(EXPORT_UNREADABLE_RESULT_MESSAGE, 0)
    }
    handlers.onResult(exportResult)
    return true
  }

  if (exportFrame.eventName === ERROR_EVENT_NAME) {
    throw new ApiError(readPayloadMessage(exportFrame.payload, EXPORT_FAILED_MESSAGE), 0)
  }

  return false
}

export const drainExportFrames = (
  state: CollectionExportStreamState,
  handlers: CollectionExportHandlers
): void => {
  let separatorIndex = state.buffer.indexOf("\n\n")
  while (separatorIndex >= 0 && !state.isDone) {
    const frame = state.buffer.slice(0, separatorIndex)
    state.buffer = state.buffer.slice(separatorIndex + 2)
    state.isDone = dispatchExportFrame(frame, handlers)
    separatorIndex = state.buffer.indexOf("\n\n")
  }
}
