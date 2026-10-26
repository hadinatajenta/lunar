import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { UtRunDone, UtRunPhase } from "../types.ts"
import {
  UT_RUN_UNREADABLE_RESULT_MESSAGE,
  UtRunStreamError,
  createUtRunStreamState,
  dispatchUtRunFrame,
  drainUtRunFrames,
  parseUtRunDone
} from "./ut-run-events.ts"
import type { UtRunEvent } from "./ut-run-events.ts"

const buildDonePayload = (): Record<string, unknown> => ({
  run_id: "run-42",
  repo: "micro-payment",
  branch: "feat/CRMMS-123",
  base_ref: "development",
  changed_files: ["internal/repo/dao.go", 7],
  test_files: [
    {
      path: "internal/repo/dao_test.go",
      package: "./internal/repo",
      language: "go",
      classification: "safe",
      reason: ""
    }
  ],
  blocked_files: [
    {
      path: "internal/seed/reset_test.go",
      package: "./internal/seed",
      language: "go",
      classification: "blocked",
      reason: "calls OpenDB and TRUNCATE"
    }
  ],
  run: {
    status: "failed",
    executed_packages: ["./internal/repo"],
    total: 3,
    passed: 2,
    failed: 1,
    duration_ms: 1200,
    output_excerpt: "--- FAIL: TestDao"
  },
  stashed: true,
  restored: true,
  tests: [
    {
      name: "TestDao",
      package: "./internal/repo",
      file: "internal/repo/dao_test.go",
      status: "failed",
      duration_ms: 800,
      output: "--- FAIL: TestDao"
    },
    {
      name: "TestList",
      package: "./internal/repo",
      file: "internal/repo/dao_test.go",
      status: "skipped",
      duration_ms: 0,
      output: ""
    },
    { name: "", package: "./internal/repo", status: "passed" },
    { name: "TestBogus", status: "exploded" }
  ],
  untested_functions: [
    { name: "CreatePayment", file: "internal/repo/payment.go", package: "./internal/repo" },
    { name: "RefundPayment", file: "internal/repo/refund.go", package: "./internal/repo" }
  ],
  screenshot_url: "/ut-runs/run-42/screenshot.png"
})

const createHandlers = () => {
  const phases: UtRunPhase[] = []
  const output: string[] = []
  const done: UtRunDone[] = []
  return {
    phases,
    output,
    done,
    handlers: {
      onPhase: (phase: UtRunPhase) => phases.push(phase),
      onOutput: (text: string) => output.push(text),
      onDone: (result: UtRunDone) => done.push(result)
    }
  }
}

