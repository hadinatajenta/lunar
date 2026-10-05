import { ref } from "vue"
import { ApiError, getStoredToken } from "../../../lib/http"
import { resolveBackendURL } from "../../code-context/api/backend-url"
import type {
  CollectionExportHandlers,
  CollectionExportRequest,
  CollectionExportResult,
  CollectionExportStageEvent
} from "../types"
import {
  CollectionExportAbortedError,
  exportHelperCollection,
  exportWorkspaceCollection
} from "../api/collection-export"
import {
  REPOSITORY_ROOT_MISSING_MESSAGE,
  hasConfiguredRoot,
  requireHelperConnection,
  resolveErrorMessage,
  syncSessionScope,
  workspace
} from "./microservices-store"

const EXPORT_FAILURE_FALLBACK_MESSAGE = "The export failed. Try again."
const COLLECTION_FILE_SUFFIX = "-postman-collection.json"
const DEFAULT_SERVICE_FILE_NAME = "collection"
const OBJECT_URL_REVOKE_DELAY_MS = 1000
const UNRESOLVED_FILE_NAME_CHARACTERS = /[\\/:*?"<>|]+/g
const LEADING_DOTS = /^\.+/

export interface CollectionDownload {
  fileName: string
  serviceName: string
  requestCount: number
  folderCount: number
  blob: Blob
}

const resolveServiceName = (exportResult: CollectionExportResult, repositoryName: string): string => {
  const serviceName = exportResult.service.trim()
  return serviceName.length > 0 ? serviceName : repositoryName
}

const buildCollectionFileName = (serviceName: string): string => {
  const safeName = serviceName.replace(UNRESOLVED_FILE_NAME_CHARACTERS, "-").replace(LEADING_DOTS, "").trim()
  return `${safeName.length > 0 ? safeName : DEFAULT_SERVICE_FILE_NAME}${COLLECTION_FILE_SUFFIX}`
}

export function useCollectionExport(repositoryName: string) {
  const stageEvents = ref<CollectionExportStageEvent[]>([])
  const isExporting = ref(false)
  const failureMessage = ref("")
  const download = ref<CollectionDownload | null>(null)
  let abortController: AbortController | null = null
  let pendingObjectUrl: string | null = null

  const releaseObjectUrl = (): void => {
    if (pendingObjectUrl === null) {
      return
    }
    URL.revokeObjectURL(pendingObjectUrl)
    pendingObjectUrl = null
  }

  const buildDownload = (exportResult: CollectionExportResult): CollectionDownload => {
    const serviceName = resolveServiceName(exportResult, repositoryName)
    return {
      fileName: buildCollectionFileName(serviceName),
      serviceName,
      requestCount: exportResult.request_count,
      folderCount: exportResult.folder_count,
      blob: new Blob([JSON.stringify(exportResult.collection, null, 2)], { type: "application/json" })
    }
  }

  const runExportRequest = async (
    modelId: string,
    handlers: CollectionExportHandlers
  ): Promise<void> => {
    syncSessionScope()
    const settings = workspace.value
    if (settings === null || !hasConfiguredRoot(settings)) {
      throw new ApiError(REPOSITORY_ROOT_MISSING_MESSAGE, 0)
    }

    const request: CollectionExportRequest = {
      root: settings.root_path,
      name: repositoryName,
      model: modelId,
      backend_url: resolveBackendURL(),
      session_token: getStoredToken() ?? ""
    }

    if (settings.source === "helper") {
      await exportHelperCollection(requireHelperConnection(), request, handlers)
      return
    }

    await exportWorkspaceCollection(request, handlers)
  }

  const startExport = async (modelId: string): Promise<void> => {
    if (isExporting.value) {
      return
    }

    stageEvents.value = []
    failureMessage.value = ""
    download.value = null
    const controller = new AbortController()
    abortController = controller
    isExporting.value = true

    try {
      await runExportRequest(modelId, {
        signal: controller.signal,
        onStage: (stageEvent) => {
          stageEvents.value = [...stageEvents.value, stageEvent]
        },
        onResult: (exportResult) => {
          download.value = buildDownload(exportResult)
        }
      })
    } catch (thrownError) {
      if (!(thrownError instanceof CollectionExportAbortedError)) {
        failureMessage.value = resolveErrorMessage(thrownError, EXPORT_FAILURE_FALLBACK_MESSAGE)
      }
    } finally {
      isExporting.value = false
      abortController = null
    }
  }

  const downloadCollection = (): void => {
    const downloadEntry = download.value
    if (downloadEntry === null) {
      return
    }

    releaseObjectUrl()
    const objectUrl = URL.createObjectURL(downloadEntry.blob)
    pendingObjectUrl = objectUrl
    const anchorElement = document.createElement("a")
    anchorElement.href = objectUrl
    anchorElement.download = downloadEntry.fileName
    document.body.appendChild(anchorElement)
    anchorElement.click()
    anchorElement.remove()
    setTimeout(() => {
      if (pendingObjectUrl === objectUrl) {
        releaseObjectUrl()
      }
    }, OBJECT_URL_REVOKE_DELAY_MS)
  }

  const dispose = (): void => {
    abortController?.abort()
    abortController = null
    releaseObjectUrl()
  }

  return { stageEvents, isExporting, failureMessage, download, startExport, downloadCollection, dispose }
}
