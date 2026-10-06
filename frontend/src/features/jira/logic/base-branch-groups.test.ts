import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { BaseBranchOption } from "../branch-types.ts"
import {
  OTHER_TEAMS_GROUP_LABEL,
  baseBranchGroups,
  emptyBaseBranchMessage
} from "./base-branch-groups.ts"

const option = (
  name: string,
  source: BaseBranchOption["source"],
  lastCommitRelative: string | null
): BaseBranchOption => ({ name, source, last_commit_relative: lastCommitRelative })

const options: BaseBranchOption[] = [
  option("release/production", "bitbucket_default", "23 Sep 2026"),
  option("pre/development", "trusted", "2 Oct 2026"),
  option("feature/CRMMS-77745-Input-MDR-QRIS-Statis", "other", "1 Oct 2026")
]

describe("base branch grouping", () => {
  test("production like branches come first and another team's branches are grouped", () => {
    const groups = baseBranchGroups(options, "")
    assert.equal(groups.length, 2)
    assert.equal(groups[0].label, "")
    assert.deepEqual(groups[0].options.map((entry) => entry.name), ["release/production", "pre/development"])
    assert.equal(groups[1].label, OTHER_TEAMS_GROUP_LABEL)
    assert.deepEqual(groups[1].options.map((entry) => entry.name), ["feature/CRMMS-77745-Input-MDR-QRIS-Statis"])
  })

  test("another team's branch is selectable, never filtered out", () => {
    const groups = baseBranchGroups(options, "crmms-77745")
    assert.equal(groups.length, 1)
    assert.equal(groups[0].label, OTHER_TEAMS_GROUP_LABEL)
    assert.equal(groups[0].options.length, 1)
  })

  test("the search is case insensitive and keeps the last commit visible", () => {
    const groups = baseBranchGroups(options, "RELEASE")
    assert.deepEqual(groups[0].options, [option("release/production", "bitbucket_default", "23 Sep 2026")])
  })

  test("a search that matches nothing reports the term", () => {
    const groups = baseBranchGroups(options, "nope")
    assert.deepEqual(groups, [])
    assert.equal(emptyBaseBranchMessage("aurora", "nope", options.length), 'No branch in aurora matches "nope"')
  })

  test("a repository with no local branch says so instead of echoing an empty term", () => {
    assert.equal(emptyBaseBranchMessage("fe-mms", "", 0), "No branch is available in fe-mms.")
  })
})
