export type ConfluenceDocType = "ut" | "query" | "sop" | "doc"

export type ConfluenceDocTypeLabel = "UT" | "QR" | "SOP" | "DOC"

export type ConfluenceDocStatus = "open" | "progress" | "done"

export type DocTypeFilter = "all" | "ut" | "query" | "sop"

export interface ConfluenceDocument {
  id: string
  type: ConfluenceDocType
  type_label: ConfluenceDocTypeLabel
  title: string
  status: ConfluenceDocStatus
  owner: string
  last_editor?: string
  updated: string
  space: string
  description: string
  body?: string
  url: string
}

export interface ConfluenceDocumentsResponse {
  documents: ConfluenceDocument[]
  total: number
  source?: string
  degraded?: boolean
  degraded_reason?: string
}

export interface StatsCounts {
  all: number
  ut: number
  query: number
  sop: number
}
