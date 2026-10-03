import type { ChangedFile, RepositoryDetail } from "../types"

export type DirtyLevel = "clean" | "dirty" | "heavy"

const HEAVY_DIRTY_FILE_THRESHOLD = 5

export function countChangedFiles(service: RepositoryDetail): number {
  return service.files.length
}

export function resolveDirtyLevel(changedFileCount: number): DirtyLevel {
  if (changedFileCount === 0) {
    return "clean"
  }
  if (changedFileCount >= HEAVY_DIRTY_FILE_THRESHOLD) {
    return "heavy"
  }
  return "dirty"
}

export function formatDirtyLabel(changedFileCount: number): string {
  if (changedFileCount === 0) {
    return "Clean"
  }
  return `${changedFileCount} file${changedFileCount === 1 ? "" : "s"} changed`
}

export function formatChangedFileCount(changedFileCount: number): string {
  return `${changedFileCount} file${changedFileCount === 1 ? "" : "s"}`
}

export function formatServiceCount(serviceCount: number): string {
  return `${serviceCount} service${serviceCount === 1 ? "" : "s"}`
}

export function resolveFileStatusLabel(status: ChangedFile["status"]): string {
  if (status === "modified") {
    return "M"
  }
  if (status === "added") {
    return "A"
  }
  if (status === "deleted") {
    return "D"
  }
  if (status === "renamed") {
    return "R"
  }
  return "U"
}

export function resolveRepositoryInitials(repositoryName: string): string {
  const alphanumericName = repositoryName.replace(/[^a-zA-Z0-9]/g, "")
  return alphanumericName.slice(0, 2).toUpperCase()
}

export function resolveShortCommitHash(commitHash: string): string {
  return commitHash.slice(0, 7)
}
