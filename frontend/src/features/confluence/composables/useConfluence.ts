import { ref, computed } from "vue"
import { ApiError } from "../../../lib/http"
import type { ConfluenceDocType, ConfluenceDocument, DocTypeFilter, StatsCounts } from "../types"
import { fetchDocuments, fetchDocumentDetail } from "../api/confluence-api"
import { useSettings } from "../../settings/composables/useSettings"

const VISIBLE_PAGE_SIZE = 6
const SEARCH_DEBOUNCE_MS = 120

const documents = ref<ConfluenceDocument[]>([])
const selectedDocument = ref<ConfluenceDocument | null>(null)
const isLoading = ref(false)
const isDetailLoading = ref(false)
const detailNotFound = ref(false)
const error = ref<string | null>(null)
const errorCode = ref<number | null>(null)
const currentFilter = ref<DocTypeFilter>("all")
const searchInput = ref("")
const searchQuery = ref("")
const visibleCount = ref(VISIBLE_PAGE_SIZE)
const hasLoaded = ref(false)

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
let detailRequestSequence = 0

const countDocumentsOfType = (type: ConfluenceDocType) =>
  documents.value.filter((confluenceDocument) => confluenceDocument.type === type).length

const filteredDocuments = computed(() => {
  const normalizedQuery = searchQuery.value.trim().toLowerCase()
  return documents.value.filter((confluenceDocument) => {
    const matchesType =
      currentFilter.value === "all" || confluenceDocument.type === currentFilter.value
    if (!matchesType) {
      return false
    }
    if (!normalizedQuery) {
      return true
    }
    return (
      confluenceDocument.title.toLowerCase().includes(normalizedQuery) ||
      confluenceDocument.id.toLowerCase().includes(normalizedQuery) ||
      confluenceDocument.space.toLowerCase().includes(normalizedQuery) ||
      confluenceDocument.description.toLowerCase().includes(normalizedQuery)
    )
  })
})

const statsCounts = computed<StatsCounts>(() => ({
  all: documents.value.length,
  ut: countDocumentsOfType("ut"),
  query: countDocumentsOfType("query"),
  sop: countDocumentsOfType("sop")
}))

const visibleDocuments = computed(() =>
  filteredDocuments.value.slice(0, visibleCount.value)
)

const remainingCount = computed(() =>
  filteredDocuments.value.length - visibleDocuments.value.length
)

const visibleSummary = computed(() => {
  if (filteredDocuments.value.length === 0) {
    return ""
  }
  return `Showing ${visibleDocuments.value.length} of ${filteredDocuments.value.length}`
})

export function useConfluence() {
  const { secrets } = useSettings()

  const clearError = () => {
    error.value = null
    errorCode.value = null
  }

  const handleFetchError = (err: unknown) => {
    if (secrets.value?.has_confluence_pat === false) {
      clearError()
      return
    }
    if (err instanceof ApiError) {
      errorCode.value = err.statusCode
      if (err.statusCode === 401) {
        error.value = "Unauthorized access: invalid or expired Confluence PAT"
      } else if (err.statusCode === 502) {
        error.value =
          "Unable to reach Confluence BRI API. Please ensure you are connected to the BRI VPN."
      } else {
        error.value = err.message || "Failed to load Confluence data"
      }
    } else if (err instanceof Error) {
      error.value = err.message
      errorCode.value = 500
    } else {
      error.value = "An unknown error occurred"
      errorCode.value = 500
    }
  }

  const fetchData = async () => {
    isLoading.value = true
    try {
      documents.value = await fetchDocuments()
      hasLoaded.value = true
      clearError()
    } catch (err: unknown) {
      handleFetchError(err)
    } finally {
      isLoading.value = false
    }
  }

  const initialize = async () => {
    if (isLoading.value || hasLoaded.value) {
      return
    }
    await fetchData()
  }

  const openDocument = async (id: string) => {
    detailRequestSequence += 1
    const requestSequence = detailRequestSequence
    const cachedDocument = documents.value.find(
      (confluenceDocument) => confluenceDocument.id === id
    )
    selectedDocument.value = cachedDocument ?? null
    detailNotFound.value = false
    isDetailLoading.value = true
    try {
      const detail = await fetchDocumentDetail(id)
      if (requestSequence !== detailRequestSequence) {
        return
      }
      if (!detail) {
        detailNotFound.value = true
        selectedDocument.value = null
      } else {
        selectedDocument.value = detail
        clearError()
      }
    } catch (err: unknown) {
      if (requestSequence !== detailRequestSequence) {
        return
      }
      if (err instanceof ApiError && err.statusCode === 404) {
        detailNotFound.value = true
        selectedDocument.value = null
      } else {
        handleFetchError(err)
      }
    } finally {
      if (requestSequence === detailRequestSequence) {
        isDetailLoading.value = false
      }
    }
  }

  const closeDocument = () => {
    detailRequestSequence += 1
    selectedDocument.value = null
    detailNotFound.value = false
  }

  const setCurrentFilter = (filter: DocTypeFilter) => {
    currentFilter.value = filter
    visibleCount.value = VISIBLE_PAGE_SIZE
  }

  const setSearchQuery = (value: string) => {
    searchInput.value = value
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }
    searchDebounceTimer = setTimeout(() => {
      searchDebounceTimer = null
      searchQuery.value = value
      visibleCount.value = VISIBLE_PAGE_SIZE
    }, SEARCH_DEBOUNCE_MS)
  }

  const showMore = () => {
    visibleCount.value += VISIBLE_PAGE_SIZE
  }

  const resetConfluenceData = () => {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
      searchDebounceTimer = null
    }
    documents.value = []
    selectedDocument.value = null
    isLoading.value = false
    isDetailLoading.value = false
    detailNotFound.value = false
    error.value = null
    errorCode.value = null
    currentFilter.value = "all"
    searchInput.value = ""
    searchQuery.value = ""
    visibleCount.value = VISIBLE_PAGE_SIZE
    hasLoaded.value = false
  }

  return {
    documents,
    selectedDocument,
    isLoading,
    isDetailLoading,
    detailNotFound,
    error,
    errorCode,
    currentFilter,
    searchInput,
    searchQuery,
    visibleCount,
    hasLoaded,
    filteredDocuments,
    statsCounts,
    visibleDocuments,
    remainingCount,
    visibleSummary,
    fetchData,
    initialize,
    openDocument,
    closeDocument,
    setCurrentFilter,
    setSearchQuery,
    showMore,
    clearError,
    resetConfluenceData
  }
}
