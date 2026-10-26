import type { QueryReviewRepository } from "../types"

const CAPTURE_HEADERS = [
  "No",
  "Fitur",
  "Query Statement",
  "Index",
  "Capture Explain / Execution Plan"
]

const GROUP_ROW_COLSPAN = 5

export const NO_QUERY_CHANGES_MESSAGE =
  "Tidak ada penambahan atau perubahan query pada pengembangan kali ini."

const escapeHtml = (value: string): string =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")

const selectRepositoriesWithQueries = (
  repositories: QueryReviewRepository[]
): QueryReviewRepository[] => repositories.filter((repository) => repository.queries.length > 0)

const renderGroupRow = (repositoryName: string): string =>
  `<tr><td colspan="${GROUP_ROW_COLSPAN}"><strong>Layanan / Microservice: <code>${escapeHtml(
    repositoryName
  )}</code></strong></td></tr>`

const selectIndexParts = (index: string, createIndex: string): string[] =>
  [index, createIndex].filter((part) => part.length > 0)

const renderIndexCell = (index: string, createIndex: string): string =>
  selectIndexParts(index, createIndex).map(escapeHtml).join("<br>")

const renderQueryRow = (
  no: number,
  feature: string,
  queryStatement: string,
  index: string,
  createIndex: string
): string =>
  `<tr><td>${escapeHtml(String(no))}</td><td>${escapeHtml(
    feature
  )}</td><td>${escapeHtml(queryStatement)}</td><td>${renderIndexCell(
    index,
    createIndex
  )}</td><td></td></tr>`

export function buildCaptureTableHtml(repositories: QueryReviewRepository[]): string {
  const groupedRepositories = selectRepositoriesWithQueries(repositories)
  const hasGroups = groupedRepositories.length > 1

  const head = `<thead><tr>${CAPTURE_HEADERS.map(
    (header) => `<th>${escapeHtml(header)}</th>`
  ).join("")}</tr></thead>`

  if (groupedRepositories.length === 0) {
    const emptyRow = `<tr><td colspan="${GROUP_ROW_COLSPAN}">${escapeHtml(
      NO_QUERY_CHANGES_MESSAGE
    )}</td></tr>`
    return `<table class="wrapped relative-table" style="width: 100%;">${head}<tbody>${emptyRow}</tbody></table>`
  }

  const body = groupedRepositories
    .map((repository) => {
      const rows: string[] = []
      if (hasGroups) {
        rows.push(renderGroupRow(repository.repository))
      }
      for (const entry of repository.queries) {
        rows.push(
          renderQueryRow(
            entry.no,
            entry.feature,
            entry.query_statement,
            entry.index,
            entry.create_index
          )
        )
      }
      return rows.join("")
    })
    .join("")

  return `<table class="wrapped relative-table" style="width: 100%;">${head}<tbody>${body}</tbody></table>`
}

export function buildCaptureTableText(repositories: QueryReviewRepository[]): string {
  const groupedRepositories = selectRepositoriesWithQueries(repositories)
  const hasGroups = groupedRepositories.length > 1

  if (groupedRepositories.length === 0) {
    return [CAPTURE_HEADERS.join("\t"), NO_QUERY_CHANGES_MESSAGE].join("\n")
  }

  const lines: string[] = [CAPTURE_HEADERS.join("\t")]
  for (const repository of groupedRepositories) {
    if (hasGroups) {
      lines.push(`Layanan / Microservice: ${repository.repository}`)
    }
    for (const entry of repository.queries) {
      lines.push(
        [
          String(entry.no),
          entry.feature,
          entry.query_statement,
          selectIndexParts(entry.index, entry.create_index).join("\n"),
          ""
        ].join("\t")
      )
    }
  }
  return lines.join("\n")
}
