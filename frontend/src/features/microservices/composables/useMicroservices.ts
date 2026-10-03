import { readonly } from "vue"
import type {
  HelperConnection,
  RepositoryDetail,
  RepositorySummary,
  ValidatePathResponse,
  WorkspaceSettings,
  WorkspaceSettingsInput
} from "../types"
import {
  fetchWorkspace,
  fetchWorkspaceRepositories,
  fetchWorkspaceServices,
  saveWorkspace as saveWorkspaceRequest,
  saveWorkspaceSelections
} from "../api/microservices-api"
import {
  HelperConnectionError,
  fetchHelperHealth,
  fetchHelperRepositories,
  fetchHelperServices,
  fetchHelperToken,
  validateHelperPath
} from "../api/helper-api"
import {
  HELPER_TOKEN_MISSING_MESSAGE,
  REPOSITORY_ROOT_MISSING_MESSAGE,
  discardHelperToken,
  error,
  hasConfiguredRoot,
  hasHelperToken,
  helperStatus,
  helperVersion,
  isLoading,
  isSaving,
  readSelectedNames,
  registerFetchError,
  repositories,
  requireHelperConnection,
  resetMicroservicesStore,
  resolveErrorMessage,
  resolveHelperConnection,
  selectedNames,
  services,
  storeHelperToken,
  syncSessionScope,
  workspace
} from "./microservices-store"

const loadWorkspace = async (): Promise<WorkspaceSettings | null> => {
  syncSessionScope()
  isLoading.value = true
  error.value = null
  try {
    const settings = await fetchWorkspace()
    workspace.value = settings
    selectedNames.value = readSelectedNames(settings) ?? []
    if (settings.has_helper_token === false) {
      discardHelperToken()
    }
    if (resolveHelperConnection() === null) {
      await connectHelper(settings.helper_url, false)
    }
    return settings
  } catch (thrownError) {
    registerFetchError(thrownError, "Failed to load the workspace settings")
    return null
  } finally {
    isLoading.value = false
  }
}

const saveWorkspace = async (input: WorkspaceSettingsInput): Promise<boolean> => {
  syncSessionScope()
  isSaving.value = true
  error.value = null
  try {
    const settings = await saveWorkspaceRequest(input)
    workspace.value = settings
    const nextSelectedNames = readSelectedNames(settings)
    if (nextSelectedNames !== null) {
      selectedNames.value = nextSelectedNames
    }
    if (typeof input.helper_token === "string") {
      if (input.helper_token.length > 0) {
        storeHelperToken(input.helper_token)
      } else {
        discardHelperToken()
      }
    } else if (settings.has_helper_token === false) {
      discardHelperToken()
    }
    return true
  } catch (thrownError) {
    registerFetchError(thrownError, "Failed to save the workspace settings")
    return false
  } finally {
    isSaving.value = false
  }
}

const resolveWorkspace = async (): Promise<WorkspaceSettings | null> => {
  if (workspace.value === null) {
    return loadWorkspace()
  }
  return workspace.value
}

const loadRepositories = async (): Promise<RepositorySummary[]> => {
  syncSessionScope()
  const settings = await resolveWorkspace()
  if (settings === null) {
    return []
  }
  if (!hasConfiguredRoot(settings)) {
    error.value = REPOSITORY_ROOT_MISSING_MESSAGE
    return []
  }

  isLoading.value = true
  error.value = null
  try {
    const response =
      settings.source === "helper"
        ? await fetchHelperRepositories(requireHelperConnection(), settings.root_path)
        : await fetchWorkspaceRepositories()
    repositories.value = response.repositories
    if (settings.source === "helper") {
      helperStatus.value = "online"
    }
    return repositories.value
  } catch (thrownError) {
    registerFetchError(thrownError, "Failed to load the repository list")
    return []
  } finally {
    isLoading.value = false
  }
}

