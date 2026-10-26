import { computed, ref, type ComputedRef, type Ref } from "vue"
import type { UtRunEvent } from "../api/ut-run-events"
import type {
  MigSkippedRepository,
  MigTarget,
  MigTargetsResult,
  UtDocsApplyMockTest,
  UtDocsApplyRepository,
  UtDocsApplyRequest,
  UtDocsApplyResult,
  UtDocsApplyRow,
  UtDocsApplyScreenshot,
  UtDocsMockTest,
  UtDocsRepositoryRun,
  UtDocsRequest,
  UtDocsResult,
  UtDocsRow,
  UtRunDone,
  UtRunPhase
} from "../types"

export type UtDocsStatus =
  | "idle"
  | "loading-targets"
  | "running"
  | "generating"
  | "success"
  | "empty"
  | "error"

export interface UtRunStreamRequest {
  repo: string
  branch: string
  base_ref?: string
}

export const UT_DOCS_MISSING_RESULT_MESSAGE =
  "The run finished without a result. Run the unit tests again."
export const UT_DOCS_FAILURE_MESSAGE = "The unit test run could not be completed. Try again."
export const UT_DOCS_EMPTY_MESSAGE =
  "No repository was found to run the unit tests on this MIG page."
export const UT_DOCS_SCREENSHOT_FAILURE_MESSAGE = "The run screenshot could not be loaded."
export const UT_DOCS_APPLY_FAILURE_MESSAGE =
  "The UT documentation could not be written to Confluence. Try again."
export const UT_DOCS_APPLY_EMPTY_MESSAGE = "There is no repository to apply to Confluence."

export class UtHelperUnreachableError extends Error {
  readonly helperUrl: string

  constructor(message: string, helperUrl: string) {
    super(message)
    this.name = "UtHelperUnreachableError"
    this.helperUrl = helperUrl
  }
}

export interface UtDocsTransports {
  loadTargets: (documentId: string, signal: AbortSignal) => Promise<MigTargetsResult>
  streamRun: (
    request: UtRunStreamRequest,
    signal: AbortSignal,
    onEvent: (event: UtRunEvent) => void
  ) => Promise<void>
  generateDocument: (
    documentId: string,
    payload: UtDocsRequest,
    signal: AbortSignal
  ) => Promise<UtDocsResult>
  loadScreenshotUrl: (result: UtRunDone, signal: AbortSignal) => Promise<string>
  releaseScreenshotUrl: (objectUrl: string) => void
  loadScreenshotPayload: (
    result: UtRunDone,
    signal: AbortSignal
  ) => Promise<UtDocsApplyScreenshot>
  apply: (
    documentId: string,
    payload: UtDocsApplyRequest,
    signal: AbortSignal
  ) => Promise<UtDocsApplyResult>
}

export interface UtDocsController {
  status: Ref<UtDocsStatus>
  phase: Ref<UtRunPhase | null>
  outputText: Ref<string>
  currentRepository: Ref<string>
  migPageTitle: Ref<string>
  targets: Ref<MigTarget[]>
  skippedRepositories: Ref<MigSkippedRepository[]>
  runResults: Ref<UtRunDone[]>
  generatedDocument: Ref<UtDocsResult | null>
  errorMessage: Ref<string>
  helperUrl: Ref<string>
  isHelperUnreachable: Ref<boolean>
  screenshotUrls: Ref<Record<string, string>>
  screenshotError: Ref<string>
  isApplying: Ref<boolean>
  applyError: Ref<string>
  applyResult: Ref<UtDocsApplyResult | null>
  isRunning: ComputedRef<boolean>
  isGenerating: ComputedRef<boolean>
  hasResult: ComputedRef<boolean>
  start: (documentId: string, model: string) => Promise<void>
  cancel: () => void
  retry: () => Promise<void>
  apply: () => Promise<void>
  releaseScreenshots: () => void
  resetForDocument: (documentId: string | null) => void
}

const EMPTY_SCREENSHOT: UtDocsApplyScreenshot = { filename: "", content_base64: "" }

const repositoryKey = (done: UtRunDone): string =>
  done.repo.trim().length > 0 ? done.repo.trim() : done.run_id

const toRepositoryRun = (done: UtRunDone): UtDocsRepositoryRun => ({
  repository: done.repo,
  branch: done.branch,
  base_ref: done.base_ref,
  test_files: done.test_files,
  blocked_files: done.blocked_files,
  run: done.run,
  tests: done.tests,
  untested_functions: done.untested_functions
})

