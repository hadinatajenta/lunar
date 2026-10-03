import { ApiError } from "../../../lib/http"
import type {
  HelperConnection,
  HelperHealth,
  RepositoryListResponse,
  ServiceListResponse,
  ValidatePathRequest,
  ValidatePathResponse
} from "../types"
import { HELPER_REQUEST_TIMEOUT_MS, RequestTimeoutError, withRequestTimeout } from "./request-timeout"

export const DEFAULT_HELPER_URL = "http://127.0.0.1:5199"

export const HELPER_EMPTY_TOKEN_MESSAGE =
  "The helper is running but did not return a token. Start it again and retry."

interface HelperTokenResponse {
  token: string
}

const HELPER_UNREACHABLE_MESSAGE =
  "The Lunar helper is not reachable. Start the helper on this machine and try again."
const HELPER_TIMEOUT_MESSAGE =
  "The Lunar helper did not respond in time. Check that it is running and try again."
const HELPER_TOKEN_REJECTED_MESSAGE =
  "The helper rejected the token. Copy the current token from ~/.lunar/helper-token into Settings."
const HELPER_INVALID_URL_MESSAGE =
  "The helper URL is not a valid address. Use the full address, for example http://127.0.0.1:5199"
const HELPER_UNSUPPORTED_SCHEME_MESSAGE = "The helper URL must start with http:// or https://"

export class HelperConnectionError extends Error {
  readonly helperUrl: string

  constructor(message: string, helperUrl: string) {
    super(message)
    this.name = "HelperConnectionError"
    this.helperUrl = helperUrl
  }
}

interface HelperRequestOptions {
  method: "GET" | "POST"
  body?: unknown
  params?: Record<string, string>
}

function normalizeHelperUrl(helperUrl: string): string {
  const trimmedUrl = helperUrl.trim()
  return trimmedUrl.length > 0 ? trimmedUrl : DEFAULT_HELPER_URL
}

function parseHelperUrl(helperUrl: string): URL | null {
  try {
    return new URL(normalizeHelperUrl(helperUrl))
  } catch {
    return null
  }
}

function buildHelperEndpoint(connection: HelperConnection, path: string): string {
  const parsedUrl = parseHelperUrl(connection.helperUrl)
  if (parsedUrl === null) {
    throw new ApiError(HELPER_INVALID_URL_MESSAGE, 0)
  }
  if (parsedUrl.protocol !== "http:" && parsedUrl.protocol !== "https:") {
    throw new ApiError(HELPER_UNSUPPORTED_SCHEME_MESSAGE, 0)
  }
  const basePath = parsedUrl.pathname.replace(/\/+$/, "")
  return `${parsedUrl.origin}${basePath}${path}`
}

function appendQueryParams(endpoint: string, params: Record<string, string>): string {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value.length > 0) {
      searchParams.append(key, value)
    }
  }
  const queryString = searchParams.toString()
  return queryString.length > 0 ? `${endpoint}?${queryString}` : endpoint
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

function resolveHelperErrorMessage(
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

async function requestHelper<T>(
  connection: HelperConnection,
  path: string,
  options: HelperRequestOptions
): Promise<T> {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)
  const endpoint = appendQueryParams(buildHelperEndpoint(connection, path), options.params ?? {})

  const headers = new Headers()
  headers.set("Accept", "application/json")
  if (connection.helperToken.length > 0) {
    headers.set("Authorization", `Bearer ${connection.helperToken}`)
  }
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

export const fetchHelperHealth = (connection: HelperConnection): Promise<HelperHealth> =>
  requestHelper<HelperHealth>({ helperUrl: connection.helperUrl, helperToken: "" }, "/health", { method: "GET" })

export const fetchHelperToken = async (helperUrl: string): Promise<string> => {
  const response = await requestHelper<HelperTokenResponse>(
    { helperUrl, helperToken: "" },
    "/token",
    { method: "GET" }
  )
  const token = typeof response.token === "string" ? response.token.trim() : ""
  if (token.length === 0) {
    throw new HelperConnectionError(HELPER_EMPTY_TOKEN_MESSAGE, normalizeHelperUrl(helperUrl))
  }
  return token
}

export const fetchHelperRepositories = (
  connection: HelperConnection,
  rootPath: string
): Promise<RepositoryListResponse> =>
  requestHelper<RepositoryListResponse>(connection, "/repositories", {
    method: "GET",
    params: { root: rootPath }
  })

export const fetchHelperServices = (
  connection: HelperConnection,
  rootPath: string,
  names: string[]
): Promise<ServiceListResponse> =>
  requestHelper<ServiceListResponse>(connection, "/services", {
    method: "GET",
    params: { root: rootPath, names: names.join(",") }
  })

export const validateHelperPath = (
  connection: HelperConnection,
  rootPath: string
): Promise<ValidatePathResponse> => {
  const requestBody: ValidatePathRequest = { root_path: rootPath }
  return requestHelper<ValidatePathResponse>(connection, "/validate-path", {
    method: "POST",
    body: requestBody
  })
}
