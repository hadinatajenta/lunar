import type { Page, Route } from "@playwright/test"
import { mockAssignedIssues, mockBacklogResponse } from "../data/jira.data"
import { mockPushes, mockPRs, mockDiff, mockAIReview } from "../data/bitbucket.data"
import {
  mockConfluenceDocuments,
  mockMigTargets,
  mockQueryReviewApplyResult,
  mockQueryReviewResult,
  mockSopApplyResult,
  mockSopGenerateResult,
  mockSopGenerateResultScanned,
  mockUtDocsApplyResult,
  mockUtDocsResult,
  mockUtRunDone,
  buildMockSopScanResult
} from "../data/confluence.data"

export function routeSecrets(
  page: Page,
  options: {
    hasJiraPat?: boolean
    hasBitbucketPat?: boolean
    hasConfluencePat?: boolean
    hasAiKeys?: boolean
    configuredAiProviders?: string[]
  } = {}
) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          user_id: "test-user-id",
          has_jira_pat: options.hasJiraPat ?? true,
          jira_username: "developer",
          has_bitbucket_pat: options.hasBitbucketPat ?? true,
          bitbucket_username: "developer",
          has_confluence_pat: options.hasConfluencePat ?? true,
          has_ai_keys: options.hasAiKeys ?? true,
          configured_ai_providers: options.configuredAiProviders ?? ["deepseek"],
          updated_at: "Just now"
        })
      })
      return
    }
    await route.continue()
  })
}

export function routeCurrentUser(page: Page) {
  return page.route(/\/api\/auth\/me$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: "test-user-id",
        email: "developer@lunar.dev",
        full_name: "Lunar Developer",
        role: "developer"
      })
    })
  })
}

export function routeSystemConfig(page: Page, systemAi: string[] = []) {
  return page.route(/\/api\/config/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        jira_base_url: "https://jira.bri.co.id",
        bitbucket_base_url: "https://bitbucket.bri.co.id",
        confluence_base_url: "https://confluence.bri.co.id",
        system_ai_providers: systemAi
      })
    })
  })
}

export function routeStandardJira(page: Page) {
  return page.route(/\/api\/jira\//, async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.includes("/issues/remotelinks")) {
      const key = url.searchParams.get("key") || ""
      await route.fulfill({
        status: 200,
        json: [
          {
            id: `link-${key}`,
            relationship: "wiki",
            object: {
              url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=UT-101",
              title: `${key} Confluence Specification`
            }
          }
        ]
      })
      return
    }
    if (url.pathname.includes("/issues/")) {
      const key = decodeURIComponent(url.pathname.split("/issues/")[1] || "")
      const found = mockAssignedIssues.find(
        (item) => item.key === key || item.id === key
      )
      if (found) {
        await route.fulfill({ status: 200, json: found })
        return
      }
      await route.fulfill({ status: 404, json: { error: "Issue not found" } })
      return
    }
    if (url.pathname.endsWith("/issues")) {
      await route.fulfill({ status: 200, json: mockAssignedIssues })
      return
    }
    if (url.pathname.endsWith("/backlog")) {
      await route.fulfill({ status: 200, json: mockBacklogResponse })
      return
    }
    await route.fulfill({ status: 404, json: { error: "Not found" } })
  })
}

export function routeStandardPushes(page: Page) {
  return page.route(/\/api\/bitbucket\/pushes/, async (route) => {
    const url = new URL(route.request().url())
    const filter = url.searchParams.get("filter")
    if (filter === "ready") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "ready") })
    } else if (filter === "stale") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "stale") })
    } else {
      await route.fulfill({ json: mockPushes })
    }
  })
}

export function routeStandardPRs(page: Page) {
  return page.route(/\/api\/bitbucket\/prs/, async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() === "POST" && url.pathname.endsWith("/prs")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "#196",
          number: 196,
          repo: payload.repo,
          title: payload.title,
          source_branch: payload.source_branch,
          target_branch: payload.target_branch,
          status: "open",
          updated_relative: "Just now",
          author: "Lunar Developer",
          files_count: 2,
          lines_added: 45,
          lines_deleted: 8,
          is_assigned_to_me: false,
          is_ai_flagged: false
        }
      })
      return
    }

    if (url.pathname.includes("/diff")) {
      await route.fulfill({ json: mockDiff })
      return
    }

    if (url.pathname.includes("/ai-review")) {
      await route.fulfill({ json: mockAIReview })
      return
    }

    if (url.pathname.includes("/action")) {
      await route.fulfill({ status: 200, json: { status: "success", action: "approve", pr_id: "188" } })
      return
    }

    if (url.pathname.includes("/comments")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "comm-101",
          pr_id: "191",
          repo: "BRI/service-map",
          author: "Lunar Developer",
          content: payload.content,
          created_at: "Just now"
        }
      })
      return
    }

    const filter = url.searchParams.get("filter")
    if (filter === "ai") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_ai_flagged) })
    } else if (filter === "mine") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_assigned_to_me) })
    } else {
      await route.fulfill({ json: mockPRs })
    }
  })
}

