import { http } from "../../../lib/http"
import type {
  BranchNameSuggestion,
  CreateBranchesRequest,
  CreateBranchesResponse,
  IssueBranches,
  RepositoryBaseBranchOptions,
  RepositoryFeedbackAction,
  RepositoryFeedbackResponse,
  RepositoryPage,
  RepositorySuggestions,
  SprintBranches
} from "../branch-types"

const JIRA_PREFIX = "/api/jira"
const WORKSPACE_PREFIX = "/api/workspace"

export async function fetchIssueBranches(issueKey: string): Promise<IssueBranches> {
  return http.get<IssueBranches>(`${JIRA_PREFIX}/issues/${encodeURIComponent(issueKey)}/branches`)
}

export async function fetchSprintBranches(squad: string): Promise<SprintBranches> {
  return http.get<SprintBranches>(`${JIRA_PREFIX}/sprint/branches`, {
    params: { squad: squad.length > 0 ? squad : undefined }
  })
}

export async function fetchRepositorySuggestions(issueKey: string): Promise<RepositorySuggestions> {
  return http.get<RepositorySuggestions>(
    `${JIRA_PREFIX}/issues/${encodeURIComponent(issueKey)}/repository-suggestions`
  )
}

export async function fetchBranchName(issueKey: string): Promise<BranchNameSuggestion> {
  return http.get<BranchNameSuggestion>(
    `${JIRA_PREFIX}/issues/${encodeURIComponent(issueKey)}/branch-name`
  )
}

export async function fetchRepositoryPage(search: string): Promise<RepositoryPage> {
  return http.get<RepositoryPage>(`${JIRA_PREFIX}/repositories`, {
    params: { search: search.length > 0 ? search : undefined }
  })
}

export async function fetchBaseBranchOptions(repo: string): Promise<RepositoryBaseBranchOptions> {
  return http.get<RepositoryBaseBranchOptions>(
    `${WORKSPACE_PREFIX}/repos/${encodeURIComponent(repo)}/base-branch-options`
  )
}

export async function createBranches(request: CreateBranchesRequest): Promise<CreateBranchesResponse> {
  return http.post<CreateBranchesResponse>(`${WORKSPACE_PREFIX}/branches`, request)
}

export async function recordRepositoryFeedback(
  issueKey: string,
  repo: string,
  action: RepositoryFeedbackAction
): Promise<RepositoryFeedbackResponse> {
  return http.post<RepositoryFeedbackResponse>(
    `${JIRA_PREFIX}/issues/${encodeURIComponent(issueKey)}/repository-feedback`,
    { repo, action }
  )
}