const loadServices = async (): Promise<RepositoryDetail[]> => {
  syncSessionScope()
  const settings = await resolveWorkspace()
  if (settings === null) {
    return []
  }
  if (!hasConfiguredRoot(settings)) {
    error.value = REPOSITORY_ROOT_MISSING_MESSAGE
    return []
  }
  if (settings.source === "helper" && selectedNames.value.length === 0) {
    services.value = []
    return []
  }

  isLoading.value = true
  error.value = null
  try {
    const response =
      settings.source === "helper"
        ? await fetchHelperServices(requireHelperConnection(), settings.root_path, selectedNames.value)
        : await fetchWorkspaceServices()
    services.value = response.services
    if (settings.source === "helper") {
      helperStatus.value = "online"
    }
    return services.value
  } catch (thrownError) {
    registerFetchError(thrownError, "Failed to load the repository details")
    return []
  } finally {
    isLoading.value = false
  }
}

const saveSelections = async (repos: string[]): Promise<boolean> => {
  syncSessionScope()
  isSaving.value = true
  error.value = null
  try {
    await saveWorkspaceSelections(repos)
    selectedNames.value = [...repos]
    services.value = services.value.filter((service) => repos.includes(service.name))
    return true
  } catch (thrownError) {
    registerFetchError(thrownError, "Failed to save the repository selections")
    return false
  } finally {
    isSaving.value = false
  }
}

const resolveHealthConnection = (connection?: HelperConnection): HelperConnection => {
  if (connection !== undefined) {
    return connection
  }
  const storedConnection = resolveHelperConnection()
  if (storedConnection !== null) {
    return storedConnection
  }
  const storedUrl = workspace.value?.helper_url.trim() ?? ""
  return { helperUrl: storedUrl, helperToken: "" }
}

const connectHelper = async (helperUrl?: string, reportErrors = true): Promise<boolean> => {
  syncSessionScope()
  if (reportErrors) {
    error.value = null
  }
  const targetUrl = (helperUrl ?? "").trim() || workspace.value?.helper_url || ""
  helperStatus.value = "checking"
  try {
    const token = await fetchHelperToken(targetUrl)
    storeHelperToken(token)
    helperStatus.value = "online"
    return true
  } catch (thrownError) {
    helperStatus.value = "offline"
    if (reportErrors) {
      error.value = resolveErrorMessage(thrownError, "The Lunar helper could not be reached")
    }
    return false
  }
}

const checkHelperHealth = async (connection?: HelperConnection): Promise<boolean> => {
  syncSessionScope()
  const healthConnection = resolveHealthConnection(connection)

  helperStatus.value = "checking"
  try {
    const health = await fetchHelperHealth(healthConnection)
    const isHealthy = health.status === "ok"
    helperVersion.value = isHealthy ? health.version : ""
    helperStatus.value = isHealthy ? "online" : "offline"
    if (isHealthy && resolveHelperConnection() === null) {
      error.value = HELPER_TOKEN_MISSING_MESSAGE
    }
    return isHealthy
  } catch (thrownError) {
    helperStatus.value = "offline"
    helperVersion.value = ""
    error.value = resolveErrorMessage(thrownError, "The Lunar helper could not be reached")
    return false
  }
}

const validateRootPath = async (
  rootPath: string,
  connection?: HelperConnection
): Promise<ValidatePathResponse> => {
  syncSessionScope()
  const resolvedConnection = connection ?? resolveHelperConnection()
  if (resolvedConnection === null) {
    return { valid: false, reason: HELPER_TOKEN_MISSING_MESSAGE }
  }

  try {
    const validation = await validateHelperPath(resolvedConnection, rootPath)
    if (validation.valid) {
      helperStatus.value = "online"
    }
    return validation
  } catch (thrownError) {
    if (thrownError instanceof HelperConnectionError) {
      helperStatus.value = "offline"
      helperVersion.value = ""
    }
    return {
      valid: false,
      reason: resolveErrorMessage(thrownError, "Failed to validate the repository root folder")
    }
  }
}

const clearError = (): void => {
  error.value = null
}

export function useMicroservices() {
  return {
    workspace: readonly(workspace),
    repositories: readonly(repositories),
    services: readonly(services),
    selectedNames: readonly(selectedNames),
    isLoading: readonly(isLoading),
    isSaving: readonly(isSaving),
    error: readonly(error),
    helperStatus: readonly(helperStatus),
    helperVersion: readonly(helperVersion),
    hasHelperToken,
    loadWorkspace,
    saveWorkspace,
    loadRepositories,
    loadServices,
    saveSelections,
    checkHelperHealth,
    connectHelper,
    validateRootPath,
    clearError,
    resetMicroservicesStore
  }
}
