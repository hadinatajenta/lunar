import { HELPER_UNREACHABLE_MESSAGE, HelperConnectionError } from "../../microservices/api/helper-api"
import {
  REPOSITORY_ROOT_MISSING_MESSAGE,
  resolveHelperConnection,
  workspace
} from "../../microservices/composables/microservices-store"
import type { HelperConnection } from "../../microservices/types"
import { applySop, requestSopGenerate } from "../api/confluence-api"
import { requestSopScan } from "../api/sop-api"
import {
  SopHelperUnreachableError,
  buildSopScanRequest,
  createSopDocsController,
  type SopDocsController,
  type SopDocsTransports
} from "./useSopDocs"

const HELPER_NOT_CONNECTED_MESSAGE =
  "The Lunar helper is not connected. Start the helper on this machine, then add its token in Settings."

const requireConnection = (): HelperConnection => {
  const connection = resolveHelperConnection()
  if (connection === null) {
    const helperUrl = workspace.value?.helper_url.trim() ?? ""
    const detail = helperUrl.length > 0 ? ` (${helperUrl})` : ""
    throw new SopHelperUnreachableError(`${HELPER_NOT_CONNECTED_MESSAGE}${detail}`, helperUrl)
  }
  return connection
}

const requireRootPath = (): string => {
  const rootPath = workspace.value?.root_path.trim() ?? ""
  if (rootPath.length === 0) {
    throw new SopHelperUnreachableError(REPOSITORY_ROOT_MISSING_MESSAGE, "")
  }
  return rootPath
}

const sopDocsTransports: SopDocsTransports = {
  generate: (documentId, payload, signal) => requestSopGenerate(documentId, payload, signal),
  scan: async (target, signal) => {
    const connection = requireConnection()
    const rootPath = requireRootPath()
    try {
      return await requestSopScan(connection, buildSopScanRequest(target, rootPath), signal)
    } catch (thrownError) {
      if (thrownError instanceof HelperConnectionError) {
        throw new SopHelperUnreachableError(
          `${HELPER_UNREACHABLE_MESSAGE} (${thrownError.helperUrl})`,
          thrownError.helperUrl
        )
      }
      throw thrownError
    }
  },
  apply: (documentId, payload) => applySop(documentId, payload)
}

const sopDocsController = createSopDocsController(sopDocsTransports)

export function useSopDocs(): SopDocsController {
  return sopDocsController
}
