import assert from "node:assert/strict"
import { describe, test } from "node:test"
import { ApiError } from "../../../lib/http.ts"
import type {
  SopApplyRequest,
  SopApplyResult,
  SopGenerateRequest,
  SopGenerateResult,
  SopScanResult,
  SopScanTarget
} from "../types.ts"
import {
  SOP_DOCS_APPLY_FAILURE_MESSAGE,
  SopHelperUnreachableError,
  buildSopScanRequest,
  createSopDocsController,
  type SopDocsTransports
} from "./useSopDocs.ts"

const buildScanResult = (
  repository: string,
  overrides: Partial<SopScanResult> = {}
): SopScanResult => ({
  repository,
  branch: "feature/CRMMS-77914",
  base_ref: "master",
  database_files: [
    {
      path: "migrations/2026_add_mdr.sql",
      statements: [{ action: "CREATE", table: "log_program_mdr", sql: "CREATE TABLE log_program_mdr" }]
    }
  ],
  tables: [{ name: "log_program_mdr", action: "CREATE" }],
  warnings: [],
  ...overrides
})

const buildGenerateResult = (
  overrides: Partial<SopGenerateResult> = {}
): SopGenerateResult => ({
  issue_key: "CRMMS-78064",
  development_page_id: "2144721483",
  repositories: [
    {
      repository: "everest",
      branch: "feature/CRMMS-77914",
      base_ref: "master",
      database_files: [],
      warnings: []
    }
  ],
  deploy: {
    title: "CRMMS-78064-SOP Deploy",
    page_id: "2144721548",
    exists: true,
    managed_block: "<p><strong>BAB I. DEPLOY DATABASE MMS</strong></p>"
  },
  operasional: {
    title: "CRMMS-78064-SOP Operasional",
    page_id: "2144721557",
    exists: true,
    children: [
      { title: "Maintenance", exists: true },
      { title: "Monitoring", exists: false },
      { title: "Troubleshooting", exists: false }
    ]
  },
  warnings: ["no Bamboo plan configured for everest"],
  ...overrides
})

const buildApplyResult = (overrides: Partial<SopApplyResult> = {}): SopApplyResult => ({
  deploy_page: {
    id: "2144721548",
    version: 14,
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721548",
    action: "updated"
  },
  operasional_page: {
    id: "2144721557",
    version: 3,
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721557"
  },
  created_children: ["Monitoring", "Troubleshooting"],
  ...overrides
})

interface HarnessOptions {
  generate?: (payload: SopGenerateRequest, callIndex: number) => Promise<SopGenerateResult>
  scan?: (target: SopScanTarget) => Promise<SopScanResult>
  apply?: (payload: SopApplyRequest) => Promise<SopApplyResult>
}

const createHarness = (options: HarnessOptions = {}) => {
  const generateCalls: { documentId: string; payload: SopGenerateRequest }[] = []
  const scanCalls: SopScanTarget[] = []
  const applyCalls: { documentId: string; payload: SopApplyRequest }[] = []
  const signals: AbortSignal[] = []

  const transports: SopDocsTransports = {
    generate: (documentId, payload, signal) => {
      const callIndex = generateCalls.length
      generateCalls.push({ documentId, payload })
      signals.push(signal)
      return options.generate
        ? options.generate(payload, callIndex)
        : Promise.resolve(buildGenerateResult())
    },
    scan: (target, signal) => {
      scanCalls.push(target)
      signals.push(signal)
      return options.scan ? options.scan(target) : Promise.resolve(buildScanResult(target.repository))
    },
    apply: (documentId, payload) => {
      applyCalls.push({ documentId, payload })
      return options.apply ? options.apply(payload) : Promise.resolve(buildApplyResult())
    }
  }

  return {
    controller: createSopDocsController(transports),
    generateCalls,
    scanCalls,
    applyCalls,
    signals
  }
}

