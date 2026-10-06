const FAILURE_MATCHES: { fragment: string; message: string }[] = [
  {
    fragment: "working tree has uncommitted changes",
    message: "The working tree has uncommitted changes. Commit or stash them, then retry."
  },
  {
    fragment: "repository is not cloned",
    message: "The repository is not cloned, so no branch can be created there."
  },
  {
    fragment: "base branch was not found",
    message: "The base branch was not found in this repository. Choose a different base branch."
  },
  {
    fragment: "not a git repository",
    message: "The repository is not a git repository."
  },
  {
    fragment: "repository could not be read",
    message: "The repository could not be read."
  },
  {
    fragment: "could not be checked on bitbucket",
    message: "The existing branch could not be checked on Bitbucket, so no branch was created. Check your connection and retry."
  },
  {
    fragment: "rejected the requested branch creation",
    message: "The repository rejected the branch creation. Check the branch name and retry."
  },
  {
    fragment: "did not finish in time",
    message: "The git operation did not finish in time. Retry."
  },
  {
    fragment: "workspace_allowed_roots",
    message: "The workspace root is outside LUNAR_WORKSPACE_ALLOWED_ROOTS."
  },
  {
    fragment: "branch name is invalid",
    message: "The branch name is invalid."
  },
  {
    fragment: "git operations are not configured",
    message: "Server-side git operations are not configured."
  },
  {
    fragment: "branch already exists",
    message: "The branch already exists in this repository."
  }
]

const SKIPPED_BRANCH_SEPARATOR = ": "

export const describeBranchFailure = (reason: string): string => {
  const normalized = reason.trim().toLowerCase()
  if (normalized.length === 0) {
    return "The branch could not be created."
  }
  for (const entry of FAILURE_MATCHES) {
    if (normalized.includes(entry.fragment)) {
      return entry.message
    }
  }
  const sentence = reason.trim().replace(/[.;]+$/, "")
  return `${sentence.charAt(0).toUpperCase()}${sentence.slice(1)}.`
}

export const existingBranchFromSkipReason = (reason: string): string | null => {
  const separatorIndex = reason.indexOf(SKIPPED_BRANCH_SEPARATOR)
  if (separatorIndex === -1) {
    return null
  }
  const branch = reason.slice(separatorIndex + SKIPPED_BRANCH_SEPARATOR.length).trim()
  return branch.length > 0 ? branch : null
}
