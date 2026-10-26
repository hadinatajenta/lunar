import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type {
  MigTargetsResult,
  UtDocsApplyRequest,
  UtDocsApplyResult,
  UtDocsApplyScreenshot,
  UtDocsRequest,
  UtDocsResult,
  UtRunDone
} from "../types.ts"
import { UtRunStreamError } from "../api/ut-run-events.ts"
import type { UtRunEvent } from "../api/ut-run-events.ts"
import {
  UT_DOCS_APPLY_EMPTY_MESSAGE,
  UT_DOCS_MISSING_RESULT_MESSAGE,
  createUtDocsController,
  UtHelperUnreachableError,
  type UtDocsTransports,
  type UtRunStreamRequest
} from "./useUtDocs.ts"

const buildTargets = (overrides: Partial<MigTargetsResult> = {}): MigTargetsResult => ({
  mig_page_title: "CRMMS-76972-MIG",
  targets: [
    { repository: "micro-payment", branch: "feat/CRMMS-76972" },
    { repository: "micro-notification", branch: "feat/CRMMS-76972" }
  ],
  skipped_repositories: [
    { repository: "fe-mms", reason: "excluded repository" },
    { repository: "mms", reason: "excluded repository" }
  ],
  ...overrides
})

const buildDone = (repository: string, overrides: Partial<UtRunDone> = {}): UtRunDone => ({
  run_id: `run-${repository}`,
  repo: repository,
  branch: "feat/CRMMS-76972",
  base_ref: "development",
  changed_files: [`internal/${repository}/dao.go`],
  test_files: [
    {
      path: `internal/${repository}/dao_test.go`,
      package: `./internal/${repository}`,
      language: "go",
      classification: "safe",
      reason: ""
    }
  ],
  blocked_files: [
    {
      path: `internal/${repository}/reset_test.go`,
      package: `./internal/${repository}`,
      language: "go",
      classification: "blocked",
      reason: "calls OpenDB and TRUNCATE"
    }
  ],
  run: {
    status: "passed",
    executed_packages: [`./internal/${repository}`],
    total: 2,
    passed: 2,
    failed: 0,
    duration_ms: 2100,
    output_excerpt: "ok"
  },
  stashed: true,
  restored: true,
  tests: [
    {
      name: "TestCreate",
      package: `./internal/${repository}`,
      file: `internal/${repository}/dao_test.go`,
      status: "passed",
      duration_ms: 1000,
      output: "ok"
    },
    {
      name: "TestUpdate",
      package: `./internal/${repository}`,
      file: `internal/${repository}/dao_test.go`,
      status: "skipped",
      duration_ms: 0,
      output: "skipped"
    }
  ],
  untested_functions: [
    {
      name: "UntestedPayment",
      file: `internal/${repository}/payment.go`,
      package: `./internal/${repository}`
    }
  ],
  screenshot_url: `/ut-runs/run-${repository}/screenshot.png`,
  ...overrides
})

const buildDocsResult = (overrides: Partial<UtDocsResult> = {}): UtDocsResult => ({
  document_id: "UT-101",
  model: "DeepSeek Flash",
  prompt_version: "utdocs-v2",
  title: "Unit test documentation",
  summary: "Mock-only tests pass for both repositories.",
  table: "<table></table>",
  rows: [
    {
      no: 1,
      repository: "micro-payment",
      function: "CreatePayment",
      scenario: "Creates a payment",
      expected_result: "The payment is stored",
      input_data: "valid payload",
      result: "Passed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestCreate"
    },
    {
      no: 2,
      repository: "micro-notification",
      function: "SendAudit",
      scenario: "Sends an audit entry",
      expected_result: "The entry is queued",
      input_data: "audit payload",
      result: "Failed",
      screen_capture: "run-micro-notification.png",
      test_function_name: "TestAudit"
    }
  ],
  markdown: "# Unit test documentation",
  blocked_notice: "A blocked file was not run.",
  screenshot_caption: "UT run",
  mock_tests: [],
  ...overrides
})

const buildApplyResult = (overrides: Partial<UtDocsApplyResult> = {}): UtDocsApplyResult => ({
  page_id: "2144721484",
  version: 12,
  url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721484",
  ...overrides
})

const buildScreenshot = (repository: string): UtDocsApplyScreenshot => ({
  filename: `${repository}.png`,
  content_base64: "AAAA"
})

