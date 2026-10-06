import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { IssueBranchMatch, WorkspaceScope } from "../branch-types.ts"
import {
  NOT_CLONED_LABEL,
  branchRows,
  emptyBranchMessage,
  remoteNoticeText,
  remoteOnlyRepositories
} from "./branch-groups.ts"

const match = (
  repo: string,
  branch: string,
  source: IssueBranchMatch["source"],
  lastCommitRelative: string | null
): IssueBranchMatch => ({
  repo,
  branch,
  source,
  is_current: false,
  is_cloned: source !== "remote",
  bitbucket_project: "NME",
  last_commit: null,
  last_commit_relative: lastCommitRelative,
  ahead: null,
  behind: null
})

const scope = (partial: Partial<WorkspaceScope>): WorkspaceScope => ({
  local_repository_count: 36,
  remote_repository_count: 65,
  coverage_percent: 55,
  verified_remotely: true,
  remote_repositories_checked: 0,
  ...partial
})

describe("branch rows", () => {
  test("rows are grouped by repository and carry the last touch", () => {
    const rows = branchRows([
      match("mms-api", "feature/CRMMS-77911-b", "both", "3 days ago"),
      match("atlas", "feature/CRMMS-77911-a", "local", "1 week ago"),
      match("mms-api", "feature/CRMMS-77911-a", "local", "yesterday")
    ])
    assert.deepEqual(rows.map((row) => row.repo), ["atlas", "mms-api", "mms-api"])
    assert.deepEqual(rows.map((row) => row.branch), [
      "feature/CRMMS-77911-a",
      "feature/CRMMS-77911-a",
      "feature/CRMMS-77911-b"
    ])
    assert.equal(rows[0].lastTouched, "1 week ago")
  })

  test("a remote only branch reads as not cloned", () => {
    const rows = branchRows([match("fe-mms", "feature/CRMMS-77911-a", "remote", null)])
    assert.equal(rows[0].lastTouched, NOT_CLONED_LABEL)
    assert.equal(rows[0].isCloned, false)
    assert.deepEqual(remoteOnlyRepositories([match("fe-mms", "feature/CRMMS-77911-a", "remote", null)]), ["fe-mms"])
  })

  test("the remote notice names every repository that is not in the workspace", () => {
    assert.equal(
      remoteNoticeText(["fe-mms"]),
      "This branch exists on Bitbucket in fe-mms, which is not in your workspace. Clone it rather than creating a second branch."
    )
    assert.match(remoteNoticeText(["fe-mms", "atlas"]), /^These branches exist on Bitbucket in fe-mms, atlas/)
    assert.equal(remoteNoticeText([]), "")
  })
})

describe("empty branch message", () => {
  test("a complete scope names the repositories that were searched", () => {
    const message = emptyBranchMessage(
      "CRMMS-77911",
      scope({ local_repository_count: 36, remote_repository_count: 36 })
    )
    assert.equal(message, "No branch found for CRMMS-77911 in the 36 repositories Lunar can see.")
  })

  test("a partial scope states the gap", () => {
    const message = emptyBranchMessage("CRMMS-77911", scope({}))
    assert.equal(
      message,
      "No branch found in the 36 of 65 repositories Lunar can see. 29 are not cloned locally."
    )
  })

  test("a remote check is reported and never implies the whole project was searched", () => {
    const message = emptyBranchMessage("CRMMS-77911", scope({ remote_repositories_checked: 5 }))
    assert.equal(
      message,
      "No branch found in the 36 repositories Lunar can see, or in the 5 most likely repositories checked on Bitbucket."
    )
  })

  test("an unreachable Bitbucket is not conclusive", () => {
    const message = emptyBranchMessage("CRMMS-77911", scope({ verified_remotely: false }))
    assert.equal(message, "No branch found locally. Bitbucket could not be reached, so this is not conclusive.")
  })
})
