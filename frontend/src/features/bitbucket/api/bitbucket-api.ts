import { http } from "../../../lib/http"
import type {
  PushBranch,
  PullRequest,
  PRDiff,
  AICodeReview,
  PRComment,
  CreatePRPayload,
} from "../types"

export async function fetchPushes(filter = "all"): Promise<PushBranch[]> {
  return http.get<PushBranch[]>("/api/bitbucket/pushes", {
    params: { filter },
  })
}

export async function fetchPullRequests(filter = "all"): Promise<PullRequest[]> {
  return http.get<PullRequest[]>("/api/bitbucket/prs", {
    params: { filter },
  })
}

export async function fetchPRDiff(prId: string, repo: string): Promise<PRDiff> {
  const cleanId = encodeURIComponent(prId.trim())
  return http.get<PRDiff>(`/api/bitbucket/prs/${cleanId}/diff`, {
    params: { repo },
  })
}

export async function createPullRequest(payload: CreatePRPayload): Promise<PullRequest> {
  return http.post<PullRequest>("/api/bitbucket/prs", payload)
}

export async function generateAIReview(prId: string, repo: string): Promise<AICodeReview> {
  const cleanId = encodeURIComponent(prId.trim())
  return http.post<AICodeReview>(`/api/bitbucket/prs/${cleanId}/ai-review`, null, {
    params: { repo },
  })
}

export async function postPRComment(
  prId: string,
  repo: string,
  content: string
): Promise<PRComment> {
  const cleanId = encodeURIComponent(prId.trim())
  return http.post<PRComment>(
    `/api/bitbucket/prs/${cleanId}/comments`,
    { content },
    { params: { repo } }
  )
}

export async function applyReviewAction(
  prId: string,
  repo: string,
  action: string
): Promise<{ status: string; action: string; pr_id: string }> {
  const cleanId = encodeURIComponent(prId.trim())
  return http.post<{ status: string; action: string; pr_id: string }>(
    `/api/bitbucket/prs/${cleanId}/action`,
    { action },
    { params: { repo } }
  )
}
