import { ref, computed } from "vue"
import { ApiError } from "../../../lib/http"
import type {
  JiraIssue,
  JiraSprint,
  SubfilterCategory,
  DocTypeFilter
} from "../types"
import { fetchMyIssues, fetchBacklog, fetchIssueDetail } from "../api/jira-api"
import { useSettings } from "../../settings/composables/useSettings"

function normalizeStatus(rawStatus: string): "open" | "progress" | "done" {
  const normalized = (rawStatus || "").toLowerCase()
  if (normalized === "progress" || normalized.includes("progress")) {
    return "progress"
  }
  if (
    normalized === "done" ||
    normalized.includes("done") ||
    normalized.includes("closed") ||
    normalized.includes("resolved")
  ) {
    return "done"
  }
  return "open"
}

function normalizeIssue(
  issue: Partial<JiraIssue> & {
    fields?: {
      summary?: string
      description?: string
      status?: { name?: string }
      priority?: { name?: string }
      issuetype?: { name?: string; subtask?: boolean }
      story_points?: number
    }
  }
): JiraIssue {
  const summary = issue.title || issue.fields?.summary || ""
  const description = issue.description || issue.fields?.description || ""
  const rawStatus =
    typeof issue.status === "string"
      ? issue.status
      : issue.fields?.status?.name || "open"
  const rawPriority =
    typeof issue.priority === "string"
      ? issue.priority
      : issue.fields?.priority?.name || "medium"
  const points = issue.points ?? issue.fields?.story_points ?? 0
  const key = issue.key || issue.id || ""

  let kind = issue.kind || ""
  let label = issue.label || ""

  const upperKey = key.toUpperCase()
  const lowerSummary = summary.toLowerCase()
  const typeName = (issue.fields?.issuetype?.name || "").toLowerCase()

  if (!kind) {
    if (
      upperKey.startsWith("SOP-") ||
      lowerSummary.includes("dokumen sop") ||
      lowerSummary.includes("sop operasional") ||
      lowerSummary.includes("[sop]") ||
      lowerSummary.includes(" sop ") ||
      lowerSummary.startsWith("sop ") ||
      lowerSummary.endsWith(" sop")
    ) {
      kind = "sop"
      label = "SOP"
    } else if (
      upperKey.startsWith("UT-") ||
      lowerSummary.includes("dokumen ut") ||
      lowerSummary.includes("unit test") ||
      lowerSummary.includes("[ut]") ||
      lowerSummary.includes(" ut ") ||
      lowerSummary.startsWith("ut ") ||
      lowerSummary.endsWith(" ut") ||
      typeName === "ut"
    ) {
      kind = "ut"
      label = "UT"
    } else if (
      upperKey.startsWith("QR-") ||
      lowerSummary.includes("query review") ||
      lowerSummary.includes("review query") ||
      lowerSummary.includes("[query]") ||
      lowerSummary.includes("[qr]") ||
      lowerSummary.includes("query ") ||
      lowerSummary.startsWith("query")
    ) {
      kind = "query"
      label = "QR"
    } else if (
      upperKey.startsWith("BUG-") ||
      typeName.includes("bug") ||
      typeName.includes("defect") ||
      lowerSummary.includes("bug") ||
      lowerSummary.includes("defect")
    ) {
      kind = "bug"
      label = "BUG"
    } else if (
      upperKey.startsWith("SUB-") ||
      Boolean(issue.fields?.issuetype?.subtask) ||
      typeName.includes("sub")
    ) {
      kind = "subtask"
      label = "SUB"
    } else if (typeName.includes("story") || lowerSummary.startsWith("sebagai ")) {
      kind = "story"
      label = "STORY"
    } else {
      kind = "task"
      label = "TASK"
    }
  }

  if (!label) {
    label = kind.toUpperCase()
  }

  return {
    ...issue,
    id: String(issue.id || key),
    key,
    title: summary,
    description,
    status: rawStatus,
    priority: rawPriority,
    points,
    kind,
    label
  }
}

