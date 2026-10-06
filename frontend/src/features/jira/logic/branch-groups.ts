import type { IssueBranchMatch, WorkspaceScope } from "../branch-types"

export interface BranchRow {
  repo: string
  branch: string
  source: "local" | "remote" | "both"
  isCloned: boolean
  lastTouched: string
}

export const NOT_CLONED_LABEL = "not cloned"

export const branchRows = (matches: IssueBranchMatch[]): BranchRow[] => {
  return [...matches]
    .sort((left, right) => {
      if (left.repo !== right.repo) {
        return left.repo.localeCompare(right.repo)
      }
      return left.branch.localeCompare(right.branch)
    })
    .map((match) => ({
      repo: match.repo,
      branch: match.branch,
      source: match.source,
      isCloned: match.is_cloned,
      lastTouched: match.last_commit_relative ?? NOT_CLONED_LABEL
    }))
}

export const remoteOnlyRepositories = (matches: IssueBranchMatch[]): string[] => {
  const repositories: string[] = []
  for (const match of matches) {
    if (match.source !== "remote" || repositories.includes(match.repo)) {
      continue
    }
    repositories.push(match.repo)
  }
  return repositories
}

export const remoteNoticeText = (repositories: string[]): string => {
  if (repositories.length === 0) {
    return ""
  }
  const names = repositories.join(", ")
  if (repositories.length === 1) {
    return `This branch exists on Bitbucket in ${names}, which is not in your workspace. Clone it rather than creating a second branch.`
  }
  return `These branches exist on Bitbucket in ${names}, which are not in your workspace. Clone them rather than creating a second branch.`
}

export const emptyBranchMessage = (issueKey: string, scope: WorkspaceScope): string => {
  const missingCount = scope.remote_repository_count - scope.local_repository_count
  if (!scope.verified_remotely && missingCount > 0) {
    return "No branch found locally. Bitbucket could not be reached, so this is not conclusive."
  }
  if (scope.remote_repositories_checked > 0) {
    return `No branch found in the ${scope.local_repository_count} repositories Lunar can see, or in the ${scope.remote_repositories_checked} most likely repositories checked on Bitbucket.`
  }
  if (missingCount > 0) {
    return `No branch found in the ${scope.local_repository_count} of ${scope.remote_repository_count} repositories Lunar can see. ${missingCount} are not cloned locally.`
  }
  return `No branch found for ${issueKey} in the ${scope.local_repository_count} repositories Lunar can see.`
}
