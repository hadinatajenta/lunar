import { computed, ref } from "vue"
import type { RepositoryOption } from "../branch-types"
import { fetchRepositoryPage } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"

const repositories = ref<RepositoryOption[]>([])
const localRepositoryCount = ref(0)
const remoteRepositoryCount = ref(0)
const catalogueError = ref("")
const isLoadingCatalogue = ref(false)
let requestSequence = 0

export function useRepositoryCatalogue() {
  const searchRepositories = async (search: string): Promise<void> => {
    requestSequence += 1
    const sequence = requestSequence
    isLoadingCatalogue.value = true
    catalogueError.value = ""
    try {
      const page = await fetchRepositoryPage(search)
      if (sequence !== requestSequence) {
        return
      }
      repositories.value = page.repositories
      localRepositoryCount.value = page.local_repository_count
      remoteRepositoryCount.value = page.remote_repository_count
    } catch (error: unknown) {
      if (sequence !== requestSequence) {
        return
      }
      catalogueError.value = describeRequestError(error, "The repository list could not be loaded.")
    } finally {
      if (sequence === requestSequence) {
        isLoadingCatalogue.value = false
      }
    }
  }

  return {
    repositories: computed(() => repositories.value),
    localRepositoryCount: computed(() => localRepositoryCount.value),
    remoteRepositoryCount: computed(() => remoteRepositoryCount.value),
    catalogueError: computed(() => catalogueError.value),
    isLoadingCatalogue: computed(() => isLoadingCatalogue.value),
    searchRepositories
  }
}
