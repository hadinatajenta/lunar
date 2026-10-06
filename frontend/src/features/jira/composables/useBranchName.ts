import { computed, ref } from "vue"
import type { BranchNameSuggestion } from "../branch-types"
import { fetchBranchName } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const nameByIssue = ref<Record<string, BranchNameSuggestion>>({})
const errorByIssue = ref<Record<string, string>>({})
const isLoadingByIssue = ref<Record<string, boolean>>({})
const loadedIssues = new Set<string>()

export function useBranchName() {
  const nameFor = (issueKey: string): BranchNameSuggestion | null => nameByIssue.value[issueKey] ?? null
  const errorFor = (issueKey: string): string => errorByIssue.value[issueKey] ?? ""
  const isLoadingFor = (issueKey: string): boolean => isLoadingByIssue.value[issueKey] === true

  const loadBranchName = async (
    issueKey: string,
    options: { force?: boolean } = {}
  ): Promise<void> => {
    if (issueKey.length === 0 || isLoadingFor(issueKey)) {
      return
    }
    if (loadedIssues.has(issueKey) && options.force !== true) {
      return
    }
    isLoadingByIssue.value = { ...isLoadingByIssue.value, [issueKey]: true }
    errorByIssue.value = { ...errorByIssue.value, [issueKey]: "" }
    try {
      const suggestion = await fetchBranchName(issueKey)
      nameByIssue.value = { ...nameByIssue.value, [issueKey]: suggestion }
      loadedIssues.add(issueKey)
    } catch (error: unknown) {
      errorByIssue.value = {
        ...errorByIssue.value,
        [issueKey]: describeRequestError(error, "The branch name for this issue could not be generated.")
      }
    } finally {
      isLoadingByIssue.value = { ...isLoadingByIssue.value, [issueKey]: false }
    }
  }

  return {
    nameByIssue: computed(() => nameByIssue.value),
    errorByIssue: computed(() => errorByIssue.value),
    isLoadingByIssue: computed(() => isLoadingByIssue.value),
    nameFor,
    errorFor,
    isLoadingFor,
    loadBranchName
  }
}
