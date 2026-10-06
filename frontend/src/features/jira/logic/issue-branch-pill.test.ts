import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { IssueBranchMatch } from "../branch-types.ts"
import { issueBranchPill } from "./issue-branch-pill.ts"

const match = (source: IssueBranchMatch["source"]): IssueBranchMatch => ({
  repo: "mms-api",
  branch: "feature/CRMMS-77911-Enhancement-Manage-User",
  source,
  is_current: false,
  is_cloned: source !== "remote",
  bitbucket_project: "NME",
  last_commit: null,
  last_commit_relative: null,
  ahead: null,
  behind: null
})

describe("issue branch pill", () => {
  test("a partial result reads as a partial result", () => {
    assert.deepEqual(issueBranchPill([], { total: 3, settled: 1, failed: 1 }), {
      text: "1 of 3 created",
      tone: "warning"
    })
  })

  test("a completed plan reads as created", () => {
    assert.deepEqual(issueBranchPill([], { total: 3, settled: 3, failed: 0 }), {
      text: "3 created",
      tone: "found"
    })
  })

  test("a plan that only found existing branches is not reported as a partial failure", () => {
    assert.deepEqual(issueBranchPill([], { total: 2, settled: 2, failed: 0 }), {
      text: "2 created",
      tone: "found"
    })
  })

  test("without a plan the branch matches decide the pill", () => {
    assert.deepEqual(issueBranchPill([], null), { text: "no branch", tone: "none" })
    assert.deepEqual(issueBranchPill([match("remote")], null), { text: "not cloned", tone: "warning" })
    assert.deepEqual(issueBranchPill([match("both"), match("local")], null), {
      text: "2 branch",
      tone: "found"
    })
  })
})