interface HarnessOptions {
  targets?: () => Promise<MigTargetsResult>
  doneForRepository?: (request: UtRunStreamRequest) => UtRunDone
  generate?: (payload: UtDocsRequest) => Promise<UtDocsResult>
  loadScreenshotUrl?: (result: UtRunDone) => Promise<string>
  loadScreenshotPayload?: (result: UtRunDone) => Promise<UtDocsApplyScreenshot>
  apply?: (payload: UtDocsApplyRequest) => Promise<UtDocsApplyResult>
}

const createHarness = (options: HarnessOptions = {}) => {
  const streamRequests: UtRunStreamRequest[] = []
  const streamSignals: AbortSignal[] = []
  const generateCalls: { documentId: string; payload: UtDocsRequest }[] = []
  const releasedUrls: string[] = []
  const payloadCalls: UtRunDone[] = []
  const applyCalls: { documentId: string; payload: UtDocsApplyRequest }[] = []
  const handlers: ((event: UtRunEvent) => void)[] = []
  const resolvers: (() => void)[] = []
  const rejecters: ((error: unknown) => void)[] = []
  let streamWaiters: (() => void)[] = []

  const transports: UtDocsTransports = {
    loadTargets: () => (options.targets ? options.targets() : Promise.resolve(buildTargets())),
    streamRun: (request, signal, onEvent) => {
      streamRequests.push(request)
      streamSignals.push(signal)
      handlers.push(onEvent)
      const waiters = streamWaiters
      streamWaiters = []
      for (const waiter of waiters) {
        waiter()
      }
      return new Promise<void>((resolve, reject) => {
        resolvers.push(resolve)
        rejecters.push(reject)
      })
    },
    generateDocument: (documentId, payload) => {
      generateCalls.push({ documentId, payload })
      return options.generate ? options.generate(payload) : Promise.resolve(buildDocsResult())
    },
    loadScreenshotUrl: (result) =>
      options.loadScreenshotUrl
        ? options.loadScreenshotUrl(result)
        : Promise.resolve(`blob:${result.run_id}`),
    releaseScreenshotUrl: (objectUrl) => {
      releasedUrls.push(objectUrl)
    },
    loadScreenshotPayload: (result) => {
      payloadCalls.push(result)
      return options.loadScreenshotPayload
        ? options.loadScreenshotPayload(result)
        : Promise.resolve(buildScreenshot(result.repo))
    },
    apply: (documentId, payload) => {
      applyCalls.push({ documentId, payload })
      return options.apply ? options.apply(payload) : Promise.resolve(buildApplyResult())
    }
  }

  const controller = createUtDocsController(transports)

  const flush = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))
  const waitForStreams = async (count: number): Promise<void> => {
    while (streamRequests.length < count) {
      if (streamWaiters.length === 0) {
        await flush()
      } else {
        await new Promise<void>((resolve) => streamWaiters.push(resolve))
      }
    }
  }

  return {
    controller,
    streamRequests,
    streamSignals,
    generateCalls,
    releasedUrls,
    payloadCalls,
    applyCalls,
    emit: (event: UtRunEvent) => handlers[handlers.length - 1](event),
    finishStream: (index: number) => resolvers[index](),
    failStream: (index: number, error: unknown) => rejecters[index](error),
    waitForStreams,
    flush
  }
}

