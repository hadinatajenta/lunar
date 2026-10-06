import { computed, ref } from "vue"
import type { CreateBranchesRequest, RepositoryFeedbackAction } from "../branch-types"
import { createBranches, recordRepositoryFeedback } from "../api/jira-branches-api"
import { describeRequestError } from "../logic/error-message"
import {
  addManualTarget,
  applyCreationResponse,
  createBranchPlan,
  removeTarget,
  type BranchPlanState
} from "../logic/branch-plan"
import {
  creatableTargets,
  planProgress,
  type PlanProgress
} from "../logic/branch-plan-progress"

const MISSING_BASE_MESSAGE = "Choose a base branch for every repository you keep"
const MISSING_NAME_MESSAGE = "The branch name is required before a branch can be created"
const CREATE_REQUEST_FAILED_MESSAGE = "The branch creation request failed"
const FEEDBACK_FAILED_MESSAGE = "Your correction could not be recorded"

const plans = ref<Record<string, BranchPlanState>>({})
const createErrorByIssue = ref<Record<string, string>>({})
const isCreatingByIssue = ref<Record<string, boolean>>({})
const feedbackError = ref("")

export function useBranchCreation() {
  const hasPlan = (issueKey: string): boolean => plans.value[issueKey] !== undefined

  const planFor = (issueKey: string): BranchPlanState => {
    if (plans.value[issueKey] === undefined) {
      plans.value[issueKey] = createBranchPlan(issueKey)
    }
    return plans.value[issueKey]
  }

  const progressFor = (issueKey: string): PlanProgress | null => {
    const plan = plans.value[issueKey]
    return plan === undefined ? null : planProgress(plan)
  }

  const createErrorFor = (issueKey: string): string => createErrorByIssue.value[issueKey] ?? ""
  const isCreatingFor = (issueKey: string): boolean => isCreatingByIssue.value[issueKey] === true

  const createFor = async (issueKey: string, repos?: string[]): Promise<boolean> => {
    const plan = planFor(issueKey)
    const targets = creatableTargets(plan).filter(
      (target) => repos === undefined || repos.includes(target.repo)
    )
    if (targets.length === 0) {
      return false
    }
    if (plan.branchName.trim().length === 0) {
      createErrorByIssue.value = { ...createErrorByIssue.value, [issueKey]: MISSING_NAME_MESSAGE }
      return false
    }
    const withoutBase = targets.filter((target) => (plan.bases[target.repo] ?? "").length === 0)
    if (withoutBase.length > 0) {
      createErrorByIssue.value = { ...createErrorByIssue.value, [issueKey]: MISSING_BASE_MESSAGE }
      return false
    }

    const request: CreateBranchesRequest = {
      key: issueKey,
      branch: plan.branchName,
      targets: targets.map((target) => ({
        repo: target.repo,
        base_branch: plan.bases[target.repo],
        remember_base: plan.rememberBases[target.repo] === true
      }))
    }

    isCreatingByIssue.value = { ...isCreatingByIssue.value, [issueKey]: true }
    createErrorByIssue.value = { ...createErrorByIssue.value, [issueKey]: "" }
    try {
      const response = await createBranches(request)
      applyCreationResponse(plan, response)
      return true
    } catch (error: unknown) {
      createErrorByIssue.value = {
        ...createErrorByIssue.value,
        [issueKey]: describeRequestError(error, CREATE_REQUEST_FAILED_MESSAGE)
      }
      return false
    } finally {
      isCreatingByIssue.value = { ...isCreatingByIssue.value, [issueKey]: false }
    }
  }

  const retryTarget = (issueKey: string, repo: string): Promise<boolean> => createFor(issueKey, [repo])

  const recordFeedback = async (
    issueKey: string,
    repo: string,
    action: RepositoryFeedbackAction
  ): Promise<void> => {
    try {
      await recordRepositoryFeedback(issueKey, repo, action)
      feedbackError.value = ""
    } catch (error: unknown) {
      feedbackError.value = describeRequestError(error, FEEDBACK_FAILED_MESSAGE)
    }
  }

  const addRepository = async (issueKey: string, repo: string, isCloned: boolean): Promise<void> => {
    addManualTarget(planFor(issueKey), repo, isCloned)
    await recordFeedback(issueKey, repo, "added")
  }

  const removeRepository = async (issueKey: string, repo: string): Promise<void> => {
    removeTarget(planFor(issueKey), repo)
    await recordFeedback(issueKey, repo, "removed")
  }

  const clearCreateError = (issueKey: string): void => {
    createErrorByIssue.value = { ...createErrorByIssue.value, [issueKey]: "" }
  }

  return {
    plans: computed(() => plans.value),
    feedbackError: computed(() => feedbackError.value),
    planFor,
    hasPlan,
    progressFor,
    createErrorFor,
    isCreatingFor,
    createFor,
    retryTarget,
    addRepository,
    removeRepository,
    recordFeedback,
    clearCreateError
  }
}
