import { ApiError, http } from "../../../lib/http"
import type {
  RepositoryListResponse,
  SelectionUpdateRequest,
  SelectionUpdateResponse,
  ServiceListResponse,
  WorkspaceSettings,
  WorkspaceSettingsInput
} from "../types"
import { RequestTimeoutError, VM_REQUEST_TIMEOUT_MS, withRequestTimeout } from "./request-timeout"

const WORKSPACE_PATH = "/api/workspace"
const WORKSPACE_REPOSITORIES_PATH = "/api/workspace/repositories"
const WORKSPACE_SERVICES_PATH = "/api/workspace/services"
const WORKSPACE_SELECTIONS_PATH = "/api/workspace/selections"

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

async function requestBackend<T>(run: (signal: AbortSignal) => Promise<T>): Promise<T> {
  try {
    return await withRequestTimeout(VM_REQUEST_TIMEOUT_MS, run)
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
