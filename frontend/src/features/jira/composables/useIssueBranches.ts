import { computed, ref } from "vue"
import type { IssueBranchMatch } from "../branch-types"
import { fetchSprintBranches } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const SPRINT_BRANCH_CACHE_TTL_MS = 30_000

const sprintMatches = ref<Record<string, IssueBranchMatch[]>>({})
const sprintMatchesLoaded = ref(false)
const sprintBranchesError = ref("")
const isLoadingSprintBranches = ref(false)
let sprintLoadedAt = 0

export function useIssueBranches() {
  const branchesForIssue = (issueKey: string): IssueBranchMatch[] => {
    return sprintMatches.value[issueKey] ?? []
  }

  const loadSprintBranches = async (squad: string, options: { force?: boolean } = {}): Promise<void> => {
    const isFresh = Date.now() - sprintLoadedAt < SPRINT_BRANCH_CACHE_TTL_MS
    if (isLoadingSprintBranches.value || (isFresh && options.force !== true)) {
      return
    }
    isLoadingSprintBranches.value = true
    sprintBranchesError.value = ""
    try {
      const response = await fetchSprintBranches(squad)
      sprintMatches.value = response.matches
      sprintMatchesLoaded.value = true
      sprintLoadedAt = Date.now()
    } catch (error: unknown) {
      sprintBranchesError.value = describeRequestError(error, "The branch scan for this sprint failed.")
    } finally {
      isLoadingSprintBranches.value = false
    }
  }

  return {
    sprintMatches: computed(() => sprintMatches.value),
    sprintMatchesLoaded: computed(() => sprintMatchesLoaded.value),
    sprintBranchesError: computed(() => sprintBranchesError.value),
    isLoadingSprintBranches: computed(() => isLoadingSprintBranches.value),
    branchesForIssue,
    loadSprintBranches
  }
}