function normalizeSprint(
  sprint: Partial<JiraSprint> & { state?: string; startDate?: string; endDate?: string }
): JiraSprint {
  const isActive = sprint.is_active ?? sprint.state === "active"
  const dates =
    sprint.dates ||
    (sprint.startDate && sprint.endDate
      ? `${sprint.startDate} · ${sprint.endDate}`
      : isActive
        ? "Active sprint"
        : "Backlog")

  return {
    id: String(sprint.id || ""),
    name: sprint.name || "Sprint",
    is_active: isActive,
    dates,
    issues: (sprint.issues || []).map(normalizeIssue)
  }
}

const issues = ref<JiraIssue[]>([])
const sprints = ref<JiraSprint[]>([])
const activeTab = ref<"assigned" | "backlog">("assigned")
const currentCategory = ref<SubfilterCategory>("docs")
const currentDocType = ref<DocTypeFilter>("all")
const detectedSquad = ref<string>("")
const availableSquads = ref<string[]>([])
const selectedSquad = ref<string>("")
const columnLimits = ref<Record<string, number>>({
  open: 5,
  progress: 5,
  done: 5
})
const selectedIssue = ref<JiraIssue | null>(null)
const isLoading = ref(false)
const error = ref<string | null>(null)
const errorCode = ref<number | null>(null)
const lastLoadedAt = ref<number | null>(null)

const assignedCount = computed(() => issues.value.length)

const backlogCount = computed(() =>
  sprints.value.reduce((total, sprint) => total + sprint.issues.length, 0)
)

const categoryCounts = computed(() => {
  const all = issues.value.length
  const docs = issues.value.filter((issue) =>
    ["ut", "query", "sop"].includes(issue.kind)
  ).length
  const bugs = issues.value.filter((issue) => issue.kind === "bug").length
  const subtasks = issues.value.filter((issue) => issue.kind === "subtask").length
  return { all, docs, bugs, subtasks }
})

const filteredIssues = computed(() => {
  if (currentCategory.value === "all") {
    return issues.value
  }
  if (currentCategory.value === "bugs") {
    return issues.value.filter((issue) => issue.kind === "bug")
  }
  if (currentCategory.value === "subtasks") {
    return issues.value.filter((issue) => issue.kind === "subtask")
  }
  const docs = issues.value.filter((issue) =>
    ["ut", "query", "sop"].includes(issue.kind)
  )
  if (currentDocType.value === "all") {
    return docs
  }
  return docs.filter((issue) => issue.kind === currentDocType.value)
})

const boardColumns = computed(() => {
  const open: JiraIssue[] = []
  const progress: JiraIssue[] = []
  const done: JiraIssue[] = []

  for (const issue of filteredIssues.value) {
    const normalized = normalizeStatus(issue.status)
    if (normalized === "progress") {
      progress.push(issue)
    } else if (normalized === "done") {
      done.push(issue)
    } else {
      open.push(issue)
    }
  }

  return { open, progress, done }
})

const isVpnError = computed(() => errorCode.value === 502)
const isAuthError = computed(() => errorCode.value === 401)