const toApplyRow = (row: UtDocsRow): UtDocsApplyRow => ({
  function: row.function,
  scenario: row.scenario,
  expected_result: row.expected_result,
  input_data: row.input_data,
  result: row.result
})

const toApplyMockTest = (mockTest: UtDocsMockTest): UtDocsApplyMockTest => ({
  function: mockTest.function,
  scenario: mockTest.scenario,
  expected_result: mockTest.expected_result,
  input_data: mockTest.input_data,
  code: mockTest.code
})

const selectApplyRows = (
  rows: UtDocsRow[],
  repository: string,
  isSingleRepository: boolean
): UtDocsApplyRow[] => {
  const selected = isSingleRepository
    ? rows
    : rows.filter((row) => row.repository === repository)
  return selected.map(toApplyRow)
}

const selectApplyMockTests = (
  mockTests: UtDocsMockTest[],
  rows: UtDocsRow[],
  repository: string,
  isSingleRepository: boolean
): UtDocsApplyMockTest[] => {
  if (isSingleRepository) {
    return mockTests.map(toApplyMockTest)
  }
  const repositoryTestNames = new Set(
    rows.filter((row) => row.repository === repository).map((row) => row.test_function_name)
  )
  return mockTests
    .filter((mockTest) => repositoryTestNames.has(mockTest.test_function_name))
    .map(toApplyMockTest)
}

const buildStreamRequest = (target: MigTarget): UtRunStreamRequest => {
  const baseRef = target.base_ref?.trim() ?? ""
  return baseRef.length > 0
    ? { repo: target.repository, branch: target.branch, base_ref: baseRef }
    : { repo: target.repository, branch: target.branch }
}

