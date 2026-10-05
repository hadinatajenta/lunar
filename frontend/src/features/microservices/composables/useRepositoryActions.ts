import { readonly } from "vue"
import { ApiError } from "../../../lib/http"
import type {
  BranchOption,
  HelperConnection,
  RepositoryActionKind,
  WorkspaceSettings
} from "../types"
import {
  checkoutHelperBranch,
  fetchHelperBranches,
  runHelperDiscard,
  runHelperFetch,
  runHelperPull,
  runHelperStash
} from "../api/helper-api"
import {
  checkoutWorkspaceBranch,
  fetchWorkspaceBranches,
  runWorkspaceDiscard,
  runWorkspaceFetch,
  runWorkspacePull,
  runWorkspaceStash
} from "../api/microservices-api"
import { useMicroservices } from "./useMicroservices"
import {
  BRANCH_CACHE_TTL_MS,
  REPOSITORY_ROOT_MISSING_MESSAGE,
  actionNotice,
  announceRepositoryAction,
  clearActionNotice,
  dropBranchCache,
  failRepositoryAction,
  hasConfiguredRoot,
  readBranchCache,
  readRepositoryActionState,
  registerFetchError,
  requireHelperConnection,
  resolveErrorMessage,
  settleRepositoryAction,
  startRepositoryAction,
  storeBranchCache,
  syncSessionScope,
  workspace
} from "./microservices-store"

const BRANCH_LIST_FALLBACK_MESSAGE = "Failed to load the branch list"

const ACTION_FALLBACK_MESSAGES: Record<RepositoryActionKind, string> = {
  checkout: "Failed to switch the branch",
  fetch: "Failed to fetch from the remote",
  pull: "Failed to pull from the remote",
  stash: "Failed to stash the local changes",
  discard: "Failed to discard the local changes"
}

const resolveRootSettings = (): WorkspaceSettings => {
  const settings = workspace.value
  if (settings === null || !hasConfiguredRoot(settings)) {
    throw new ApiError(REPOSITORY_ROOT_MISSING_MESSAGE, 0)
  }
  return settings
}

const runBySource = <T>(
  runHelper: (connection: HelperConnection, rootPath: string) => Promise<T>,
  runServer: () => Promise<T>
): Promise<T> => {
  const settings = resolveRootSettings()
  if (settings.source === "helper") {
    return runHelper(requireHelperConnection(), settings.root_path)
  }
  return runServer()
}

const refreshServices = async (): Promise<void> => {
  await useMicroservices().loadServices()
}

const loadBranches = async (repositoryName: string): Promise<BranchOption[] | null> => {
  syncSessionScope()
  const cachedEntry = readBranchCache(repositoryName)
  if (cachedEntry !== null && Date.now() - cachedEntry.fetchedAt < BRANCH_CACHE_TTL_MS) {
    return cachedEntry.branches
  }

  try {
    const response = await runBySource(
      (connection, rootPath) => fetchHelperBranches(connection, rootPath, repositoryName),
      () => fetchWorkspaceBranches(repositoryName)
    )
    storeBranchCache(repositoryName, response.branches)
    return response.branches
  } catch (thrownError) {
    registerFetchError(thrownError, BRANCH_LIST_FALLBACK_MESSAGE)
    return null
  }
}

const runRepositoryAction = async (
  repositoryName: string,
  kind: RepositoryActionKind,
  run: () => Promise<string>
): Promise<boolean> => {
  syncSessionScope()
  startRepositoryAction(repositoryName, kind)

  let message: string
  try {
    message = await run()
  } catch (thrownError) {
    const failureMessage = resolveErrorMessage(thrownError, ACTION_FALLBACK_MESSAGES[kind])
    failRepositoryAction(repositoryName, kind, failureMessage)
    registerFetchError(thrownError, failureMessage)
    return false
  }

  await refreshServices()
  settleRepositoryAction(repositoryName)
  announceRepositoryAction(message)
  return true
}

const formatCountLabel = (count: number, singular: string, plural: string): string =>
  `${count} ${count === 1 ? singular : plural}`

const checkoutBranch = (repositoryName: string, branch: string): Promise<boolean> =>
  runRepositoryAction(repositoryName, "checkout", async () => {
    const response = await runBySource(
      (connection, rootPath) => checkoutHelperBranch(connection, rootPath, repositoryName, branch),
      () => checkoutWorkspaceBranch(repositoryName, branch)
    )
    dropBranchCache(repositoryName)
    return `Switched ${repositoryName} to ${response.branch}`
  })

const fetchRepository = (repositoryName: string): Promise<boolean> =>
  runRepositoryAction(repositoryName, "fetch", async () => {
    const response = await runBySource(
      (connection, rootPath) => runHelperFetch(connection, rootPath, repositoryName),
      () => runWorkspaceFetch(repositoryName)
    )
    return `Fetched ${repositoryName}: ${response.ahead} ahead, ${response.behind} behind`
  })

const pullRepository = (repositoryName: string): Promise<boolean> =>
  runRepositoryAction(repositoryName, "pull", async () => {
    const response = await runBySource(
      (connection, rootPath) => runHelperPull(connection, rootPath, repositoryName),
      () => runWorkspacePull(repositoryName)
    )
    return `Pulled ${repositoryName} to ${response.branch} · ${response.updated_relative}`
  })

const stashRepository = (repositoryName: string): Promise<boolean> =>
  runRepositoryAction(repositoryName, "stash", async () => {
    const response = await runBySource(
      (connection, rootPath) => runHelperStash(connection, rootPath, repositoryName),
      () => runWorkspaceStash(repositoryName)
    )
    const stashLabel = formatCountLabel(response.stash_count, "stash entry", "stash entries")
    return `Stashed local changes in ${repositoryName}: ${stashLabel}`
  })

const discardRepository = (repositoryName: string): Promise<boolean> =>
  runRepositoryAction(repositoryName, "discard", async () => {
    const response = await runBySource(
      (connection, rootPath) => runHelperDiscard(connection, rootPath, repositoryName),
      () => runWorkspaceDiscard(repositoryName)
    )
    const fileLabel = formatCountLabel(response.restored_count, "file", "files")
    return `Restored ${fileLabel} in ${repositoryName}`
  })

export function useRepositoryActions() {
  return {
    actionNotice: readonly(actionNotice),
    resolveActionState: readRepositoryActionState,
    clearActionNotice,
    loadBranches,
    checkoutBranch,
    fetchRepository,
    pullRepository,
    stashRepository,
    discardRepository
  }
}
