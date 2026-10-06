import type { BaseBranchOption } from "../branch-types"

export interface BaseBranchGroup {
  label: string
  options: BaseBranchOption[]
}

export const OTHER_TEAMS_GROUP_LABEL = "Other teams' branches"

export const isProductionLikeOption = (option: BaseBranchOption): boolean => option.source !== "other"

export const filterBaseBranchOptions = (
  options: BaseBranchOption[],
  searchTerm: string
): BaseBranchOption[] => {
  const query = searchTerm.trim().toLowerCase()
  if (query.length === 0) {
    return options
  }
  return options.filter((option) => option.name.toLowerCase().includes(query))
}

export const baseBranchGroups = (
  options: BaseBranchOption[],
  searchTerm: string
): BaseBranchGroup[] => {
  const matching = filterBaseBranchOptions(options, searchTerm)
  const productionLike = matching.filter(isProductionLikeOption)
  const otherTeams = matching.filter((option) => !isProductionLikeOption(option))
  const groups: BaseBranchGroup[] = []
  if (productionLike.length > 0) {
    groups.push({ label: "", options: productionLike })
  }
  if (otherTeams.length > 0) {
    groups.push({ label: OTHER_TEAMS_GROUP_LABEL, options: otherTeams })
  }
  return groups
}

export const emptyBaseBranchMessage = (repo: string, searchTerm: string, optionCount: number): string => {
  if (optionCount === 0) {
    return `No branch is available in ${repo}.`
  }
  return `No branch in ${repo} matches "${searchTerm}"`
}