export function createUtDocsController(transports: UtDocsTransports): UtDocsController {
  const status = ref<UtDocsStatus>("idle")
  const phase = ref<UtRunPhase | null>(null)
  const outputText = ref("")
  const currentRepository = ref("")
  const migPageTitle = ref("")
  const targets = ref<MigTarget[]>([])
  const skippedRepositories = ref<MigSkippedRepository[]>([])
  const runResults = ref<UtRunDone[]>([])
  const generatedDocument = ref<UtDocsResult | null>(null)
  const errorMessage = ref("")
  const helperUrl = ref("")
  const isHelperUnreachable = ref(false)
  const screenshotUrls = ref<Record<string, string>>({})
  const screenshotError = ref("")
  const isApplying = ref(false)
  const applyError = ref("")
  const applyResult = ref<UtDocsApplyResult | null>(null)

  let activeDocumentId: string | null = null
  let lastModel = ""
  let requestSequence = 0
  let abortController: AbortController | null = null

  const isRunning = computed(() => status.value === "running")
  const isGenerating = computed(
    () =>
      status.value === "loading-targets" ||
      status.value === "running" ||
      status.value === "generating"
  )
  const hasResult = computed(() => generatedDocument.value !== null)

  const isStale = (requestId: number, signal: AbortSignal): boolean =>
    requestId !== requestSequence || signal.aborted

  const releaseScreenshots = (): void => {
    for (const objectUrl of Object.values(screenshotUrls.value)) {
      transports.releaseScreenshotUrl(objectUrl)
    }
    screenshotUrls.value = {}
  }

  const abortActiveRequest = (): void => {
    abortController?.abort()
    abortController = null
  }

  const clearState = (): void => {
    phase.value = null
    outputText.value = ""
    currentRepository.value = ""
    migPageTitle.value = ""
    targets.value = []
    skippedRepositories.value = []
    runResults.value = []
    generatedDocument.value = null
    errorMessage.value = ""
    helperUrl.value = ""
    isHelperUnreachable.value = false
    screenshotError.value = ""
    applyError.value = ""
    applyResult.value = null
    releaseScreenshots()
  }

  const storeRepositoryScreenshot = async (
    done: UtRunDone,
    signal: AbortSignal,
    requestId: number
  ): Promise<void> => {
    try {
      const objectUrl = await transports.loadScreenshotUrl(done, signal)
      if (isStale(requestId, signal)) {
        transports.releaseScreenshotUrl(objectUrl)
        return
      }
      screenshotUrls.value = { ...screenshotUrls.value, [repositoryKey(done)]: objectUrl }
    } catch {
      if (isStale(requestId, signal)) {
        return
      }
      screenshotError.value = UT_DOCS_SCREENSHOT_FAILURE_MESSAGE
    }
  }

  const run = async (documentId: string, model: string): Promise<void> => {
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
    status.value = "loading-targets"

    try {
      const targetsResult = await transports.loadTargets(documentId, controller.signal)
      if (isStale(requestId, controller.signal)) {
        return
      }
      migPageTitle.value = targetsResult.mig_page_title
      targets.value = targetsResult.targets
      skippedRepositories.value = targetsResult.skipped_repositories

      if (targetsResult.targets.length === 0) {
        status.value = "empty"
        return
      }

      status.value = "running"
      const collected: UtRunDone[] = []

      for (const target of targetsResult.targets) {
        if (isStale(requestId, controller.signal)) {
          return
        }
        currentRepository.value = target.repository
        phase.value = null
        outputText.value = ""
        screenshotError.value = ""

        let done: UtRunDone | null = null
        await transports.streamRun(
          buildStreamRequest(target),
          controller.signal,
          (event: UtRunEvent) => {
            if (event.kind === "phase") {
              phase.value = event.phase
              return
            }
            if (event.kind === "output") {
              outputText.value += event.text
              return
            }
            done = event.result
          }
        )
        if (isStale(requestId, controller.signal)) {
          return
        }
        if (done === null) {
          throw new Error(UT_DOCS_MISSING_RESULT_MESSAGE)
        }

        collected.push(done)
        runResults.value = [...collected]
        await storeRepositoryScreenshot(done, controller.signal, requestId)
      }

      currentRepository.value = ""
      status.value = "generating"

      const generated = await transports.generateDocument(
        documentId,
        { model, repositories: collected.map(toRepositoryRun) },
        controller.signal
      )
      if (isStale(requestId, controller.signal)) {
        return
      }
      generatedDocument.value = generated
      status.value = "success"
    } catch (thrownError) {
      if (isStale(requestId, controller.signal)) {
        return
      }
      if (thrownError instanceof UtHelperUnreachableError) {
        isHelperUnreachable.value = true
        helperUrl.value = thrownError.helperUrl
        errorMessage.value = thrownError.message
      } else if (thrownError instanceof Error && thrownError.message.length > 0) {
        errorMessage.value = thrownError.message
      } else {
        errorMessage.value = UT_DOCS_FAILURE_MESSAGE
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
    if (activeDocumentId === null || lastModel.length === 0) {
      return Promise.resolve()
    }
    return run(activeDocumentId, lastModel)
  }

  const readApplyErrorMessage = (thrownError: unknown): string => {
    if (thrownError instanceof Error && thrownError.message.length > 0) {
      return thrownError.message
    }
    return UT_DOCS_APPLY_FAILURE_MESSAGE
  }

  const apply = async (): Promise<void> => {
    const applyDocumentId = activeDocumentId
    if (applyDocumentId === null || isApplying.value) {
      return
    }
    if (runResults.value.length === 0) {
      applyError.value = UT_DOCS_APPLY_EMPTY_MESSAGE
      return
    }
    const currentDocument = generatedDocument.value
    if (currentDocument === null) {
      return
    }

    isApplying.value = true
    applyError.value = ""
    applyResult.value = null
    const controller = new AbortController()

    try {
      const isSingleRepository = runResults.value.length === 1
      const repositories: UtDocsApplyRepository[] = []
      for (const done of runResults.value) {
        const screenshot = await transports.loadScreenshotPayload(done, controller.signal)
        repositories.push({
          repository: done.repo,
          screenshot: screenshot ?? EMPTY_SCREENSHOT,
          rows: selectApplyRows(currentDocument.rows, done.repo, isSingleRepository),
          mock_tests: selectApplyMockTests(
            currentDocument.mock_tests,
            currentDocument.rows,
            done.repo,
            isSingleRepository
          )
        })
      }
      const applied = await transports.apply(
        applyDocumentId,
        { repositories },
        controller.signal
      )
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
    if (!isGenerating.value) {
      return
    }
    requestSequence += 1
    abortActiveRequest()
    status.value = "idle"
    errorMessage.value = ""
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
    phase,
    outputText,
    currentRepository,
    migPageTitle,
    targets,
    skippedRepositories,
    runResults,
    generatedDocument,
    errorMessage,
    helperUrl,
    isHelperUnreachable,
    screenshotUrls,
    screenshotError,
    isApplying,
    applyError,
    applyResult,
    isRunning,
    isGenerating,
    hasResult,
    start,
    cancel,
    retry,
    apply,
    releaseScreenshots,
    resetForDocument
  }
}
