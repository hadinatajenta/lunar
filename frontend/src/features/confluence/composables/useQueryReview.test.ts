import assert from "node:assert/strict"
import { describe, test } from "node:test"
import { ApiError } from "../../../lib/http.ts"
import type {
  QueryReviewApplyResult,
  QueryReviewEntry,
  QueryReviewRepository,
  QueryReviewRequest,
  QueryReviewResult
} from "../types.ts"
import {
  QUERY_REVIEW_APPLY_EMPTY_MESSAGE,
  createQueryReviewController,
  type QueryReviewTransport
} from "./useQueryReview.ts"

interface RecordedCall {
  documentId: string
  payload: QueryReviewRequest
  signal: AbortSignal
}

const buildEntry = (overrides: Partial<QueryReviewEntry> = {}): QueryReviewEntry => ({
  no: 1,
  feature: "[micro-payment] Get user by PN",
  repository: "micro-payment",
  table: "user_data",
  action_type: "SELECT",
  query_statement: "select id, username from user_data where pn = ?",
  index: "pn",
  create_index: "CREATE INDEX idx_user_data_pn ON user_data (pn);",
  file: "internal/repo/dao.go",
  line: 42,
  notes: "",
  ...overrides
})

const buildRepository = (
  overrides: Partial<QueryReviewRepository> = {}
): QueryReviewRepository => ({
  repository: "micro-payment",
  project_key: "NME",
  source_branch: "feat/CRMMS-123",
  target_branch: "development",
  compare_url:
    "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
  status: "success",
  analyzed_files: [{ path: "internal/repo/dao.go", added_lines: 12, language: "go" }],
  skipped_files: [{ path: "internal/repo/dao_test.go", reason: "test file" }],
  queries: [buildEntry()],
  summary: "One SELECT was introduced on this branch.",
  truncated: false,
  error: "",
  ...overrides
})

const buildResult = (overrides: Partial<QueryReviewResult> = {}): QueryReviewResult => ({
  document_id: "QR-201",
  model: "DeepSeek Flash",
  prompt_version: "queryreview-v1",
  mig_page_id: "2086971956",
  mig_page_title: "CRMMS-76972-MIG",
  skipped_repositories: [{ repository: "fe-mms", reason: "excluded repository" }],
  repositories: [buildRepository()],
  ...overrides
})

const buildApplyResult = (overrides: Partial<QueryReviewApplyResult> = {}): QueryReviewApplyResult => ({
  page_id: "2144721484",
  version: 12,
  url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721484",
  ...overrides
})

const createTransport = (
  request: QueryReviewTransport["request"],
  apply: QueryReviewTransport["apply"] = async () => buildApplyResult()
): QueryReviewTransport => ({ request, apply })

const createDeferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

const createRecordingTransport = (
  run: (call: RecordedCall) => Promise<QueryReviewResult>
): { calls: RecordedCall[]; transport: QueryReviewTransport } => {
  const calls: RecordedCall[] = []
  return {
    calls,
    transport: {
      request: (documentId, payload, signal) => {
        const call: RecordedCall = { documentId, payload, signal }
        calls.push(call)
        return run(call)
      },
      apply: async () => buildApplyResult()
    }
  }
}