describe("useSopDocs state machine", () => {
  test("starts idle with no result and no scans", () => {
    const harness = createHarness()

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.controller.result.value, null)
    assert.deepEqual(harness.controller.scans.value, {})
    assert.equal(harness.controller.hasResult.value, false)
    assert.equal(harness.controller.isLoading.value, false)
  })

  test("a generate success scans every repository and exposes the plan", async () => {
    const harness = createHarness()
    const startPromise = harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "loading")
    assert.equal(harness.controller.isLoading.value, true)
    await startPromise

    assert.equal(harness.controller.status.value, "success")
    assert.equal(harness.controller.hasResult.value, true)
    assert.equal(harness.generateCalls.length, 2)
    assert.equal(harness.generateCalls[0].documentId, "2144721483")
    assert.deepEqual(harness.generateCalls[0].payload, { model: "DeepSeek Flash" })
    assert.deepEqual(harness.generateCalls[1].payload, {
      model: "DeepSeek Flash",
      repositories: [buildScanResult("everest")]
    })

    assert.equal(harness.scanCalls.length, 1)
    assert.deepEqual(harness.scanCalls[0], {
      repository: "everest",
      branch: "feature/CRMMS-77914",
      base_ref: "master"
    })
    assert.equal(harness.controller.scans.value["everest"].tables[0].name, "log_program_mdr")
    assert.equal(harness.controller.result.value?.issue_key, "CRMMS-78064")
  })

  test("the second generate receives the scanned repositories and drives the preview", async () => {
    const scanned = buildScanResult("everest", {
      tables: [{ name: "log_program_mdr", action: "CREATE" }]
    })
    const harness = createHarness({
      generate: async (_payload, callIndex) =>
        callIndex === 0
          ? buildGenerateResult({
              deploy: {
                title: "CRMMS-78064-SOP Deploy",
                page_id: "2144721548",
                exists: true,
                managed_block: "<p>Tidak ada penambahan Table/Database</p>"
              }
            })
          : buildGenerateResult({
              deploy: {
                title: "CRMMS-78064-SOP Deploy",
                page_id: "2144721548",
                exists: true,
                managed_block: "<p>log_program_mdr</p>"
              }
            }),
      scan: async () => scanned
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "success")
    assert.deepEqual(harness.generateCalls[1].payload, {
      model: "DeepSeek Flash",
      repositories: [scanned]
    })
    assert.equal(harness.controller.result.value?.deploy.managed_block, "<p>log_program_mdr</p>")
  })

  test("a second generate failure moves to the error state with retry", async () => {
    const harness = createHarness({
      generate: async (_payload, callIndex) => {
        if (callIndex === 1) {
          throw new ApiError("The SOP plan could not be built", 502)
        }
        return buildGenerateResult()
      }
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.errorMessage.value, "The SOP plan could not be built")
    assert.equal(harness.controller.hasResult.value, false)
    assert.deepEqual(harness.controller.scans.value, {})

    await harness.controller.retry()
    assert.equal(harness.controller.status.value, "success")
  })

  test("a multi-repository plan scans each repository in response order", async () => {
    const harness = createHarness({
      generate: async () =>
        buildGenerateResult({
          repositories: [
            { repository: "everest", branch: "feature/a", database_files: [], warnings: [] },
            { repository: "fe-mms", branch: "feature/a", base_ref: "master", database_files: [], warnings: [] }
          ]
        })
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "success")
    assert.deepEqual(harness.scanCalls, [
      { repository: "everest", branch: "feature/a" },
      { repository: "fe-mms", branch: "feature/a", base_ref: "master" }
    ])
    assert.deepEqual(Object.keys(harness.controller.scans.value), ["everest", "fe-mms"])
  })

  test("exposes the generate warnings on the result", async () => {
    const harness = createHarness({
      generate: async () => buildGenerateResult({ warnings: ["no Bamboo plan configured for everest"] })
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.deepEqual(harness.controller.result.value?.warnings, [
      "no Bamboo plan configured for everest"
    ])
  })

  test("a scan failure moves to the error state with the backend message", async () => {
    const harness = createHarness({
      scan: async () => {
        throw new ApiError("The helper could not check out the branch", 422)
      }
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.errorMessage.value, "The helper could not check out the branch")
    assert.equal(harness.controller.hasResult.value, false)
    assert.deepEqual(harness.controller.scans.value, {})
  })

  test("an unreachable helper is named and keeps the setup path", async () => {
    const harness = createHarness({
      scan: async () => {
        throw new SopHelperUnreachableError("The Lunar helper is not reachable.", "http://127.0.0.1:5199")
      }
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")

    assert.equal(harness.controller.status.value, "error")
    assert.equal(harness.controller.isHelperUnreachable.value, true)
    assert.equal(harness.controller.helperUrl.value, "http://127.0.0.1:5199")
  })

  test("retry repeats the last generate after a failure", async () => {
    let attempt = 0
    const harness = createHarness({
      scan: async (target) => {
        attempt += 1
        if (attempt === 1) {
          throw new Error("network down")
        }
        return buildScanResult(target.repository)
      }
    })

    await harness.controller.start("2144721483", "DeepSeek Flash")
    assert.equal(harness.controller.status.value, "error")

    await harness.controller.retry()

    assert.equal(harness.controller.status.value, "success")
    assert.equal(harness.generateCalls.length, 3)
    assert.equal(harness.scanCalls.length, 2)
  })

  test("cancel aborts the request and a late generate is dropped", async () => {
    let resolveGenerate!: (value: SopGenerateResult) => void
    const harness = createHarness({
      generate: () =>
        new Promise<SopGenerateResult>((resolve) => {
          resolveGenerate = resolve
        })
    })

    const startPromise = harness.controller.start("2144721483", "DeepSeek Flash")
    harness.controller.cancel()

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.signals[0]?.aborted, true)

    resolveGenerate(buildGenerateResult())
    await startPromise

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.controller.result.value, null)
  })

  test("apply uses update mode when the deploy page exists and reports the applied version", async () => {
    const harness = createHarness()
    await harness.controller.start("2144721483", "DeepSeek Flash")

    await harness.controller.apply()

    assert.equal(harness.controller.isApplying.value, false)
    assert.equal(harness.controller.applyError.value, "")
    assert.deepEqual(harness.controller.applyResult.value, buildApplyResult())
    assert.equal(harness.applyCalls.length, 1)
    assert.deepEqual(harness.applyCalls[0].payload, {
      deploy: { mode: "update", managed_block: buildGenerateResult().deploy.managed_block },
      operasional: { create_children: true }
    })
  })

  test("apply uses create mode when the deploy page does not exist", async () => {
    const harness = createHarness({
      generate: async () =>
        buildGenerateResult({
          deploy: { title: "CRMMS-78064-SOP Deploy", page_id: "", exists: false, managed_block: "<p>BAB I</p>" }
        })
    })
    await harness.controller.start("2144721483", "DeepSeek Flash")

    await harness.controller.apply()

    assert.deepEqual(harness.applyCalls[0].payload, {
      deploy: { mode: "create", managed_block: "<p>BAB I</p>" },
      operasional: { create_children: true }
    })
  })

  test("a failed apply records the error and retry succeeds", async () => {
    let attempt = 0
    const harness = createHarness({
      apply: async () => {
        attempt += 1
        if (attempt === 1) {
          throw new ApiError("The page was updated elsewhere", 502)
        }
        return buildApplyResult()
      }
    })
    await harness.controller.start("2144721483", "DeepSeek Flash")

    await harness.controller.apply()
    assert.equal(harness.controller.applyError.value, "The page was updated elsewhere")
    assert.equal(harness.controller.applyResult.value, null)

    await harness.controller.apply()
    assert.equal(harness.controller.applyError.value, "")
    assert.deepEqual(harness.controller.applyResult.value, buildApplyResult())
    assert.equal(harness.applyCalls.length, 2)
  })

  test("apply before a result falls back to the apply failure message without a call", async () => {
    let applyCallCount = 0
    const harness = createHarness({
      apply: async () => {
        applyCallCount += 1
        return buildApplyResult()
      }
    })

    await harness.controller.apply()

    assert.equal(applyCallCount, 0)
    assert.equal(harness.controller.applyError.value, "")
    assert.equal(SOP_DOCS_APPLY_FAILURE_MESSAGE.length > 0, true)
  })

  test("resetForDocument clears the result, the scans, and the apply state", async () => {
    const harness = createHarness()
    await harness.controller.start("2144721483", "DeepSeek Flash")
    await harness.controller.apply()
    assert.equal(harness.controller.hasResult.value, true)

    harness.controller.resetForDocument("2144729999")

    assert.equal(harness.controller.status.value, "idle")
    assert.equal(harness.controller.result.value, null)
    assert.deepEqual(harness.controller.scans.value, {})
    assert.equal(harness.controller.applyResult.value, null)
  })
})

describe("buildSopScanRequest", () => {
  test("builds the helper request with root, repo, branch, and base_ref", () => {
    assert.deepEqual(
      buildSopScanRequest(
        { repository: "everest", branch: "feature/CRMMS-77914", base_ref: "master" },
        "/Users/erendt/BRI"
      ),
      {
        root: "/Users/erendt/BRI",
        repo: "everest",
        branch: "feature/CRMMS-77914",
        base_ref: "master"
      }
    )
  })

  test("omits an empty base_ref", () => {
    assert.deepEqual(
      buildSopScanRequest({ repository: "fe-mms", branch: "feature/CRMMS-77914" }, "/Users/erendt/BRI"),
      { root: "/Users/erendt/BRI", repo: "fe-mms", branch: "feature/CRMMS-77914" }
    )
  })
})
