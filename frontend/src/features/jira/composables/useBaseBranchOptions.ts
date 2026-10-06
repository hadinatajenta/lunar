import { computed, ref } from "vue"
import type { RepositoryBaseBranchOptions } from "../branch-types"
import { fetchBaseBranchOptions } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const optionsByRepo = ref<Record<string, RepositoryBaseBranchOptions>>({})
const errorByRepo = ref<Record<string, string>>({})
const isLoadingByRepo = ref<Record<string, boolean>>({})

export function useBaseBranchOptions() {
  const optionsFor = (repo: string): RepositoryBaseBranchOptions | null => optionsByRepo.value[repo] ?? null
  const errorFor = (repo: string): string => errorByRepo.value[repo] ?? ""
  const isLoadingFor = (repo: string): boolean => isLoadingByRepo.value[repo] === true

  const loadBaseBranchOptions = async (repo: string): Promise<void> => {
    if (repo.length === 0 || isLoadingFor(repo)) {
      return
    }
    isLoadingByRepo.value = { ...isLoadingByRepo.value, [repo]: true }
    errorByRepo.value = { ...errorByRepo.value, [repo]: "" }
    try {
      const response = await fetchBaseBranchOptions(repo)
      optionsByRepo.value = { ...optionsByRepo.value, [repo]: response }
    } catch (error: unknown) {
      errorByRepo.value = {
        ...errorByRepo.value,
        [repo]: describeRequestError(error, "The base branches for this repository could not be loaded.")
      }
    } finally {
      isLoadingByRepo.value = { ...isLoadingByRepo.value, [repo]: false }
    }
  }

  return {
    optionsByRepo: computed(() => optionsByRepo.value),
    errorByRepo: computed(() => errorByRepo.value),
    isLoadingByRepo: computed(() => isLoadingByRepo.value),
    optionsFor,
    errorFor,
    isLoadingFor,
    loadBaseBranchOptions
  }
}
