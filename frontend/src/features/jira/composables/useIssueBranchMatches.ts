import { computed, ref } from "vue"
import type { IssueBranchMatch, WorkspaceScope } from "../branch-types"
import { fetchIssueBranches } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const ISSUE_BRANCH_CACHE_TTL_MS = 30_000

const matchesByIssue = ref<Record<string, IssueBranchMatch[]>>({})
const scopeByIssue = ref<Record<string, WorkspaceScope | null>>({})
const errorByIssue = ref<Record<string, string>>({})
const isLoadingByIssue = ref<Record<string, boolean>>({})
const loadedAtByIssue: Record<string, number> = {}

export function useIssueBranchMatches() {
  const matchesFor = (issueKey: string): IssueBranchMatch[] => matchesByIssue.value[issueKey] ?? []
  const scopeFor = (issueKey: string): WorkspaceScope | null => scopeByIssue.value[issueKey] ?? null
  const errorFor = (issueKey: string): string => errorByIssue.value[issueKey] ?? ""
  const isLoadingFor = (issueKey: string): boolean => isLoadingByIssue.value[issueKey] === true

  const loadIssueBranchMatches = async (
    issueKey: string,
    options: { force?: boolean } = {}
  ): Promise<void> => {
    if (issueKey.length === 0 || isLoadingFor(issueKey)) {
      return
    }
    const isFresh = Date.now() - (loadedAtByIssue[issueKey] ?? 0) < ISSUE_BRANCH_CACHE_TTL_MS
    if (isFresh && options.force !== true) {
      return
    }
    isLoadingByIssue.value = { ...isLoadingByIssue.value, [issueKey]: true }
    errorByIssue.value = { ...errorByIssue.value, [issueKey]: "" }
    try {
      const response = await fetchIssueBranches(issueKey)
      matchesByIssue.value = { ...matchesByIssue.value, [issueKey]: response.matches }
      scopeByIssue.value = { ...scopeByIssue.value, [issueKey]: response.scope }
      loadedAtByIssue[issueKey] = Date.now()
    } catch (error: unknown) {
      errorByIssue.value = {
        ...errorByIssue.value,
        [issueKey]: describeRequestError(error, "The branches for this issue could not be loaded.")
      }
    } finally {
      isLoadingByIssue.value = { ...isLoadingByIssue.value, [issueKey]: false }
    }
  }

  return {
    matchesByIssue: computed(() => matchesByIssue.value),
    errorByIssue: computed(() => errorByIssue.value),
    isLoadingByIssue: computed(() => isLoadingByIssue.value),
    matchesFor,
    scopeFor,
    errorFor,
    isLoadingFor,
    loadIssueBranchMatches
  }
}