describe("useUtDocs MIG-driven state machine", () => {
  test("starts idle with no targets and no document", () => {
    const harness = createHarness()

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.controller.generatedDocument.value, null)
    assert.equal(harness.controller.hasResult.value, false)
    assert.deepEqual(harness.controller.targets.value, [])
  })

  test("loads the MIG targets, runs every repository, and generates one aggregate document", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "loading-targets")
    await harness.waitForStreams(1)

    assert.equal(harness.controller.status.value, "running")
    assert.equal(harness.controller.migPageTitle.value, "CRMMS-76972-MIG")
    assert.equal(harness.controller.targets.value.length, 2)
    assert.equal(harness.controller.skippedRepositories.value[0].repository, "fe-mms")
    assert.deepEqual(harness.streamRequests[0], {
      repo: "micro-payment",
      branch: "feat/CRMMS-76972"
    })

    harness.emit({ kind: "phase", phase: "scan" })
    assert.equal(harness.controller.phase.value, "scan")
    harness.emit({ kind: "output", text: "=== RUN TestCreate\n" })
    assert.equal(harness.controller.outputText.value, "=== RUN TestCreate\n")
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)

    await harness.waitForStreams(2)
    assert.equal(harness.controller.currentRepository.value, "micro-notification")
    assert.deepEqual(harness.streamRequests[1], {
      repo: "micro-notification",
      branch: "feat/CRMMS-76972"
    })

    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    assert.equal(harness.controller.status.value, "success")
    assert.equal(harness.controller.runResults.value.length, 2)
    assert.equal(harness.generateCalls.length, 1)
    assert.equal(harness.generateCalls[0].documentId, "UT-101")
    assert.deepEqual(harness.controller.generatedDocument.value, buildDocsResult())

    const payload = harness.generateCalls[0].payload
    assert.equal(payload.model, "DeepSeek Flash")
    assert.equal(payload.repositories.length, 2)
    assert.equal(payload.repositories[0].repository, "micro-payment")
    assert.equal(payload.repositories[0].tests.length, 2)
    assert.deepEqual(payload.repositories[0].untested_functions, [
      {
        name: "UntestedPayment",
        file: "internal/micro-payment/payment.go",
        package: "./internal/micro-payment"
      }
    ])
    assert.equal(payload.repositories[1].repository, "micro-notification")
    assert.equal(payload.repositories[1].blocked_files[0].classification, "blocked")
    assert.equal(payload.repositories[1].untested_functions[0].name, "UntestedPayment")
  })

  test("forwards the target base ref to the helper when the backend returns one", async () => {
    const harness = createHarness({
      targets: async () =>
        buildTargets({ targets: [{ repository: "micro-payment", branch: "feat/x", base_ref: "main" }] })
    })
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)

    assert.deepEqual(harness.streamRequests[0], {
      repo: "micro-payment",
      branch: "feat/x",
      base_ref: "main"
    })

    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await startPromise

    assert.equal(harness.controller.status.value, "success")
  })

  test("shows the empty state when the MIG page has no target", async () => {
    const harness = createHarness({
      targets: async () => buildTargets({ targets: [], skipped_repositories: [] })
    })

    await harness.controller.start("UT-101", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "empty")
    assert.equal(harness.streamRequests.length, 0)
    assert.equal(harness.controller.generatedDocument.value, null)
  })

  test("names the helper and keeps the setup link reachable when the helper is unreachable", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)

    harness.failStream(
      0,
      new UtHelperUnreachableError("The Lunar helper is not reachable.", "http://127.0.0.1:5199")
    )
    await startPromise

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.isHelperUnreachable.value, true)
    assert.equal(harness.controller.helperUrl.value, "http://127.0.0.1:5199")
  })

  test("a helper error event on the second repository reports the message", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)

    harness.failStream(1, new UtRunStreamError("The helper could not check out the branch"))
    await startPromise

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.errorMessage.value, "The helper could not check out the branch")
    assert.equal(harness.controller.runResults.value.length, 1)
  })

  test("a stream that ends without a result fails the run", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.finishStream(0)
    await startPromise

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.errorMessage.value, UT_DOCS_MISSING_RESULT_MESSAGE)
  })

  test("a screenshot failure does not fail the run", async () => {
    const harness = createHarness({
      loadScreenshotUrl: async () => {
        throw new Error("screenshot unavailable")
      }
    })
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)
    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    assert.equal(harness.controller.status.value, "success")
    assert.equal(harness.controller.screenshotError.value.length > 0, true)
  })

  test("apply sends each repository screenshot and its rows and exposes the result", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)
    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    await harness.controller.apply()

    assert.equal(harness.controller.isApplying.value, false)
    assert.equal(harness.controller.applyError.value, "")
    assert.deepEqual(harness.controller.applyResult.value, buildApplyResult())
    assert.equal(harness.applyCalls.length, 1)
    assert.equal(harness.applyCalls[0].documentId, "UT-101")

    const repositories = harness.applyCalls[0].payload.repositories
    assert.equal(repositories.length, 2)
    assert.deepEqual(repositories[0].screenshot, { filename: "micro-payment.png", content_base64: "AAAA" })
    assert.deepEqual(repositories[0].rows, [
      {
        function: "CreatePayment",
        scenario: "Creates a payment",
        expected_result: "The payment is stored",
        input_data: "valid payload",
        result: "Passed"
      }
    ])
    assert.equal(repositories[1].rows[0].function, "SendAudit")
  })

  test("apply forwards each repository mock test code and drops the helper-internal name", async () => {
    const notRunRow = (repository: string, testFunctionName: string) => ({
      no: 1,
      repository,
      function: "CreatePayment",
      scenario: "Creates a payment",
      expected_result: "The payment is stored",
      input_data: "valid payload",
      result: "Not Run",
      screen_capture: "",
      test_function_name: testFunctionName
    })

    const harness = createHarness({
      generate: async () =>
        buildDocsResult({
          rows: [
            notRunRow("micro-payment", "TestCreatePaymentMock"),
            notRunRow("micro-notification", "TestSendAuditMock")
          ],
          mock_tests: [
            {
              test_function_name: "TestCreatePaymentMock",
              function: "CreatePayment",
              scenario: "Creates a payment",
              expected_result: "The payment is stored",
              input_data: "valid payload",
              code: "func TestCreatePaymentMock(t *testing.T) {\n\tsqlmock.New()\n}"
            },
            {
              test_function_name: "TestSendAuditMock",
              function: "SendAudit",
              scenario: "Sends an audit entry",
              expected_result: "The entry is queued",
              input_data: "audit payload",
              code: "func TestSendAuditMock(t *testing.T) {\n\tmockQueue()\n}"
            }
          ]
        })
    })
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)
    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    await harness.controller.apply()

    const repositories = harness.applyCalls[0].payload.repositories
    assert.equal(repositories.length, 2)
    assert.deepEqual(repositories[0].mock_tests, [
      {
        function: "CreatePayment",
        scenario: "Creates a payment",
        expected_result: "The payment is stored",
        input_data: "valid payload",
        code: "func TestCreatePaymentMock(t *testing.T) {\n\tsqlmock.New()\n}"
      }
    ])
    assert.deepEqual(repositories[1].mock_tests, [
      {
        function: "SendAudit",
        scenario: "Sends an audit entry",
        expected_result: "The entry is queued",
        input_data: "audit payload",
        code: "func TestSendAuditMock(t *testing.T) {\n\tmockQueue()\n}"
      }
    ])
  })

  test("a failed apply keeps the error and retries through the same action", async () => {
    let applyAttempt = 0
    const harness = createHarness({
      apply: async () => {
        applyAttempt += 1
        if (applyAttempt === 1) {
          throw new Error("The page was updated elsewhere.")
        }
        return buildApplyResult()
      }
    })
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)
    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    await harness.controller.apply()
    assert.equal(harness.controller.applyError.value, "The page was updated elsewhere.")
    assert.equal(harness.controller.applyResult.value, null)

    await harness.controller.apply()
    assert.equal(harness.controller.applyError.value, "")
    assert.deepEqual(harness.controller.applyResult.value, buildApplyResult())
    assert.equal(harness.applyCalls.length, 2)
  })

  test("apply with no run result reports the empty apply message", async () => {
    const harness = createHarness({
      targets: async () => buildTargets({ targets: [], skipped_repositories: [] })
    })

    await harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.controller.apply()

    assert.equal(harness.controller.applyError.value, UT_DOCS_APPLY_EMPTY_MESSAGE)
    assert.equal(harness.applyCalls.length, 0)
  })

  test("cancel aborts the helper request and a late result is dropped", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)

    harness.controller.cancel()

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.streamSignals[0].aborted, true)

    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await startPromise

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.controller.runResults.value.length, 0)
    assert.equal(harness.controller.generatedDocument.value, null)
  })

  test("resetting for another document releases the screenshot object URLs and clears state", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("UT-101", "DeepSeek Flash")
    await harness.waitForStreams(1)
    harness.emit({ kind: "done", result: buildDone("micro-payment") })
    harness.finishStream(0)
    await harness.waitForStreams(2)
    harness.emit({ kind: "done", result: buildDone("micro-notification") })
    harness.finishStream(1)
    await startPromise

    assert.equal(harness.controller.screenshotUrls.value["micro-payment"], "blob:run-micro-payment")

    harness.controller.resetForDocument("UT-102")

    assert.equal(harness.controller.status.value, "idle")
    assert.deepEqual(harness.controller.screenshotUrls.value, {})
    assert.equal(harness.controller.generatedDocument.value, null)
    assert.deepEqual(harness.releasedUrls, ["blob:run-micro-payment", "blob:run-micro-notification"])
  })
})
