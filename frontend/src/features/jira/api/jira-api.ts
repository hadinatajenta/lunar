import { http } from "../../../lib/http"
import type { JiraIssue, JiraBacklogResponse } from "../types"

export interface JiraRemoteLink {
  id?: string | number
  relationship?: string
  object: {
    url: string
    title: string
  }
}

export async function fetchMyIssues(options?: { refresh?: boolean }): Promise<JiraIssue[]> {
  const query = options?.refresh ? "?refresh=1" : ""
  const response = await http.get<JiraIssue[] | { issues: JiraIssue[] }>(`/api/jira/issues${query}`)
  if (Array.isArray(response)) {
    return response
  }
  if (response && Array.isArray(response.issues)) {
    return response.issues
  }
  return []
}

export async function fetchBacklog(
  squad?: string,
  options?: { refresh?: boolean }
): Promise<JiraBacklogResponse> {
  const queryParts: string[] = []
  if (squad) {
    queryParts.push(`squad=${encodeURIComponent(squad)}`)
  }
  if (options?.refresh) {
    queryParts.push("refresh=1")
  }
  const query = queryParts.length > 0 ? `?${queryParts.join("&")}` : ""
  return http.get<JiraBacklogResponse>(`/api/jira/backlog${query}`)
}

export async function fetchIssueDetail(key: string): Promise<JiraIssue> {
  const cleanKey = encodeURIComponent(key.trim())
  return http.get<JiraIssue>(`/api/jira/issues/${cleanKey}`)
}

export async function fetchIssueRemoteLinks(issueKey: string): Promise<JiraRemoteLink[]> {
  const cleanKey = encodeURIComponent(issueKey.trim())
  const response = await http.get<JiraRemoteLink[] | { remotelinks?: JiraRemoteLink[]; remote_links?: JiraRemoteLink[] }>(
    `/api/jira/issues/remotelinks?key=${cleanKey}`
  )
  if (Array.isArray(response)) {
    return response
  }
  if (response && Array.isArray(response.remotelinks)) {
    return response.remotelinks
  }
  if (response && Array.isArray(response.remote_links)) {
    return response.remote_links
  }
  return []
}