const CONFLUENCE_AI_PATHS = ["/query-review", "/ut-docs", "/mig-targets", "/ut-docs/apply", "/sop/generate", "/sop/apply"]

export function routeStandardConfluence(page: Page, documents = mockConfluenceDocuments) {
  return page.route(/\/api\/confluence\//, async (route) => {
    const url = new URL(route.request().url())
    if (CONFLUENCE_AI_PATHS.some((suffix) => url.pathname.endsWith(suffix))) {
      await route.fulfill({ status: 404, json: { error: "Not mocked" } })
      return
    }
    if (url.pathname.includes("/documents/")) {
      const id = decodeURIComponent(url.pathname.split("/documents/")[1] || "")
      const found = documents.find((d) => d.id === id)
      if (found) {
        await route.fulfill({ status: 200, json: found })
        return
      }
      await route.fulfill({ status: 404, json: { error: "Document not found" } })
      return
    }
    if (url.pathname.endsWith("/documents")) {
      await route.fulfill({ status: 200, json: documents })
      return
    }
    await route.fulfill({ status: 404, json: { error: "Not found" } })
  })
}

export interface ConfluenceAiResponse {
  status?: number
  body?: unknown
}

export interface ConfluenceAiRouteRecorder {
  requests: string[]
  registration: Promise<unknown>
}

const QUERY_REVIEW_PATTERN = /\/api\/confluence\/documents\/[^/]+\/query-review$/
const UT_DOCS_PATTERN = /\/api\/confluence\/documents\/[^/]+\/ut-docs$/

const createResponsePicker = (responses: ConfluenceAiResponse[]) => {
  let responseIndex = 0
  return () => {
    const response = responses[Math.min(responseIndex, responses.length - 1)]
    responseIndex += 1
    return response
  }
}

