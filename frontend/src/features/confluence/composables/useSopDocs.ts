import { computed, ref, type ComputedRef, type Ref } from "vue"
import type {
  SopApplyRequest,
  SopApplyResult,
  SopDeployMode,
  SopGenerateRepository,
  SopGenerateRequest,
  SopGenerateResult,
  SopScanRequest,
  SopScanResult,
  SopScanTarget
} from "../types"

export type SopDocsStatus = "idle" | "loading" | "success" | "error"

export const SOP_DOCS_LOADING_MESSAGE =
  "Generating the SOP Deploy and SOP Operasional plan from the branch changes."
export const SOP_DOCS_FAILURE_MESSAGE = "The SOP documents could not be generated. Try again."
export const SOP_DOCS_APPLY_FAILURE_MESSAGE =
  "The SOP documents could not be written to Confluence. Try again."

export class SopHelperUnreachableError extends Error {
  readonly helperUrl: string

  constructor(message: string, helperUrl: string) {
    super(message)
    this.name = "SopHelperUnreachableError"
    this.helperUrl = helperUrl
  }
}

export const buildSopScanRequest = (target: SopScanTarget, rootPath: string): SopScanRequest => {
  const baseRef = target.base_ref?.trim() ?? ""
  return baseRef.length > 0
    ? { root: rootPath, repo: target.repository, branch: target.branch, base_ref: baseRef }
    : { root: rootPath, repo: target.repository, branch: target.branch }
}

export const buildSopScanTarget = (repository: {
  repository: string
  branch: string
  base_ref?: string
}): SopScanTarget => {
  const baseRef = repository.base_ref?.trim() ?? ""
  return baseRef.length > 0
    ? { repository: repository.repository, branch: repository.branch, base_ref: baseRef }
    : { repository: repository.repository, branch: repository.branch }
}

export interface SopDocsTransports {
  generate: (
    documentId: string,
    payload: SopGenerateRequest,
    signal: AbortSignal
  ) => Promise<SopGenerateResult>
  scan: (target: SopScanTarget, signal: AbortSignal) => Promise<SopScanResult>
  apply: (documentId: string, payload: SopApplyRequest) => Promise<SopApplyResult>
}

export interface SopDocsController {
  status: Ref<SopDocsStatus>
  result: Ref<SopGenerateResult | null>
  scans: Ref<Record<string, SopScanResult>>
  errorMessage: Ref<string>
  helperUrl: Ref<string>
  isHelperUnreachable: Ref<boolean>
  isApplying: Ref<boolean>
  applyError: Ref<string>
  applyResult: Ref<SopApplyResult | null>
  isLoading: ComputedRef<boolean>
  hasResult: ComputedRef<boolean>
  start: (documentId: string, model: string) => Promise<void>
  retry: () => Promise<void>
  apply: () => Promise<void>
  cancel: () => void
  resetForDocument: (documentId: string | null) => void
}

