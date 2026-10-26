import { http } from "../../../lib/http"
import type {
  ConfluenceDocument,
  ConfluenceDocumentsResponse,
  MigTargetsResult,
  QueryReviewApplyRequest,
  QueryReviewApplyResult,
  QueryReviewRepository,
  QueryReviewRequest,
  QueryReviewResult,
  SopApplyRequest,
  SopApplyResult,
  SopGenerateRequest,
  SopGenerateResult,
  UtDocsApplyRequest,
  UtDocsApplyResult,
  UtDocsRequest,
  UtDocsResult
} from "../types"

export async function fetchDocuments(): Promise<ConfluenceDocument[]> {
  const response = await http.get<ConfluenceDocumentsResponse | ConfluenceDocument[]>(
    "/api/confluence/documents"
  )
  if (Array.isArray(response)) {
    return response
  }
  if (response && Array.isArray(response.documents)) {
    return response.documents
  }
  return []
}

export async function fetchDocumentDetail(id: string): Promise<ConfluenceDocument> {
  const cleanId = encodeURIComponent(id.trim())
  return http.get<ConfluenceDocument>(`/api/confluence/documents/${cleanId}`)
}

export async function requestQueryReview(
  id: string,
  payload: QueryReviewRequest,
  signal?: AbortSignal
): Promise<QueryReviewResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.post<QueryReviewResult>(
    `/api/confluence/documents/${cleanId}/query-review`,
    payload,
    { signal }
  )
}

export async function applyQueryReview(
  id: string,
  repositories: QueryReviewRepository[]
): Promise<QueryReviewApplyResult> {
  const cleanId = encodeURIComponent(id.trim())
  const payload: QueryReviewApplyRequest = {
    repositories: repositories.map((repository) => ({
      repository: repository.repository,
      queries: repository.queries.map((query) => ({
        no: query.no,
        feature: query.feature,
        query_statement: query.query_statement,
        index: query.index,
        create_index: query.create_index
      }))
    }))
  }
  return http.post<QueryReviewApplyResult>(
    `/api/confluence/documents/${cleanId}/query-review/apply`,
    payload
  )
}

export async function requestUtDocs(
  id: string,
  payload: UtDocsRequest,
  signal?: AbortSignal
): Promise<UtDocsResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.post<UtDocsResult>(
    `/api/confluence/documents/${cleanId}/ut-docs`,
    payload,
    { signal }
  )
}

export async function fetchMigTargets(
  id: string,
  signal?: AbortSignal
): Promise<MigTargetsResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.get<MigTargetsResult>(
    `/api/confluence/documents/${cleanId}/mig-targets`,
    { signal }
  )
}

export async function applyUtDocs(
  id: string,
  payload: UtDocsApplyRequest,
  signal?: AbortSignal
): Promise<UtDocsApplyResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.post<UtDocsApplyResult>(
    `/api/confluence/documents/${cleanId}/ut-docs/apply`,
    payload,
    { signal }
  )
}

export async function requestSopGenerate(
  id: string,
  payload: SopGenerateRequest,
  signal?: AbortSignal
): Promise<SopGenerateResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.post<SopGenerateResult>(
    `/api/confluence/documents/${cleanId}/sop/generate`,
    payload,
    { signal }
  )
}

export async function applySop(
  id: string,
  payload: SopApplyRequest
): Promise<SopApplyResult> {
  const cleanId = encodeURIComponent(id.trim())
  return http.post<SopApplyResult>(
    `/api/confluence/documents/${cleanId}/sop/apply`,
    payload
  )
}
