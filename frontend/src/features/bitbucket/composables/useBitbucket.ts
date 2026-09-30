import { ref } from "vue"
import type {
  PushBranch,
  PullRequest,
  PRDiff,
  AICodeReview,
  CreatePRPayload,
} from "../types"
import {
  fetchPushes,
  fetchPullRequests,
  fetchPRDiff,
  createPullRequest,
  generateAIReview,
  postPRComment,
  applyReviewAction,
} from "../api/bitbucket-api"

export function useBitbucket() {
  const pushes = ref<PushBranch[]>([])
  const pullRequests = ref<PullRequest[]>([])
  const selectedDiff = ref<PRDiff | null>(null)
  const activeAIReview = ref<AICodeReview | null>(null)

  const pushFilter = ref("all")
  const prFilter = ref("all")

  const isLoadingPushes = ref(false)
  const isLoadingPRs = ref(false)
  const isLoadingDiff = ref(false)
  const isGeneratingAI = ref(false)
  const isSubmittingPR = ref(false)
  const isPostingComment = ref(false)

  const error = ref<string | null>(null)

  const loadPushes = async (filter = pushFilter.value) => {
    pushFilter.value = filter
    isLoadingPushes.value = true
    try {
      pushes.value = await fetchPushes(filter)
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load pushed branches"
    } finally {
      isLoadingPushes.value = false
    }
  }

  const loadPullRequests = async (filter = prFilter.value) => {
    prFilter.value = filter
    isLoadingPRs.value = true
    try {
      pullRequests.value = await fetchPullRequests(filter)
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load pull requests"
    } finally {
      isLoadingPRs.value = false
    }
  }

  const loadDiff = async (prId: string, repo: string) => {
    isLoadingDiff.value = true
    error.value = null
    try {
      selectedDiff.value = await fetchPRDiff(prId, repo)
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load diff"
    } finally {
      isLoadingDiff.value = false
    }
  }

  const requestAIReview = async (
    prId: string,
    repo: string,
    model?: string
  ): Promise<AICodeReview | null> => {
    isGeneratingAI.value = true
    error.value = null
    try {
      const review = await generateAIReview(prId, repo, model)
      activeAIReview.value = review
      return review
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to generate AI review"
      error.value = message
      throw err
    } finally {
      isGeneratingAI.value = false
    }
  }

  const submitPullRequest = async (payload: CreatePRPayload): Promise<PullRequest> => {
    isSubmittingPR.value = true
    error.value = null
    try {
      const created = await createPullRequest(payload)
      pullRequests.value = [created, ...pullRequests.value]
      pushes.value = pushes.value.filter(
        (p) => !(p.repo === payload.repo && p.branch === payload.source_branch)
      )
      return created
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create pull request"
      error.value = message
      throw err
    } finally {
      isSubmittingPR.value = false
    }
  }

  const sendPRComment = async (prId: string, repo: string, content: string) => {
    isPostingComment.value = true
    error.value = null
    try {
      return await postPRComment(prId, repo, content)
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to post comment"
      error.value = message
      throw err
    } finally {
      isPostingComment.value = false
    }
  }

  const executeReviewAction = async (prId: string, repo: string, action: string) => {
    try {
      await applyReviewAction(prId, repo, action)
      const found = pullRequests.value.find((p) => p.id === prId)
      if (found) {
        if (action === "approve") found.status = "approved"
        else if (action === "decline") found.status = "declined"
        else if (action === "needs-work") found.status = "needs_work"
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to update review status"
      throw err
    }
  }

  return {
    pushes,
    pullRequests,
    selectedDiff,
    activeAIReview,
    pushFilter,
    prFilter,
    isLoadingPushes,
    isLoadingPRs,
    isLoadingDiff,
    isGeneratingAI,
    isSubmittingPR,
    isPostingComment,
    error,
    loadPushes,
    loadPullRequests,
    loadDiff,
    requestAIReview,
    submitPullRequest,
    sendPRComment,
    executeReviewAction,
  }
}
