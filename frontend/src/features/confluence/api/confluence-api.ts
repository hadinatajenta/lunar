import { http } from "../../../lib/http"
import type { ConfluenceDocument, ConfluenceDocumentsResponse } from "../types"

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