describe("useQueryReview state machine", () => {
  test("starts idle with no result", () => {
    const controller = createQueryReviewController(createTransport(async () => buildResult()))

    assert.equal(controller.status.value, "idle")
    assert.equal(controller.result.value, null)
    assert.equal(controller.hasResult.value, false)
    assert.equal(controller.isLoading.value, false)
  })

  test("a successful request moves through loading to success and exposes the result", async () => {
    const result = buildResult()
    const { calls, transport } = createRecordingTransport(async () => result)
    const controller = createQueryReviewController(transport)

    const startPromise = controller.start("QR-201", "DeepSeek Flash")
    assert.equal(controller.status.value, "loading")
    assert.equal(controller.isLoading.value, true)
    await startPromise

    assert.equal(controller.status.value, "success")
    assert.equal(controller.isLoading.value, false)
    assert.equal(controller.hasResult.value, true)
    assert.deepEqual(controller.result.value, result)
    assert.equal(
      controller.result.value?.repositories[0].queries[0].create_index,
      "CREATE INDEX idx_user_data_pn ON user_data (pn);"
    )
    assert.equal(calls.length, 1)
    assert.equal(calls[0].documentId, "QR-201")
    assert.deepEqual(calls[0].payload, { model: "DeepSeek Flash" })
  })

  test("a multi-repository result keeps every repository in response order", async () => {
    const repositories = [
      buildRepository({ repository: "micro-payment" }),
      buildRepository({
        repository: "micro-notification",
        queries: [buildEntry({ no: 1, repository: "micro-notification" })]
      })
    ]
    const result = buildResult({ repositories })
    const controller = createQueryReviewController(createTransport(async () => result))

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.status.value, "success")
    const repositoryNames = controller.result.value?.repositories.map((entry) => entry.repository)
    assert.deepEqual(repositoryNames, ["micro-payment", "micro-notification"])
  })

  test("a repository error status does not fail the other repositories", async () => {
    const repositories = [
      buildRepository({ repository: "micro-payment" }),
      buildRepository({
        repository: "micro-notification",
        status: "error",
        queries: [],
        error: "The compare diff could not be fetched."
      }),
      buildRepository({ repository: "micro-report", status: "no_queries", queries: [] })
    ]
    const result = buildResult({ repositories })
    const controller = createQueryReviewController(createTransport(async () => result))

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.status.value, "success")
    assert.equal(controller.hasResult.value, true)
    assert.equal(controller.result.value?.repositories.length, 3)
    assert.equal(controller.result.value?.repositories[1].status, "error")
    assert.equal(
      controller.result.value?.repositories[1].error,
      "The compare diff could not be fetched."
    )
  })

  test("an empty repository list becomes the empty state with the result retained", async () => {
    const result = buildResult({ repositories: [], skipped_repositories: [] })
    const controller = createQueryReviewController(createTransport(async () => result))

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.status.value, "empty")
    assert.equal(controller.hasResult.value, true)
    assert.deepEqual(controller.result.value, result)
  })

  test("a generic ApiError becomes the error state with the backend message", async () => {
    const controller = createQueryReviewController(
      createTransport(async () => {
        throw new ApiError("The model provider did not answer", 502)
      })
    )

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.status.value, "error")
    assert.equal(controller.errorMessage.value, "The model provider did not answer")
    assert.equal(controller.errorCode.value, null)
    assert.equal(controller.hasResult.value, false)
  })

  test("NO_COMPARE_LINK keeps the manual source flow available", async () => {
    const controller = createQueryReviewController(
      createTransport(async () => {
        throw new ApiError("no compare link", 422, { code: "NO_COMPARE_LINK", message: "no compare link" })
      })
    )

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.status.value, "error")
    assert.equal(controller.errorCode.value, "NO_COMPARE_LINK")
    assert.match(controller.errorMessage.value, /compare URL/)
  })

  test("retryWithSource sends the pasted source and can succeed", async () => {
    let attempt = 0
    const { calls, transport } = createRecordingTransport(async () => {
      attempt += 1
      if (attempt === 1) {
        throw new ApiError("no compare link", 422, { code: "NO_COMPARE_LINK" })
      }
      return buildResult()
    })
    const controller = createQueryReviewController(transport)

    await controller.start("QR-201", "DeepSeek Flash")
    assert.equal(controller.status.value, "error")

    await controller.retryWithSource({
      compare_url: "https://bitbucket.bri.co.id/projects/NME/repos/payment/compare/diff"
    })

    assert.equal(controller.status.value, "success")
    assert.equal(calls.length, 2)
    assert.deepEqual(calls[1].payload, {
      model: "DeepSeek Flash",
      source: { compare_url: "https://bitbucket.bri.co.id/projects/NME/repos/payment/compare/diff" }
    })
  })

  test("retry repeats the last request after a failure", async () => {
    let attempt = 0
    const { calls, transport } = createRecordingTransport(async () => {
      attempt += 1
      if (attempt === 1) {
        throw new Error("network down")
      }
      return buildResult()
    })
    const controller = createQueryReviewController(transport)

    await controller.start("QR-201", "DeepSeek Flash")
    assert.equal(controller.status.value, "error")
    assert.equal(controller.errorMessage.value, "network down")

    await controller.retry()

    assert.equal(controller.status.value, "success")
    assert.equal(calls.length, 2)
    assert.deepEqual(calls[1].payload, { model: "DeepSeek Flash" })
  })

  test("a stale response cannot overwrite a newer request", async () => {
    const firstDeferred = createDeferred<QueryReviewResult>()
    const secondDeferred = createDeferred<QueryReviewResult>()
    const deferreds = [firstDeferred, secondDeferred]
    const controller = createQueryReviewController(createTransport(() => deferreds.shift()!.promise))

    const firstStart = controller.start("QR-201", "Model A")
    const secondStart = controller.start("QR-201", "Model B")

    const newestResult = buildResult({ model: "Model B", mig_page_title: "Newest" })
    secondDeferred.resolve(newestResult)
    await secondStart
    assert.equal(controller.status.value, "success")
    assert.equal(controller.result.value?.mig_page_title, "Newest")

    firstDeferred.resolve(buildResult({ model: "Model A", mig_page_title: "Stale" }))
    await firstStart

    assert.equal(controller.status.value, "success")
    assert.equal(controller.result.value?.mig_page_title, "Newest")
  })

  test("cancel aborts the request and a late response is dropped", async () => {
    const deferred = createDeferred<QueryReviewResult>()
    const capturedSignals: AbortSignal[] = []
    const controller = createQueryReviewController(
      createTransport((_documentId, _payload, signal) => {
        capturedSignals.push(signal)
        return deferred.promise
      })
    )

    const startPromise = controller.start("QR-201", "DeepSeek Flash")
    controller.cancel()
    assert.equal(controller.status.value, "idle")
    assert.equal(capturedSignals[0]?.aborted, true)

    deferred.resolve(buildResult())
    await startPromise

    assert.equal(controller.status.value, "idle")
    assert.equal(controller.result.value, null)
  })

  test("resetForDocument aborts the request and clears the previous result", async () => {
    const controller = createQueryReviewController(createTransport(async () => buildResult()))
    await controller.start("QR-201", "DeepSeek Flash")
    assert.equal(controller.hasResult.value, true)

    controller.resetForDocument("QR-202")

    assert.equal(controller.status.value, "idle")
    assert.equal(controller.result.value, null)
    assert.equal(controller.hasResult.value, false)
  })

  test("apply sends the latest result repositories and exposes the applied page", async () => {
    const applyCalls: { documentId: string; repositories: QueryReviewRepository[] }[] = []
    const controller = createQueryReviewController(
      createTransport(
        async () => buildResult(),
        async (documentId, repositories) => {
          applyCalls.push({ documentId, repositories })
          return buildApplyResult()
        }
      )
    )

    await controller.start("QR-201", "DeepSeek Flash")
    await controller.apply()

    assert.equal(applyCalls.length, 1)
    assert.equal(applyCalls[0].documentId, "QR-201")
    assert.equal(applyCalls[0].repositories.length, 1)
    assert.equal(applyCalls[0].repositories[0].repository, "micro-payment")
    assert.equal(controller.isApplying.value, false)
    assert.equal(controller.applyError.value, "")
    assert.deepEqual(controller.applyResult.value, buildApplyResult())
  })

  test("a failed apply records the error, keeps the result, and clears the applied page", async () => {
    const controller = createQueryReviewController(
      createTransport(async () => buildResult(), async () => {
        throw new ApiError("The page was updated elsewhere", 502)
      })
    )

    await controller.start("QR-201", "DeepSeek Flash")
    await controller.apply()

    assert.equal(controller.applyError.value, "The page was updated elsewhere")
    assert.equal(controller.applyResult.value, null)
    assert.equal(controller.isApplying.value, false)
    assert.equal(controller.hasResult.value, true)
  })

  test("apply without a result does not call the transport", async () => {
    let applyCallCount = 0
    const controller = createQueryReviewController(
      createTransport(async () => buildResult(), async () => {
        applyCallCount += 1
        return buildApplyResult()
      })
    )

    await controller.apply()

    assert.equal(applyCallCount, 0)
    assert.equal(controller.isApplying.value, false)
    assert.equal(controller.applyError.value, "")
    assert.equal(controller.applyResult.value, null)
  })

  test("apply writes the fallback row when no repository has a query", async () => {
    let applyCallCount = 0
    const controller = createQueryReviewController(
      createTransport(
        async () => buildResult({ repositories: [buildRepository({ queries: [] })] }),
        async () => {
          applyCallCount += 1
          return buildApplyResult()
        }
      )
    )

    await controller.start("QR-201", "DeepSeek Flash")
    await controller.apply()

    assert.equal(applyCallCount, 1)
    assert.equal(controller.applyError.value, "")
    assert.deepEqual(controller.applyResult.value, buildApplyResult())
  })

  test("apply with no repository keeps the empty message and does not call the transport", async () => {
    let applyCallCount = 0
    const controller = createQueryReviewController(
      createTransport(
        async () => buildResult({ repositories: [] }),
        async () => {
          applyCallCount += 1
          return buildApplyResult()
        }
      )
    )

    await controller.start("QR-201", "DeepSeek Flash")
    await controller.apply()

    assert.equal(applyCallCount, 0)
    assert.equal(controller.applyError.value, QUERY_REVIEW_APPLY_EMPTY_MESSAGE)
    assert.equal(controller.applyResult.value, null)
  })

  test("re-running the review clears a previous apply result", async () => {
    const controller = createQueryReviewController(createTransport(async () => buildResult()))

    await controller.start("QR-201", "DeepSeek Flash")
    await controller.apply()
    assert.deepEqual(controller.applyResult.value, buildApplyResult())

    await controller.start("QR-201", "DeepSeek Flash")

    assert.equal(controller.applyResult.value, null)
    assert.equal(controller.applyError.value, "")
  })
})
