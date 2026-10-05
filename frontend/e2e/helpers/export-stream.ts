import type { Page, Route } from "@playwright/test"

export interface ExportStageFrame {
  stage: string
  detail: string
}

export interface ExportResultFrame {
  service: string
  request_count: number
  folder_count: number
  collection: unknown
}

export interface ExportRecordedRequest {
  root: string
  name: string
  model: string
  backendUrl: string
  sessionToken: string
  fieldNames: string[]
}

export interface ExportRecorder {
  requests: ExportRecordedRequest[]
  registration: Promise<unknown>
  countFor: (repositoryName: string) => number
}

export interface ExportRouteOptions {
  streamBody?: string
  streamBodies?: string[]
  status?: number
  responseBody?: Record<string, unknown>
}

export interface ExportStreamChunk {
  delayMs: number
  body: string
}

export interface StreamingExportScript {
  chunks: ExportStreamChunk[]
  status?: number
  responseBody?: Record<string, unknown>
}

export interface StreamingExportRecorder {
  registration: Promise<unknown>
  readRequests: () => Promise<ExportRecordedRequest[]>
}

export const EXPORT_ROUTE_PATTERN = "**/export"

export const CANNED_COLLECTION = {
  info: {
    name: "atlas",
    schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  item: [],
  variable: []
}

export const DEFAULT_COLLECTION_NAME = "atlas"
export const DEFAULT_REQUEST_COUNT = 29
export const DEFAULT_FOLDER_COUNT = 5

export const DEFAULT_EXPORT_STAGES: ExportStageFrame[] = [
  { stage: "selecting", detail: "selected 37 files from atlas" },
  { stage: "prompt", detail: "built a prompt with the gin appendix" },
  { stage: "model", detail: "awaiting the provider" },
  { stage: "validating", detail: "validating" }
]

export const buildStageFrame = (frame: ExportStageFrame): string =>
  `event: stage\ndata: ${JSON.stringify({ stage: frame.stage, detail: frame.detail })}\n\n`

export const buildResultFrame = (frame: ExportResultFrame): string =>
  `event: result\ndata: ${JSON.stringify(frame)}\n\n`

export const buildErrorFrame = (message: string): string =>
  `event: error\ndata: ${JSON.stringify({ error: message })}\n\n`

export const buildExportStream = (...frames: string[]): string => frames.join("")

export const buildDefaultResultFrame = (): string =>
  buildResultFrame({
    service: DEFAULT_COLLECTION_NAME,
    request_count: DEFAULT_REQUEST_COUNT,
    folder_count: DEFAULT_FOLDER_COUNT,
    collection: CANNED_COLLECTION
  })

export const buildDefaultExportStream = (): string =>
  buildExportStream(...DEFAULT_EXPORT_STAGES.map(buildStageFrame), buildDefaultResultFrame())

const readTextField = (payload: Record<string, unknown>, fieldName: string): string =>
  typeof payload[fieldName] === "string" ? (payload[fieldName] as string) : ""

const recordRequest = (payload: Record<string, unknown>): ExportRecordedRequest => ({
  root: readTextField(payload, "root"),
  name: readTextField(payload, "name"),
  model: readTextField(payload, "model"),
  backendUrl: readTextField(payload, "backend_url"),
  sessionToken: readTextField(payload, "session_token"),
  fieldNames: Object.keys(payload).sort()
})

export function routeCollectionExport(page: Page, options: ExportRouteOptions = {}): ExportRecorder {
  const requests: ExportRecordedRequest[] = []
  const streamBodies = options.streamBodies ?? []
  let responseIndex = 0

  const routeHandler = async (route: Route) => {
    const payload = JSON.parse(route.request().postData() ?? "{}") as Record<string, unknown>
    requests.push(recordRequest(payload))

    if (options.status !== undefined) {
      await route.fulfill({
        status: options.status,
        contentType: "application/json",
        body: JSON.stringify(options.responseBody ?? {})
      })
      return
    }

    const streamBody = streamBodies.length === 0
      ? options.streamBody ?? buildDefaultExportStream()
      : streamBodies[Math.min(responseIndex, streamBodies.length - 1)]
    responseIndex += 1

    await route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      body: streamBody
    })
  }

  const registration = page.route(EXPORT_ROUTE_PATTERN, routeHandler)

  return {
    requests,
    registration,
    countFor: (repositoryName) => requests.filter((request) => request.name === repositoryName).length
  }
}

export function installStreamingExport(
  page: Page,
  script: StreamingExportScript
): StreamingExportRecorder {
  const registration = page.addInitScript((streamScript) => {
    const recordedRequests: ExportRecordedRequest[] = []
    const sink = window as unknown as { __lunarExportRequests: ExportRecordedRequest[] }
    sink.__lunarExportRequests = recordedRequests
    const originalFetch = window.fetch.bind(window)

    const record = (payload: Record<string, unknown>): void => {
      const textField = (fieldName: string): string =>
        typeof payload[fieldName] === "string" ? (payload[fieldName] as string) : ""
      recordedRequests.push({
        root: textField("root"),
        name: textField("name"),
        model: textField("model"),
        backendUrl: textField("backend_url"),
        sessionToken: textField("session_token"),
        fieldNames: Object.keys(payload).sort()
      })
    }

    window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const targetUrl =
        typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url
      if (!targetUrl.endsWith("/export")) {
        return originalFetch(input, init)
      }

      const requestBody = typeof init?.body === "string" ? init.body : "{}"
      record(JSON.parse(requestBody) as Record<string, unknown>)

      if (streamScript.status !== undefined) {
        return new Response(JSON.stringify(streamScript.responseBody ?? {}), {
          status: streamScript.status,
          headers: { "Content-Type": "application/json" }
        })
      }

      const encoder = new TextEncoder()
      const bodyStream = new ReadableStream<Uint8Array>({
        async start(controller) {
          for (const chunk of streamScript.chunks) {
            await new Promise((resolve) => setTimeout(resolve, chunk.delayMs))
            controller.enqueue(encoder.encode(chunk.body))
          }
          controller.close()
        }
      })

      return new Response(bodyStream, {
        status: 200,
        headers: { "Content-Type": "text/event-stream" }
      })
    }
  }, script)

  return {
    registration,
    readRequests: () =>
      page.evaluate<ExportRecordedRequest[]>(() => {
        const sink = window as unknown as { __lunarExportRequests?: ExportRecordedRequest[] }
        return sink.__lunarExportRequests ?? []
      })
  }
}
