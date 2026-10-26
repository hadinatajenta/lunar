import { HELPER_UNREACHABLE_MESSAGE, HelperConnectionError } from "../../microservices/api/helper-api"
import {
  REPOSITORY_ROOT_MISSING_MESSAGE,
  resolveHelperConnection,
  workspace
} from "../../microservices/composables/microservices-store"
import type { HelperConnection } from "../../microservices/types"
import { applyUtDocs, fetchMigTargets, requestUtDocs } from "../api/confluence-api"
import { fetchUtScreenshotBlob, fetchUtScreenshotObjectUrl, streamUtRun } from "../api/ut-docs-api"
import type { UtDocsApplyScreenshot, UtRunDone } from "../types"
import {
  UtHelperUnreachableError,
  createUtDocsController,
  type UtDocsController,
  type UtDocsTransports
} from "./useUtDocs"

const HELPER_NOT_CONNECTED_MESSAGE =
  "The Lunar helper is not connected. Start the helper on this machine, then add its token in Settings."
const SCREENSHOT_ENCODE_FAILURE_MESSAGE =
  "The run screenshot could not be encoded for Confluence."

const requireConnection = (): HelperConnection => {
  const connection = resolveHelperConnection()
  if (connection === null) {
    const helperUrl = workspace.value?.helper_url.trim() ?? ""
    const detail = helperUrl.length > 0 ? ` (${helperUrl})` : ""
    throw new UtHelperUnreachableError(`${HELPER_NOT_CONNECTED_MESSAGE}${detail}`, helperUrl)
  }
  return connection
}

const requireRootPath = (): string => {
  const rootPath = workspace.value?.root_path.trim() ?? ""
  if (rootPath.length === 0) {
    throw new UtHelperUnreachableError(REPOSITORY_ROOT_MISSING_MESSAGE, "")
  }
  return rootPath
}

const resolveScreenshotPath = (result: UtRunDone): string =>
  result.screenshot_url.trim().length > 0
    ? result.screenshot_url
    : `/ut-runs/${encodeURIComponent(result.run_id)}/screenshot.png`

const resolveScreenshotFilename = (result: UtRunDone): string => {
  const repositoryName = result.repo.trim().length > 0 ? result.repo.trim() : "ut-run"
  return `${repositoryName}-${result.run_id}.png`
}

const blobToBase64 = (screenshotBlob: Blob): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = typeof reader.result === "string" ? reader.result : ""
      const separatorIndex = result.indexOf(",")
      resolve(separatorIndex >= 0 ? result.slice(separatorIndex + 1) : result)
    }
    reader.onerror = () => reject(new Error(SCREENSHOT_ENCODE_FAILURE_MESSAGE))
    reader.readAsDataURL(screenshotBlob)
  })

const utDocsTransports: UtDocsTransports = {
  loadTargets: (documentId, signal) => fetchMigTargets(documentId, signal),
  streamRun: async (request, signal, onEvent) => {
    const connection = requireConnection()
    const rootPath = requireRootPath()
    try {
      await streamUtRun(connection, { root: rootPath, ...request }, { signal, onEvent })
    } catch (thrownError) {
      if (thrownError instanceof HelperConnectionError) {
        throw new UtHelperUnreachableError(
          `${HELPER_UNREACHABLE_MESSAGE} (${thrownError.helperUrl})`,
          thrownError.helperUrl
        )
      }
      throw thrownError
    }
  },
  generateDocument: (documentId, payload, signal) => requestUtDocs(documentId, payload, signal),
  loadScreenshotUrl: (result, signal) => {
    const connection = requireConnection()
    return fetchUtScreenshotObjectUrl(connection, resolveScreenshotPath(result), signal)
  },
  releaseScreenshotUrl: (objectUrl) => {
    URL.revokeObjectURL(objectUrl)
  },
  loadScreenshotPayload: async (result, signal) => {
    const connection = requireConnection()
    const screenshotBlob = await fetchUtScreenshotBlob(connection, resolveScreenshotPath(result), signal)
    const contentBase64 = await blobToBase64(screenshotBlob)
    const screenshot: UtDocsApplyScreenshot = {
      filename: resolveScreenshotFilename(result),
      content_base64: contentBase64
    }
    return screenshot
  },
  apply: (documentId, payload, signal) => applyUtDocs(documentId, payload, signal)
}

const utDocsController = createUtDocsController(utDocsTransports)

export function useUtDocs(): UtDocsController {
  return utDocsController
}
