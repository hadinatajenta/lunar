import { ApiError, http } from "../../../lib/http"
import type {
  BranchCheckoutResponse,
  BranchListResponse,
  RepositoryDiscardResponse,
  RepositoryFetchResponse,
  RepositoryListResponse,
  RepositoryPullResponse,
  RepositoryStashResponse,
  SelectionUpdateRequest,
  SelectionUpdateResponse,
  ServiceListResponse,
  WorkspaceSettings,
  WorkspaceSettingsInput
} from "../types"
import {
  GIT_ACTION_TIMEOUT_MS,
  RequestTimeoutError,
  VM_REQUEST_TIMEOUT_MS,
  withRequestTimeout
} from "./request-timeout"

const WORKSPACE_PATH = "/api/workspace"
const WORKSPACE_REPOSITORIES_PATH = "/api/workspace/repositories"
const WORKSPACE_SERVICES_PATH = "/api/workspace/services"
const WORKSPACE_SELECTIONS_PATH = "/api/workspace/selections"
const WORKSPACE_BRANCHES_PATH = "/api/workspace/branches"
const WORKSPACE_CHECKOUT_PATH = "/api/workspace/checkout"
const WORKSPACE_FETCH_PATH = "/api/workspace/fetch"
const WORKSPACE_PULL_PATH = "/api/workspace/pull"
const WORKSPACE_STASH_PATH = "/api/workspace/stash"
const WORKSPACE_DISCARD_PATH = "/api/workspace/discard"

const BACKEND_UNREACHABLE_MESSAGE =
  "Failed to reach the Lunar backend. Check your connection and try again."
const BACKEND_TIMEOUT_MESSAGE =
  "The Lunar backend did not respond in time. Check that it is running and try again."

function mapBackendError(thrownError: unknown): ApiError {
  if (thrownError instanceof ApiError) {
    if (thrownError.statusCode !== 0) {
      return thrownError
    }
    return new ApiError(BACKEND_UNREACHABLE_MESSAGE, 0, thrownError)
  }
  if (thrownError instanceof RequestTimeoutError) {
    return new ApiError(BACKEND_TIMEOUT_MESSAGE, 0, thrownError)
  }
  if (thrownError instanceof Error) {
    return new ApiError(thrownError.message, 0, thrownError)
  }
  return new ApiError(BACKEND_UNREACHABLE_MESSAGE, 0, thrownError)
}

async function requestBackend<T>(
  run: (signal: AbortSignal) => Promise<T>,
  timeoutMs: number = VM_REQUEST_TIMEOUT_MS
): Promise<T> {
  try {
    return await withRequestTimeout(timeoutMs, run)
  } catch (thrownError) {
    throw mapBackendError(thrownError)
  }
}

export const fetchWorkspace = (): Promise<WorkspaceSettings> =>
  requestBackend((signal) => http.get<WorkspaceSettings>(WORKSPACE_PATH, { signal }))

export const saveWorkspace = (input: WorkspaceSettingsInput): Promise<WorkspaceSettings> =>
  requestBackend((signal) => http.put<WorkspaceSettings>(WORKSPACE_PATH, input, { signal }))

export const fetchWorkspaceRepositories = (): Promise<RepositoryListResponse> =>
  requestBackend((signal) =>
    http.get<RepositoryListResponse>(WORKSPACE_REPOSITORIES_PATH, { signal })
  )

export const fetchWorkspaceServices = (): Promise<ServiceListResponse> =>
  requestBackend((signal) => http.get<ServiceListResponse>(WORKSPACE_SERVICES_PATH, { signal }))

export const saveWorkspaceSelections = (repos: string[]): Promise<SelectionUpdateResponse> => {
  const requestBody: SelectionUpdateRequest = { repos }
  return requestBackend((signal) =>
    http.put<SelectionUpdateResponse>(WORKSPACE_SELECTIONS_PATH, requestBody, { signal })
  )
}

export const fetchWorkspaceBranches = (repositoryName: string): Promise<BranchListResponse> =>
  requestBackend((signal) =>
    http.get<BranchListResponse>(WORKSPACE_BRANCHES_PATH, { signal, params: { name: repositoryName } })
  )

export const checkoutWorkspaceBranch = (
  repositoryName: string,
  branch: string
): Promise<BranchCheckoutResponse> =>
  requestBackend(
    (signal) =>
      http.post<BranchCheckoutResponse>(WORKSPACE_CHECKOUT_PATH, { name: repositoryName, branch }, { signal }),
    GIT_ACTION_TIMEOUT_MS
  )

export const runWorkspaceFetch = (repositoryName: string): Promise<RepositoryFetchResponse> =>
  requestBackend(
    (signal) => http.post<RepositoryFetchResponse>(WORKSPACE_FETCH_PATH, { name: repositoryName }, { signal }),
    GIT_ACTION_TIMEOUT_MS
  )

export const runWorkspacePull = (repositoryName: string): Promise<RepositoryPullResponse> =>
  requestBackend(
    (signal) => http.post<RepositoryPullResponse>(WORKSPACE_PULL_PATH, { name: repositoryName }, { signal }),
    GIT_ACTION_TIMEOUT_MS
  )

export const runWorkspaceStash = (repositoryName: string): Promise<RepositoryStashResponse> =>
  requestBackend(
    (signal) => http.post<RepositoryStashResponse>(WORKSPACE_STASH_PATH, { name: repositoryName }, { signal }),
    GIT_ACTION_TIMEOUT_MS
  )

export const runWorkspaceDiscard = (repositoryName: string): Promise<RepositoryDiscardResponse> =>
  requestBackend(
    (signal) => http.post<RepositoryDiscardResponse>(WORKSPACE_DISCARD_PATH, { name: repositoryName }, { signal }),
    GIT_ACTION_TIMEOUT_MS
  )
