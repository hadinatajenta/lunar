import { computed, ref } from "vue"
import type {
  RepositoryCatalogue,
  RepositorySuggestion,
  SuggestionConfidence
} from "../branch-types"
import { fetchRepositorySuggestions } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const suggestionsByIssue = ref<Record<string, RepositorySuggestion[]>>({})
const confidenceByIssue = ref<Record<string, SuggestionConfidence>>({})
const unavailableByIssue = ref<Record<string, { signal: string; reason: string }[]>>({})
const catalogueByIssue = ref<Record<string, RepositoryCatalogue | null>>({})
const errorByIssue = ref<Record<string, string>>({})
const isLoadingByIssue = ref<Record<string, boolean>>({})
const loadedIssues = new Set<string>()

export function useRepositorySuggestion() {
  const suggestionsFor = (issueKey: string): RepositorySuggestion[] =>
    suggestionsByIssue.value[issueKey] ?? []
  const confidenceFor = (issueKey: string): SuggestionConfidence | null =>
    confidenceByIssue.value[issueKey] ?? null
  const catalogueFor = (issueKey: string): RepositoryCatalogue | null =>
    catalogueByIssue.value[issueKey] ?? null
  const unavailableFor = (issueKey: string): string[] =>
    (unavailableByIssue.value[issueKey] ?? []).map((entry) => entry.reason)
  const errorFor = (issueKey: string): string => errorByIssue.value[issueKey] ?? ""
  const isLoadingFor = (issueKey: string): boolean => isLoadingByIssue.value[issueKey] === true
  const hasLoadedFor = (issueKey: string): boolean => loadedIssues.has(issueKey)

  const loadSuggestions = async (
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
      const response = await fetchRepositorySuggestions(issueKey)
      suggestionsByIssue.value = { ...suggestionsByIssue.value, [issueKey]: response.suggestions }
      confidenceByIssue.value = { ...confidenceByIssue.value, [issueKey]: response.confidence }
      unavailableByIssue.value = {
        ...unavailableByIssue.value,
        [issueKey]: response.signals_unavailable
      }
      catalogueByIssue.value = { ...catalogueByIssue.value, [issueKey]: response.catalogue ?? null }
      loadedIssues.add(issueKey)
    } catch (error: unknown) {
      errorByIssue.value = {
        ...errorByIssue.value,
        [issueKey]: describeRequestError(error, "Lunar could not suggest a repository for this issue.")
      }
    } finally {
      isLoadingByIssue.value = { ...isLoadingByIssue.value, [issueKey]: false }
    }
  }

  return {
    suggestionsByIssue: computed(() => suggestionsByIssue.value),
    errorByIssue: computed(() => errorByIssue.value),
    isLoadingByIssue: computed(() => isLoadingByIssue.value),
    suggestionsFor,
    confidenceFor,
    catalogueFor,
    unavailableFor,
    errorFor,
    isLoadingFor,
    hasLoadedFor,
    loadSuggestions
  }
}
