import type {
  UtRunDone,
  UtRunPhase,
  UtRunResult,
  UtRunStatus,
  UtTestFile,
  UtTestResult,
  UtTestStatus,
  UtUntestedFunction
} from "../types"

export const UT_RUN_PHASES: UtRunPhase[] = ["scan", "safety", "run", "screenshot"]

const UT_RUN_STATUSES: UtRunStatus[] = ["passed", "failed", "blocked", "not_run"]

const UT_TEST_STATUSES: UtTestStatus[] = ["passed", "failed", "skipped"]

export const UT_RUN_UNREADABLE_RESULT_MESSAGE =
  "The helper finished the run but its result could not be read."
export const UT_RUN_FAILED_MESSAGE = "The helper reported that the run failed."

export class UtRunStreamError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "UtRunStreamError"
  }
}

export interface UtRunPhaseEvent {
  kind: "phase"
  phase: UtRunPhase
}

export interface UtRunOutputEvent {
  kind: "output"
  text: string
}

export interface UtRunDoneEvent {
  kind: "done"
  result: UtRunDone
}

export type UtRunEvent = UtRunPhaseEvent | UtRunOutputEvent | UtRunDoneEvent

export interface UtRunStreamHandlers {
  onPhase: (phase: UtRunPhase) => void
  onOutput: (text: string) => void
  onDone: (result: UtRunDone) => void
}

export interface UtRunStreamState {
  buffer: string
  isDone: boolean
}

export const createUtRunStreamState = (): UtRunStreamState => ({ buffer: "", isDone: false })

interface UtRunFrame {
  eventName: string
  payload: string
}

const isUtRunPhase = (candidate: unknown): candidate is UtRunPhase =>
  typeof candidate === "string" && UT_RUN_PHASES.some((phase) => phase === candidate)

const isUtRunStatus = (candidate: unknown): candidate is UtRunStatus =>
  typeof candidate === "string" && UT_RUN_STATUSES.some((status) => status === candidate)