export function routeQueryReview(
  page: Page,
  options: { responses?: ConfluenceAiResponse[]; delayMs?: number } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockQueryReviewResult }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(QUERY_REVIEW_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    if (options.delayMs !== undefined) {
      await new Promise((resolve) => setTimeout(resolve, options.delayMs))
    }
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

const QUERY_REVIEW_APPLY_PATTERN = /\/api\/confluence\/documents\/[^/]+\/query-review\/apply$/

export function routeQueryReviewApply(
  page: Page,
  options: { responses?: ConfluenceAiResponse[] } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockQueryReviewApplyResult }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(QUERY_REVIEW_APPLY_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

export function captureClipboard(page: Page) {
  return page.addInitScript(() => {
    const captured: Record<string, string>[] = []
    ;(window as unknown as { __capturedClipboard: Record<string, string>[] }).__capturedClipboard =
      captured
    Object.defineProperty(navigator.clipboard, "write", {
      configurable: true,
      value: async (items: ClipboardItem[]) => {
        for (const item of items) {
          const entry: Record<string, string> = {}
          for (const type of item.types) {
            entry[type] = await (await item.getType(type)).text()
          }
          captured.push(entry)
        }
      }
    })
  })
}

export function captureClipboardText(page: Page) {
  return page.addInitScript(() => {
    const captured: string[] = []
    ;(window as unknown as { __capturedClipboardText: string[] }).__capturedClipboardText = captured
    Object.defineProperty(navigator.clipboard, "writeText", {
      configurable: true,
      value: async (text: string) => {
        captured.push(text)
      }
    })
  })
}

export function routeUtDocs(
  page: Page,
  options: { responses?: ConfluenceAiResponse[] } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockUtDocsResult }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(UT_DOCS_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

const MIG_TARGETS_PATTERN = /\/api\/confluence\/documents\/[^/]+\/mig-targets$/

export function routeMigTargets(
  page: Page,
  options: { responses?: ConfluenceAiResponse[]; delayMs?: number } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockMigTargets }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(MIG_TARGETS_PATTERN, async (route) => {
    requests.push(route.request().url())
    if (options.delayMs !== undefined) {
      await new Promise((resolve) => setTimeout(resolve, options.delayMs))
    }
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

const UT_DOCS_APPLY_PATTERN = /\/api\/confluence\/documents\/[^/]+\/ut-docs\/apply$/

export function routeUtDocsApply(
  page: Page,
  options: { responses?: ConfluenceAiResponse[] } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockUtDocsApplyResult }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(UT_DOCS_APPLY_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

const SOP_GENERATE_PATTERN = /\/api\/confluence\/documents\/[^/]+\/sop\/generate$/
const SOP_APPLY_PATTERN = /\/api\/confluence\/documents\/[^/]+\/sop\/apply$/

export function routeSopGenerate(
  page: Page,
  options: { responses?: ConfluenceAiResponse[]; delayMs?: number } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [
    { status: 200, body: mockSopGenerateResult },
    { status: 200, body: mockSopGenerateResultScanned }
  ]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(SOP_GENERATE_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    if (options.delayMs !== undefined) {
      await new Promise((resolve) => setTimeout(resolve, options.delayMs))
    }
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

export function routeSopApply(
  page: Page,
  options: { responses?: ConfluenceAiResponse[] } = {}
): ConfluenceAiRouteRecorder {
  const requests: string[] = []
  const responses = options.responses ?? [{ status: 200, body: mockSopApplyResult }]
  const pickResponse = createResponsePicker(responses)

  const registration = page.route(SOP_APPLY_PATTERN, async (route) => {
    requests.push(route.request().postData() ?? "")
    const response = pickResponse()
    await route.fulfill({ status: response.status ?? 200, json: response.body ?? {} })
  })

  return { requests, registration }
}

export const HELPER_BASE_URL = "http://127.0.0.1:5199"
export const HELPER_ROOT_PATH = "/Users/erendt/BRI"
export const HELPER_REPOSITORY_NAMES = ["micro-payment", "mms-frontend"]

const HELPER_CORS_HEADERS: Record<string, string> = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, content-type, accept",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS"
}

const SCREENSHOT_PNG_BASE64 =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
const SCREENSHOT_PNG = Buffer.from(SCREENSHOT_PNG_BASE64, "base64")

export function routeHelperWorkspace(
  page: Page,
  options: { rootPath?: string; helperUrl?: string } = {}
): Promise<unknown> {
  return page.route(/\/api\/workspace$/, (route) =>
    route.fulfill({
      json: {
        source: "helper",
        root_path: options.rootPath ?? HELPER_ROOT_PATH,
        helper_url: options.helperUrl ?? HELPER_BASE_URL,
        has_helper_token: false,
        selected: HELPER_REPOSITORY_NAMES
      }
    })
  )
}

export interface UtRunHelperOptions {
  streamBody?: string
  streamDelayMs?: number
  tokenStatus?: number
  repositories?: string[]
  buildStream?: (requestBody: { repo?: string }) => string
}

export interface UtRunHelperRecorder {
  runRequests: string[]
  registrations: Promise<unknown>[]
}

export const buildUtRunStream = (done: unknown = mockUtRunDone): string =>
  [
    'event: phase\ndata: {"phase":"scan"}\n\n',
    'event: phase\ndata: {"phase":"safety"}\n\n',
    'event: phase\ndata: {"phase":"run"}\n\n',
    'event: output\ndata: {"text":"=== RUN   TestDao\\n"}\n\n',
    'event: output\ndata: {"text":"--- PASS: TestDao (0.02s)\\nPASS\\n"}\n\n',
    'event: phase\ndata: {"phase":"screenshot"}\n\n',
    `event: done\ndata: ${JSON.stringify(done)}\n\n`
  ].join("")

export function routeUtRunHelper(page: Page, options: UtRunHelperOptions = {}): UtRunHelperRecorder {
  const runRequests: string[] = []
  const registrations: Promise<unknown>[] = []

  const fulfillPreflight = async (route: Route): Promise<boolean> => {
    if (route.request().method() !== "OPTIONS") {
      return false
    }
    await route.fulfill({ status: 204, headers: HELPER_CORS_HEADERS })
    return true
  }

  registrations.push(
    page.route("**/token", async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      const tokenStatus = options.tokenStatus ?? 200
      if (tokenStatus !== 200) {
        await route.fulfill({
          status: tokenStatus,
          headers: HELPER_CORS_HEADERS,
          json: { message: "the helper token is unavailable" }
        })
        return
      }
      await route.fulfill({ status: 200, headers: HELPER_CORS_HEADERS, json: { token: "e2e-helper-token" } })
    })
  )

  registrations.push(
    page.route("**/repositories**", async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      const rootPath = new URL(route.request().url()).searchParams.get("root") ?? HELPER_ROOT_PATH
      const repositoryNames = options.repositories ?? HELPER_REPOSITORY_NAMES
      await route.fulfill({
        status: 200,
        headers: HELPER_CORS_HEADERS,
        json: {
          root_path: rootPath,
          repositories: repositoryNames.map((name) => ({
            name,
            is_git: true,
            branch: "development",
            dirty_count: 0,
            updated_relative: "2 hours ago",
            error: ""
          }))
        }
      })
    })
  )

  registrations.push(
    page.route("**/ut-runs", async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      const requestBody = JSON.parse(route.request().postData() ?? "{}") as { repo?: string }
      runRequests.push(route.request().postData() ?? "")
      if (options.streamDelayMs !== undefined) {
        await new Promise((resolve) => setTimeout(resolve, options.streamDelayMs))
      }
      const streamBody = options.buildStream
        ? options.buildStream(requestBody)
        : options.streamBody ?? buildUtRunStream()
      await route.fulfill({
        status: 200,
        headers: HELPER_CORS_HEADERS,
        contentType: "text/event-stream",
        body: streamBody
      })
    })
  )

  registrations.push(
    page.route(/\/ut-runs\/[^/]+\/screenshot\.png$/, async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      await route.fulfill({
        status: 200,
        headers: { ...HELPER_CORS_HEADERS, "Content-Type": "image/png" },
        body: SCREENSHOT_PNG
      })
    })
  )

  return { runRequests, registrations }
}

export interface SopHelperOptions {
  scanResponses?: Record<string, unknown>
  scanStatus?: number
}

export interface SopHelperRecorder {
  scanRequests: string[]
  registrations: Promise<unknown>[]
}

export function routeSopHelper(page: Page, options: SopHelperOptions = {}): SopHelperRecorder {
  const scanRequests: string[] = []
  const registrations: Promise<unknown>[] = []

  const fulfillPreflight = async (route: Route): Promise<boolean> => {
    if (route.request().method() !== "OPTIONS") {
      return false
    }
    await route.fulfill({ status: 204, headers: HELPER_CORS_HEADERS })
    return true
  }

  registrations.push(
    page.route("**/token", async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      await route.fulfill({ status: 200, headers: HELPER_CORS_HEADERS, json: { token: "e2e-helper-token" } })
    })
  )

  registrations.push(
    page.route("**/sop/scan", async (route) => {
      if (await fulfillPreflight(route)) {
        return
      }
      const requestBody = JSON.parse(route.request().postData() ?? "{}") as { repo?: string }
      scanRequests.push(route.request().postData() ?? "")
      if (options.scanStatus !== undefined && options.scanStatus !== 200) {
        await route.fulfill({
          status: options.scanStatus,
          headers: HELPER_CORS_HEADERS,
          json: { message: "the helper could not scan the repository" }
        })
        return
      }
      const repository = requestBody.repo ?? ""
      const response = options.scanResponses?.[repository] ?? buildMockSopScanResult(repository)
      await route.fulfill({ status: 200, headers: HELPER_CORS_HEADERS, json: response })
    })
  )

  return { scanRequests, registrations }
}

export type WorkspaceGitAction = "checkout" | "fetch" | "pull" | "stash" | "discard"

export interface WorkspaceGitActionCall {
  action: WorkspaceGitAction
  repositoryName: string
  branch?: string
}

export interface WorkspaceGitActionRecorder {
  calls: WorkspaceGitActionCall[]
  registration: Promise<unknown>
  callsFor: (action: WorkspaceGitAction) => WorkspaceGitActionCall[]
  countFor: (action: WorkspaceGitAction) => number
}

const WORKSPACE_GIT_ACTION_PATTERN = /\/api\/workspace\/(checkout|fetch|pull|stash|discard)$/

const buildGitActionResponse = (action: WorkspaceGitAction, branch: string) => {
  if (action === "checkout") return { branch }
  if (action === "fetch") return { ahead: 2, behind: 1 }
  if (action === "pull") return { branch: "development", updated_relative: "Just now" }
  if (action === "stash") return { stash_count: 1 }
  return { restored_count: 3 }
}

export function routeWorkspaceGitActions(page: Page): WorkspaceGitActionRecorder {
  const calls: WorkspaceGitActionCall[] = []

  const routeHandler = async (route: Route) => {
    const request = route.request()
    const action = new URL(request.url()).pathname.split("/").pop() as WorkspaceGitAction
    const requestBody = JSON.parse(request.postData() ?? "{}") as { name?: string; branch?: string }
    const recordedCall: WorkspaceGitActionCall = {
      action,
      repositoryName: requestBody.name ?? ""
    }
    if (requestBody.branch !== undefined) {
      recordedCall.branch = requestBody.branch
    }
    calls.push(recordedCall)
    await route.fulfill({ status: 200, json: buildGitActionResponse(action, requestBody.branch ?? "") })
  }

  const registration = page.route(WORKSPACE_GIT_ACTION_PATTERN, routeHandler)

  return {
    calls,
    registration,
    callsFor: (action) => calls.filter((call) => call.action === action),
    countFor: (action) => calls.filter((call) => call.action === action).length
  }
}
