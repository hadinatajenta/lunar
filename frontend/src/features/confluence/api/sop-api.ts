import { ApiError } from "../../../lib/http"
import {
  buildHelperEndpoint,
  createHelperHeaders,
  normalizeHelperUrl,
  resolveHelperErrorMessage
} from "../../code-context/api/code-index-api"
import { HELPER_UNREACHABLE_MESSAGE, HelperConnectionError } from "../../microservices/api/helper-api"
import type { HelperConnection } from "../../microservices/types"
import type { SopScanRequest, SopScanResult } from "../types"

const SOP_SCAN_PATH = "/sop/scan"

const readErrorPayload = async (response: Response): Promise<unknown> => {
  const contentType = response.headers.get("Content-Type") ?? ""
  if (contentType.includes("application/json")) {
    return response.json().catch(() => null)
  }
  return response.text().catch(() => null)
}

export const requestSopScan = async (
  connection: HelperConnection,
  request: SopScanRequest,
  signal?: AbortSignal
): Promise<SopScanResult> => {
  const helperUrl = normalizeHelperUrl(connection.helperUrl)
  const headers = createHelperHeaders(connection, "application/json")
  headers.set("Content-Type", "application/json")

  let response: Response
  try {
    response = await fetch(buildHelperEndpoint(connection, SOP_SCAN_PATH), {
      method: "POST",
      headers,
      body: JSON.stringify(request),
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

  return (await response.json()) as SopScanResult
}
