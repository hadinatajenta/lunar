import { computed, ref, type ComputedRef, type Ref } from "vue"
import { ApiError } from "../../../lib/http.ts"
import type {
  QueryReviewApplyResult,
  QueryReviewErrorCode,
  QueryReviewRepository,
  QueryReviewRequest,
  QueryReviewResult,
  QueryReviewSource
} from "../types"

export type QueryReviewStatus = "idle" | "loading" | "success" | "empty" | "error"

export const QUERY_REVIEW_LOADING_MESSAGE =
  "Reading the branch compare diff and extracting the queries introduced on it."
export const QUERY_REVIEW_FAILURE_MESSAGE = "The query review could not be completed. Try again."
export const QUERY_REVIEW_EMPTY_MESSAGE = "No repository was found to review on the MIG page."

export const QUERY_REVIEW_MISSING_SOURCE_MESSAGE =
  "The document has no Bitbucket compare link. Paste the compare URL to extract the queries."
export const QUERY_REVIEW_APPLY_FAILURE_MESSAGE =
  "The query table could not be written to Confluence. Try again."
export const QUERY_REVIEW_APPLY_EMPTY_MESSAGE = "There is no repository to apply to Confluence."

const KNOWN_ERROR_CODES: QueryReviewErrorCode[] = ["NO_COMPARE_LINK"]

export interface QueryReviewTransport {
  request: (
    documentId: string,
    payload: QueryReviewRequest,
    signal: AbortSignal
  ) => Promise<QueryReviewResult>
  apply: (
    documentId: string,
    repositories: QueryReviewRepository[]
  ) => Promise<QueryReviewApplyResult>
}

export interface QueryReviewController {
  status: Ref<QueryReviewStatus>
  result: Ref<QueryReviewResult | null>
  errorMessage: Ref<string>
  errorCode: Ref<QueryReviewErrorCode | null>
  isApplying: Ref<boolean>
  applyError: Ref<string>
  applyResult: Ref<QueryReviewApplyResult | null>
  isLoading: ComputedRef<boolean>
  hasResult: ComputedRef<boolean>
  start: (documentId: string, model: string) => Promise<void>
  retry: () => Promise<void>
  retryWithSource: (source: QueryReviewSource) => Promise<void>
  apply: () => Promise<void>
  cancel: () => void
  resetForDocument: (documentId: string | null) => void
}

const readErrorCode = (thrownError: ApiError): QueryReviewErrorCode | null => {
  const details = thrownError.details
  if (typeof details !== "object" || details === null) {
    return null
  }
  const code = (details as Record<string, unknown>).code
  if (typeof code !== "string") {
    return null
  }
  return KNOWN_ERROR_CODES.find((knownCode) => knownCode === code) ?? null
}

export function createQueryReviewController(
  transport: QueryReviewTransport
): QueryReviewController {
  const status = ref<QueryReviewStatus>("idle")
  const result = ref<QueryReviewResult | null>(null)
  const errorMessage = ref("")
  const errorCode = ref<QueryReviewErrorCode | null>(null)
  const isApplying = ref(false)
  const applyError = ref("")
  const applyResult = ref<QueryReviewApplyResult | null>(null)

  let activeDocumentId: string | null = null
  let lastModel = ""
  let requestSequence = 0
  let abortController: AbortController | null = null

  const isLoading = computed(() => status.value === "loading")
  const hasResult = computed(() => result.value !== null)

  const abortActiveRequest = (): void => {
    abortController?.abort()
    abortController = null
  }

  const clearState = (): void => {
    result.value = null
    errorMessage.value = ""
    errorCode.value = null
    applyError.value = ""
    applyResult.value = null
  }

  const run = async (
    documentId: string,
    model: string,
    source: QueryReviewSource | undefined
  ): Promise<void> => {
    if (documentId.length === 0 || model.length === 0) {
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
      const payload: QueryReviewRequest = source === undefined ? { model } : { model, source }
      const response = await transport.request(documentId, payload, controller.signal)
      if (requestId !== requestSequence || controller.signal.aborted) {
        return
      }
      result.value = response
      status.value = response.repositories.length === 0 ? "empty" : "success"
    } catch (thrownError) {
      if (requestId !== requestSequence || controller.signal.aborted) {
        return
      }
      if (thrownError instanceof ApiError) {
        const code = readErrorCode(thrownError)
        if (code === "NO_COMPARE_LINK") {
          errorCode.value = code
          errorMessage.value = QUERY_REVIEW_MISSING_SOURCE_MESSAGE
        } else {
          errorMessage.value = thrownError.message || QUERY_REVIEW_FAILURE_MESSAGE
        }
      } else if (thrownError instanceof Error && thrownError.message.length > 0) {
        errorMessage.value = thrownError.message
      } else {
        errorMessage.value = QUERY_REVIEW_FAILURE_MESSAGE
      }
      status.value = "error"
    } finally {
      if (requestId === requestSequence) {
        abortController = null
      }
    }
  }

  const start = (documentId: string, model: string): Promise<void> =>
    run(documentId, model, undefined)

  const retry = (): Promise<void> => {
    if (activeDocumentId === null || lastModel.length === 0) {
      return Promise.resolve()
    }
    return run(activeDocumentId, lastModel, undefined)
  }

  const retryWithSource = (source: QueryReviewSource): Promise<void> => {
    const trimmedUrl = source.compare_url.trim()
    if (activeDocumentId === null || lastModel.length === 0 || trimmedUrl.length === 0) {
      return Promise.resolve()
    }
    return run(activeDocumentId, lastModel, { ...source, compare_url: trimmedUrl })
  }

  const readApplyErrorMessage = (thrownError: unknown): string => {
    if (thrownError instanceof ApiError) {
      return thrownError.message || QUERY_REVIEW_APPLY_FAILURE_MESSAGE
    }
    if (thrownError instanceof Error && thrownError.message.length > 0) {
      return thrownError.message
    }
    return QUERY_REVIEW_APPLY_FAILURE_MESSAGE
  }

  const apply = async (): Promise<void> => {
    const currentResult = result.value
    if (currentResult === null || activeDocumentId === null || isApplying.value) {
      return
    }
    if (currentResult.repositories.length === 0) {
      applyError.value = QUERY_REVIEW_APPLY_EMPTY_MESSAGE
      return
    }
    const applyDocumentId = activeDocumentId
    isApplying.value = true
    applyError.value = ""
    applyResult.value = null

    try {
      const applied = await transport.apply(applyDocumentId, currentResult.repositories)
      if (activeDocumentId === applyDocumentId) {
        applyResult.value = applied
      }
    } catch (thrownError) {
      if (activeDocumentId === applyDocumentId) {
        applyError.value = readApplyErrorMessage(thrownError)
      }
    } finally {
      isApplying.value = false
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
    clearState()
  }

  return {
    status,
    result,
    errorMessage,
    errorCode,
    isApplying,
    applyError,
    applyResult,
    isLoading,
    hasResult,
    start,
    retry,
    retryWithSource,
    apply,
    cancel,
    resetForDocument
  }
}