const parseFrame = (frame: string): UtRunFrame | null => {
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

const readStringField = (source: Record<string, unknown>, key: string): string =>
  typeof source[key] === "string" ? (source[key] as string) : ""

const readNumberField = (source: Record<string, unknown>, key: string): number => {
  const value = source[key]
  return typeof value === "number" && Number.isFinite(value) ? value : 0
}

const readBooleanField = (source: Record<string, unknown>, key: string): boolean =>
  source[key] === true

const readStringList = (value: unknown): string[] =>
  Array.isArray(value) ? value.filter((entry): entry is string => typeof entry === "string") : []

const readTestFileList = (
  value: unknown,
  fallbackClassification: UtTestFile["classification"]
): UtTestFile[] => {
  if (!Array.isArray(value)) {
    return []
  }
  return value.flatMap((entry) => {
    if (typeof entry !== "object" || entry === null) {
      return []
    }
    const record = entry as Record<string, unknown>
    const path = readStringField(record, "path")
    if (path.length === 0) {
      return []
    }
    const classification = record.classification === "blocked" ? "blocked" : fallbackClassification
    return [{
      path,
      package: readStringField(record, "package"),
      language: readStringField(record, "language"),
      classification,
      reason: readStringField(record, "reason")
    }]
  })
}

const readRunResult = (value: unknown): UtRunResult | null => {
  if (typeof value !== "object" || value === null) {
    return null
  }
  const record = value as Record<string, unknown>
  if (!isUtRunStatus(record.status)) {
    return null
  }
  return {
    status: record.status,
    executed_packages: readStringList(record.executed_packages),
    total: readNumberField(record, "total"),
    passed: readNumberField(record, "passed"),
    failed: readNumberField(record, "failed"),
    duration_ms: readNumberField(record, "duration_ms"),
    output_excerpt: readStringField(record, "output_excerpt")
  }
}

const readTestResultList = (value: unknown): UtTestResult[] => {
  if (!Array.isArray(value)) {
    return []
  }
  return value.flatMap((entry) => {
    if (typeof entry !== "object" || entry === null) {
      return []
    }
    const record = entry as Record<string, unknown>
    const name = readStringField(record, "name")
    if (name.length === 0) {
      return []
    }
    const status: UtTestStatus = UT_TEST_STATUSES.find((candidate) => candidate === record.status) ?? "skipped"
    return [{
      name,
      package: readStringField(record, "package"),
      file: readStringField(record, "file"),
      status,
      duration_ms: readNumberField(record, "duration_ms"),
      output: readStringField(record, "output")
    }]
  })
}

const readUntestedFunctionList = (value: unknown): UtUntestedFunction[] => {
  if (!Array.isArray(value)) {
    return []
  }
  return value.flatMap((entry) => {
    if (typeof entry !== "object" || entry === null) {
      return []
    }
    const record = entry as Record<string, unknown>
    const name = readStringField(record, "name")
    if (name.length === 0) {
      return []
    }
    return [{
      name,
      file: readStringField(record, "file"),
      package: readStringField(record, "package")
    }]
  })
}

export const parseUtRunDone = (payload: string): UtRunDone => {
  const record = parseJsonObject(payload)
  if (record === null) {
    throw new UtRunStreamError(UT_RUN_UNREADABLE_RESULT_MESSAGE)
  }
  const run = readRunResult(record.run)
  const runId = readStringField(record, "run_id")
  if (run === null || runId.length === 0) {
    throw new UtRunStreamError(UT_RUN_UNREADABLE_RESULT_MESSAGE)
  }
  return {
    run_id: runId,
    repo: readStringField(record, "repo"),
    branch: readStringField(record, "branch"),
    base_ref: readStringField(record, "base_ref"),
    changed_files: readStringList(record.changed_files),
    test_files: readTestFileList(record.test_files, "safe"),
    blocked_files: readTestFileList(record.blocked_files, "blocked"),
    run,
    stashed: readBooleanField(record, "stashed"),
    restored: readBooleanField(record, "restored"),
    tests: readTestResultList(record.tests),
    untested_functions: readUntestedFunctionList(record.untested_functions),
    screenshot_url: readStringField(record, "screenshot_url")
  }
}

const resolveEventName = (frame: UtRunFrame, payloadRecord: Record<string, unknown> | null): string => {
  if (frame.eventName.length > 0) {
    return frame.eventName
  }
  if (payloadRecord !== null) {
    const typeField = payloadRecord.type
    if (typeof typeField === "string") {
      return typeField
    }
  }
  return ""
}

const readErrorEventMessage = (payload: string, payloadRecord: Record<string, unknown> | null): string => {
  if (payloadRecord !== null) {
    const message = payloadRecord.message
    if (typeof message === "string" && message.trim().length > 0) {
      return message
    }
    const error = payloadRecord.error
    if (typeof error === "string" && error.trim().length > 0) {
      return error
    }
  }
  return payload.trim().length > 0 ? payload.trim() : UT_RUN_FAILED_MESSAGE
}

const readOutputEventText = (payload: string, payloadRecord: Record<string, unknown> | null): string => {
  if (payloadRecord !== null) {
    const text = payloadRecord.text
    if (typeof text === "string") {
      return text
    }
    const output = payloadRecord.output
    if (typeof output === "string") {
      return output
    }
    const chunk = payloadRecord.chunk
    if (typeof chunk === "string") {
      return chunk
    }
  }
  return payload
}

export const dispatchUtRunFrame = (frame: string, handlers: UtRunStreamHandlers): boolean => {
  const parsedFrame = parseFrame(frame)
  if (parsedFrame === null) {
    return false
  }
  if (parsedFrame.payload === "[DONE]") {
    return true
  }

  const payloadRecord = parseJsonObject(parsedFrame.payload)
  const eventName = resolveEventName(parsedFrame, payloadRecord)

  if (eventName === "phase") {
    const phaseCandidate = payloadRecord === null ? parsedFrame.payload : payloadRecord.phase
    if (isUtRunPhase(phaseCandidate)) {
      handlers.onPhase(phaseCandidate)
    }
    return false
  }

  if (eventName === "output") {
    handlers.onOutput(readOutputEventText(parsedFrame.payload, payloadRecord))
    return false
  }

  if (eventName === "done") {
    handlers.onDone(parseUtRunDone(parsedFrame.payload))
    return true
  }

  if (eventName === "error") {
    throw new UtRunStreamError(readErrorEventMessage(parsedFrame.payload, payloadRecord))
  }

  return false
}

export const drainUtRunFrames = (state: UtRunStreamState, handlers: UtRunStreamHandlers): void => {
  let separatorIndex = state.buffer.indexOf("\n\n")
  while (separatorIndex >= 0 && !state.isDone) {
    const frame = state.buffer.slice(0, separatorIndex)
    state.buffer = state.buffer.slice(separatorIndex + 2)
    state.isDone = dispatchUtRunFrame(frame, handlers)
    separatorIndex = state.buffer.indexOf("\n\n")
  }
}