describe("ut run stream frames", () => {
  test("an explicit event name parses a phase frame", () => {
    const harness = createHandlers()

    const isDone = dispatchUtRunFrame('event: phase\ndata: {"phase":"safety"}', harness.handlers)

    assert.equal(isDone, false)
    assert.deepEqual(harness.phases, ["safety"])
  })

  test("a type field in the payload selects the event when the frame has no event name", () => {
    const harness = createHandlers()

    dispatchUtRunFrame('data: {"type":"output","text":"PASS"}', harness.handlers)

    assert.deepEqual(harness.output, ["PASS"])
  })

  test("a raw data line is treated as output text", () => {
    const harness = createHandlers()

    dispatchUtRunFrame("event: output\ndata: ok internal/repo", harness.handlers)

    assert.deepEqual(harness.output, ["ok internal/repo"])
  })

  test("an error frame throws with the reported message", () => {
    const harness = createHandlers()

    assert.throws(
      () => dispatchUtRunFrame('event: error\ndata: {"message":"the repository is not cloned"}', harness.handlers),
      (error: unknown) =>
        error instanceof UtRunStreamError && error.message === "the repository is not cloned"
    )
  })

  test("a done frame parses and normalizes the run result", () => {
    const harness = createHandlers()
    const payload = buildDonePayload()

    const isDone = dispatchUtRunFrame(`event: done\ndata: ${JSON.stringify(payload)}`, harness.handlers)

    assert.equal(isDone, true)
    assert.equal(harness.done.length, 1)
    assert.deepEqual(harness.done[0].changed_files, ["internal/repo/dao.go"])
    assert.equal(harness.done[0].run.status, "failed")
    assert.equal(harness.done[0].blocked_files[0].classification, "blocked")
    assert.equal(harness.done[0].stashed, true)
    assert.equal(harness.done[0].restored, true)
    assert.equal(harness.done[0].tests.length, 3)
    assert.deepEqual(harness.done[0].tests[0], {
      name: "TestDao",
      package: "./internal/repo",
      file: "internal/repo/dao_test.go",
      status: "failed",
      duration_ms: 800,
      output: "--- FAIL: TestDao"
    })
    assert.equal(harness.done[0].tests[1].status, "skipped")
    assert.equal(harness.done[0].tests[2].name, "TestBogus")
    assert.equal(harness.done[0].tests[2].status, "skipped")
    assert.deepEqual(harness.done[0].untested_functions, [
      { name: "CreatePayment", file: "internal/repo/payment.go", package: "./internal/repo" },
      { name: "RefundPayment", file: "internal/repo/refund.go", package: "./internal/repo" }
    ])
  })

  test("a done payload drops malformed untested functions instead of rendering them", () => {
    const parsed = parseUtRunDone(
      JSON.stringify({
        run_id: "run-9",
        run: { status: "passed", total: 1, passed: 1, failed: 0 },
        untested_functions: [
          { name: "CreatePayment", file: "internal/repo/payment.go", package: "./internal/repo" },
          { name: "", file: "internal/repo/blank.go", package: "./internal/repo" },
          null,
          "not an object",
          { name: "RefundPayment", file: 7, package: "./internal/repo" }
        ],
        screenshot_url: "/ut-runs/run-9/screenshot.png"
      })
    )

    assert.deepEqual(parsed.untested_functions, [
      { name: "CreatePayment", file: "internal/repo/payment.go", package: "./internal/repo" },
      { name: "RefundPayment", file: "", package: "./internal/repo" }
    ])
  })

  test("a done payload without the strict-run fields defaults them safely", () => {
    const withoutStrictFields = parseUtRunDone(
      JSON.stringify({
        run_id: "run-7",
        run: { status: "passed", total: 1, passed: 1, failed: 0 },
        screenshot_url: "/ut-runs/run-7/screenshot.png"
      })
    )

    assert.equal(withoutStrictFields.stashed, false)
    assert.equal(withoutStrictFields.restored, false)
    assert.deepEqual(withoutStrictFields.tests, [])
    assert.deepEqual(withoutStrictFields.untested_functions, [])
  })

  test("a done payload without a run result is rejected rather than rendered", () => {
    assert.throws(
      () => parseUtRunDone('{"run_id":"run-1"}'),
      (error: unknown) =>
        error instanceof UtRunStreamError && error.message === UT_RUN_UNREADABLE_RESULT_MESSAGE
    )
  })

  test("the [DONE] marker ends the stream without an event", () => {
    const harness = createHandlers()

    assert.equal(dispatchUtRunFrame("data: [DONE]", harness.handlers), true)
    assert.deepEqual(harness.done, [])
  })

  test("draining a buffer dispatches every complete frame and keeps the partial tail", () => {
    const harness = createHandlers()
    const state = createUtRunStreamState()
    state.buffer =
      'event: phase\ndata: {"phase":"scan"}\n\n' +
      'event: output\ndata: {"text":"x"}\n\n' +
      'event: phase\ndata: {"phase":"run"}'

    drainUtRunFrames(state, harness.handlers)

    assert.deepEqual(harness.phases, ["scan"])
    assert.deepEqual(harness.output, ["x"])
    assert.equal(state.buffer, 'event: phase\ndata: {"phase":"run"}')

    state.buffer += "\n\n"
    drainUtRunFrames(state, harness.handlers)
    assert.deepEqual(harness.phases, ["scan", "run"])
    assert.equal(state.buffer, "")
  })

  test("an unknown event name is ignored", () => {
    const harness = createHandlers()
    const events: UtRunEvent[] = []

    const isDone = dispatchUtRunFrame('event: heartbeat\ndata: {"at":1}', {
      ...harness.handlers,
      onOutput: (text) => events.push({ kind: "output", text })
    })

    assert.equal(isDone, false)
    assert.deepEqual(events, [])
  })
})