export function createSopDocsController(transports: SopDocsTransports): SopDocsController {
  const status = ref<SopDocsStatus>("idle")
  const result = ref<SopGenerateResult | null>(null)
  const scans = ref<Record<string, SopScanResult>>({})
  const errorMessage = ref("")
  const helperUrl = ref("")
  const isHelperUnreachable = ref(false)
  const isApplying = ref(false)
  const applyError = ref("")
  const applyResult = ref<SopApplyResult | null>(null)

  let activeDocumentId: string | null = null
  let lastModel = ""
  let requestSequence = 0
  let abortController: AbortController | null = null

  const isLoading = computed(() => status.value === "loading")
  const hasResult = computed(() => result.value !== null)

  const isStale = (requestId: number, signal: AbortSignal): boolean =>
    requestId !== requestSequence || signal.aborted

  const abortActiveRequest = (): void => {
    abortController?.abort()
    abortController = null
  }

  const clearState = (): void => {
    result.value = null
    scans.value = {}
    errorMessage.value = ""
    helperUrl.value = ""
    isHelperUnreachable.value = false
    applyError.value = ""
    applyResult.value = null
  }

  const scanRepositories = async (
    repositories: SopGenerateRepository[],
    signal: AbortSignal,
    requestId: number
  ): Promise<SopScanResult[] | null> => {
    const collected: SopScanResult[] = []
    const collectedByRepository: Record<string, SopScanResult> = {}
    for (const repository of repositories) {
      if (isStale(requestId, signal)) {
        return null
      }
      const scan = await transports.scan(buildSopScanTarget(repository), signal)
      if (isStale(requestId, signal)) {
        return null
      }
      collected.push(scan)
      collectedByRepository[repository.repository] = scan
      scans.value = { ...collectedByRepository }
    }
    return collected
  }

  const run = async (documentId: string, model: string): Promise<void> => {
    if (documentId.length === 0) {
      return
    }

    requestSequence += 1
    const requestId = requestSequence
    abortActiveRequest()
    const controller = new AbortController()
    abortController = controller

    activeDocumentId = documentId
    lastModel = model
    clearState()
    status.value = "loading"

    try {
      const discovered = await transports.generate(documentId, { model }, controller.signal)
      if (isStale(requestId, controller.signal)) {
        return
      }
      const scanned = await scanRepositories(discovered.repositories, controller.signal, requestId)
      if (scanned === null || isStale(requestId, controller.signal)) {
        return
      }
      const generated = await transports.generate(
        documentId,
        { model, repositories: scanned },
        controller.signal
      )
      if (isStale(requestId, controller.signal)) {
        return
      }
      result.value = generated
      status.value = "success"
    } catch (thrownError) {
      if (isStale(requestId, controller.signal)) {
        return
      }
      scans.value = {}
      if (thrownError instanceof SopHelperUnreachableError) {
        isHelperUnreachable.value = true
        helperUrl.value = thrownError.helperUrl
        errorMessage.value = thrownError.message
      } else if (thrownError instanceof Error && thrownError.message.length > 0) {
        errorMessage.value = thrownError.message
      } else {
        errorMessage.value = SOP_DOCS_FAILURE_MESSAGE
      }
      status.value = "error"
    } finally {
      if (requestId === requestSequence) {
        abortController = null
      }
    }
  }

  const start = (documentId: string, model: string): Promise<void> => run(documentId, model)

  const retry = (): Promise<void> => {
    if (activeDocumentId === null) {
      return Promise.resolve()
    }
    return run(activeDocumentId, lastModel)
  }

  const readApplyErrorMessage = (thrownError: unknown): string => {
    if (thrownError instanceof Error && thrownError.message.length > 0) {
      return thrownError.message
    }
    return SOP_DOCS_APPLY_FAILURE_MESSAGE
  }

  const apply = async (): Promise<void> => {
    const currentResult = result.value
    const applyDocumentId = activeDocumentId
    if (currentResult === null || applyDocumentId === null || isApplying.value) {
      return
    }
    const mode: SopDeployMode = currentResult.deploy.exists ? "update" : "create"

    isApplying.value = true
    applyError.value = ""
    applyResult.value = null

    try {
      const applied = await transports.apply(applyDocumentId, {
        deploy: { mode, managed_block: currentResult.deploy.managed_block },
        operasional: { create_children: true }
      })
      if (activeDocumentId === applyDocumentId) {
        applyResult.value = applied
      }
    } catch (thrownError) {
      if (activeDocumentId === applyDocumentId) {
        applyError.value = readApplyErrorMessage(thrownError)
      }
    } finally {
      if (activeDocumentId === applyDocumentId) {
        isApplying.value = false
      }
    }
  }

  const cancel = (): void => {
    if (status.value !== "loading") {
      return
    }
    requestSequence += 1
    abortActiveRequest()
    status.value = "idle"
    clearState()
  }

  const resetForDocument = (documentId: string | null): void => {
    requestSequence += 1
    abortActiveRequest()
    activeDocumentId = documentId
    lastModel = ""
    status.value = "idle"
    isApplying.value = false
    clearState()
  }

  return {
    status,
    result,
    scans,
    errorMessage,
    helperUrl,
    isHelperUnreachable,
    isApplying,
    applyError,
    applyResult,
    isLoading,
    hasResult,
    start,
    retry,
    apply,
    cancel,
    resetForDocument
  }
}