export function useJira() {
  const { secrets, fetchSettings } = useSettings()

  const loadMore = (status: string) => {
    const currentLimit = columnLimits.value[status] ?? 5
    columnLimits.value = {
      ...columnLimits.value,
      [status]: currentLimit + 5
    }
  }

  const openModal = async (issue: JiraIssue) => {
    selectedIssue.value = issue
    try {
      const detail = await fetchIssueDetail(issue.key || issue.id)
      if (detail && selectedIssue.value?.id === issue.id) {
        selectedIssue.value = normalizeIssue({
          ...selectedIssue.value,
          ...detail
        })
      }
    } catch {
      return
    }
  }

  const closeModal = () => {
    selectedIssue.value = null
  }

  const setActiveTab = (tab: "assigned" | "backlog") => {
    activeTab.value = tab
  }

  const setCategory = (category: SubfilterCategory) => {
    currentCategory.value = category
  }

  const setDocType = (docType: DocTypeFilter) => {
    currentDocType.value = docType
  }

  const setSquad = async (squad: string) => {
    selectedSquad.value = squad
    isLoading.value = true
    try {
      const fetchedBacklog = await fetchBacklog(squad === "all" ? "all" : squad)
      if (fetchedBacklog && Array.isArray(fetchedBacklog.sprints)) {
        sprints.value = fetchedBacklog.sprints.map(normalizeSprint)
      }
    } catch (err: unknown) {
      handleFetchError(err)
    } finally {
      isLoading.value = false
    }
  }

  const clearError = () => {
    error.value = null
    errorCode.value = null
  }

  const handleFetchError = (err: unknown) => {
    if (secrets.value?.has_jira_pat === false) {
      clearError()
      return
    }
    if (err instanceof ApiError) {
      errorCode.value = err.statusCode
      if (err.statusCode === 401) {
        error.value = "Unauthorized access: invalid or expired Jira PAT"
      } else if (err.statusCode === 502) {
        error.value =
          "Unable to reach Jira BRI API. Please ensure you are connected to the BRI VPN."
      } else {
        error.value = err.message || "Failed to load Jira data"
      }
    } else if (err instanceof Error) {
      error.value = err.message
      errorCode.value = 500
    } else {
      error.value = "An unknown error occurred"
      errorCode.value = 500
    }
  }

  const loadJiraData = async (options: { refresh?: boolean } = {}) => {
    const [fetchedIssues, fetchedBacklog] = await Promise.all([
      fetchMyIssues(options),
      fetchBacklog(selectedSquad.value || undefined, options)
    ])

    if (Array.isArray(fetchedIssues)) {
      issues.value = fetchedIssues.map(normalizeIssue)
    }
    if (fetchedBacklog && Array.isArray(fetchedBacklog.sprints)) {
      sprints.value = fetchedBacklog.sprints.map(normalizeSprint)
      if (fetchedBacklog.detected_squad) {
        detectedSquad.value = fetchedBacklog.detected_squad
        if (!selectedSquad.value) {
          selectedSquad.value = fetchedBacklog.detected_squad
        }
      }
      if (fetchedBacklog.available_squads) {
        availableSquads.value = fetchedBacklog.available_squads
      }
    }
    lastLoadedAt.value = Date.now()
    clearError()
  }

  const fetchData = async () => {
    isLoading.value = true
    try {
      await loadJiraData({ refresh: true })
    } catch (err: unknown) {
      handleFetchError(err)
    } finally {
      isLoading.value = false
    }
  }

  const initialize = async () => {
    if (isLoading.value || lastLoadedAt.value !== null) {
      return
    }
    await fetchData()
  }

  const prefetchJiraData = async () => {
    if (secrets.value === null) {
      await fetchSettings()
    }
    if (secrets.value?.has_jira_pat === false) {
      return
    }
    await initialize()
  }

  const resetJiraData = () => {
    issues.value = []
    sprints.value = []
    detectedSquad.value = ""
    availableSquads.value = []
    selectedSquad.value = ""
    selectedIssue.value = null
    error.value = null
    errorCode.value = null
    lastLoadedAt.value = null
    isLoading.value = false
  }

  return {
    issues,
    sprints,
    activeTab,
    currentCategory,
    currentDocType,
    detectedSquad,
    availableSquads,
    selectedSquad,
    columnLimits,
    selectedIssue,
    isLoading,
    error,
    errorCode,
    isVpnError,
    isAuthError,
    assignedCount,
    backlogCount,
    categoryCounts,
    filteredIssues,
    boardColumns,
    loadMore,
    openModal,
    closeModal,
    setActiveTab,
    setCategory,
    setDocType,
    setSquad,
    fetchData,
    clearError,
    initialize,
    prefetchJiraData,
    resetJiraData
  }
}
