import { ApiError } from "../../../lib/http"
import { DEFAULT_HELPER_URL, HelperConnectionError } from "../../microservices/api/helper-api"
import {
  HELPER_REQUEST_TIMEOUT_MS,
  RequestTimeoutError,
  withRequestTimeout
} from "../../microservices/api/request-timeout"
import type { HelperConnection } from "../../microservices/types"
import type { CodeIndexStatus, CodeSearchResponse } from "../types"

export const CODE_INDEX_SEARCH_DEFAULT_LIMIT = 8
export const CODE_INDEX_SEARCH_MAX_LIMIT = 25

const HELPER_UNREACHABLE_MESSAGE =
  "The Lunar helper is not reachable. Start the helper on this machine and try again."
const HELPER_TIMEOUT_MESSAGE =
  "The Lunar helper did not respond in time. Check that it is running and try again."
const HELPER_TOKEN_REJECTED_MESSAGE =
  "The helper rejected the token. Copy the current token from ~/.lunar/helper-token into Settings."
const HELPER_INVALID_URL_MESSAGE =
  "The helper URL is not a valid address. Use the full address, for example http://127.0.0.1:5199"
const HELPER_UNSUPPORTED_SCHEME_MESSAGE = "The helper URL must start with http:// or https://"

export interface CodeIndexBuildRequest {
  root: string
  repos?: string[]
  rebuild?: boolean
}

export interface CodeIndexBuildResponse {
  started: boolean
}

export interface CodeSearchRequest {
  query: string
  repos?: string[]
  limit?: number
}

export interface HelperRequestOptions {
  method: "GET" | "POST"
  body?: unknown
  params?: Record<string, string>
}

export function normalizeHelperUrl(helperUrl: string): string {
  const trimmedUrl = helperUrl.trim()
  return trimmedUrl.length > 0 ? trimmedUrl : DEFAULT_HELPER_URL
}

export function buildHelperEndpoint(connection: HelperConnection, path: string): string {
  let parsedUrl: URL
  try {
    parsedUrl = new URL(normalizeHelperUrl(connection.helperUrl))
  } catch {
    throw new ApiError(HELPER_INVALID_URL_MESSAGE, 0)
  }
  if (parsedUrl.protocol !== "http:" && parsedUrl.protocol !== "https:") {
    throw new ApiError(HELPER_UNSUPPORTED_SCHEME_MESSAGE, 0)
  }
  const basePath = parsedUrl.pathname.replace(/\/+$/, "")
  return `${parsedUrl.origin}${basePath}${path}`
}

export function appendQueryParams(endpoint: string, params: Record<string, string>): string {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value.length > 0) {
      searchParams.append(key, value)
    }
  }
  const queryString = searchParams.toString()
  return queryString.length > 0 ? `${endpoint}?${queryString}` : endpoint
}

export function createHelperHeaders(connection: HelperConnection, accept: string): Headers {
  const headers = new Headers()
  headers.set("Accept", accept)
  if (connection.helperToken.length > 0) {
    headers.set("Authorization", `Bearer ${connection.helperToken}`)
  }
  return headers
}

function extractPayloadMessage(payload: unknown): string | null {
  if (typeof payload !== "object" || payload === null) {
    return null
  }
  if ("message" in payload && typeof payload.message === "string") {
    return payload.message
  }
  if ("error" in payload && typeof payload.error === "string") {
    return payload.error
  }
  return null
}

export function resolveHelperErrorMessage(
  statusCode: number,
  payload: unknown,
  statusText: string
): string {
  if (statusCode === 401) {
    return HELPER_TOKEN_REJECTED_MESSAGE
  }
  const payloadMessage = extractPayloadMessage(payload)
  if (payloadMessage !== null && payloadMessage.length > 0) {
    return payloadMessage
  }
  if (statusText.length > 0) {
    return statusText
  }
  return `The helper request failed with status ${statusCode}`
}

export function normalizeSearchLimit(limit?: number): number {
  if (limit === undefined || !Number.isFinite(limit)) {
    return CODE_INDEX_SEARCH_DEFAULT_LIMIT
  }
  return Math.min(Math.max(Math.trunc(limit), 1), CODE_INDEX_SEARCH_MAX_LIMIT)
}

export async function requestHelperJson<T>(
  connection: HelperConnection,
  path: string,
  options: HelperRequestOptions
): Promise<T> {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)
  const endpoint = appendQueryParams(buildHelperEndpoint(connection, path), options.params ?? {})
  const headers = createHelperHeaders(connection, "application/json")
  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json")
  }

  let response: Response
  try {
    response = await withRequestTimeout(HELPER_REQUEST_TIMEOUT_MS, (signal) =>
      fetch(endpoint, {
        method: options.method,
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
        signal
      })
    )
  } catch (thrownError) {
    if (thrownError instanceof RequestTimeoutError) {
      throw new HelperConnectionError(HELPER_TIMEOUT_MESSAGE, helperUrl)
    }
    throw new HelperConnectionError(HELPER_UNREACHABLE_MESSAGE, helperUrl)
  }

  const contentType = response.headers.get("Content-Type") ?? ""
  const isJsonResponse = contentType.includes("application/json")
  const parsedPayload: unknown = isJsonResponse
    ? await response.json().catch(() => null)
    : await response.text().catch(() => null)

  if (!response.ok) {
    throw new ApiError(
      resolveHelperErrorMessage(response.status, parsedPayload, response.statusText),
      response.status,
      parsedPayload
    )
  }

  return parsedPayload as T
}

export const fetchCodeIndexStatus = (connection: HelperConnection): Promise<CodeIndexStatus> =>
  requestHelperJson<CodeIndexStatus>(connection, "/index/status", { method: "GET" })

export const startCodeIndexBuild = (
  connection: HelperConnection,
  request: CodeIndexBuildRequest
): Promise<CodeIndexBuildResponse> =>
  requestHelperJson<CodeIndexBuildResponse>(connection, "/index/build", {
    method: "POST",
    body: {
      root: request.root,
      repos: request.repos ?? [],
      rebuild: request.rebuild ?? false
    }
  })

export const searchCodeIndex = (
  connection: HelperConnection,
  request: CodeSearchRequest
): Promise<CodeSearchResponse> =>
  requestHelperJson<CodeSearchResponse>(connection, "/search", {
    method: "GET",
    params: {
      q: request.query,
      repos: (request.repos ?? []).join(","),
      limit: String(normalizeSearchLimit(request.limit))
    }
  })
