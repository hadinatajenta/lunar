import assert from "node:assert/strict"
import { describe, test } from "node:test"
import { hasIssueKey, readIssueKey } from "./issue-key.ts"

describe("readIssueKey", () => {
  test("reads the issue key from a wiki title", () => {
    assert.equal(readIssueKey("CRMMS-78064-[2026MMDD] Dokumen Query Review"), "CRMMS-78064")
    assert.equal(readIssueKey("CRMMS-78064-SOP Deploy"), "CRMMS-78064")
  })

  test("returns null when the title carries no issue key", () => {
    assert.equal(readIssueKey("Architecture Overview"), null)
    assert.equal(readIssueKey(""), null)
    assert.equal(readIssueKey("Deploy notes for release"), null)
  })

  test("hasIssueKey mirrors readIssueKey", () => {
    assert.equal(hasIssueKey("CRMMS-78064 SOP Operasional"), true)
    assert.equal(hasIssueKey("Deploy notes for release"), false)
  })
})
