import type { UtDocsMockTest, UtDocsRow } from "../types"

export const UT_SCENARIO_HEADERS = [
  "No",
  "Function",
  "Scenario",
  "Expected Result",
  "Input Data",
  "Result",
  "Screen Capture"
]

export const UT_SCENARIO_GROUP_COLSPAN = 7

export const NO_UT_SCENARIO_MESSAGE = "No test function was reported for this run."

export type UtResultTone = "passed" | "failed" | "skipped" | "not_run" | "unknown"

const RESULT_TONES: Record<string, UtResultTone> = {
  passed: "passed",
  pass: "passed",
  failed: "failed",
  fail: "failed",
  skipped: "skipped",
  skip: "skipped",
  "not run": "not_run",
  not_run: "not_run"
}

const RESULT_LABELS: Record<UtResultTone, string> = {
  passed: "Passed",
  failed: "Failed",
  skipped: "Skipped",
  not_run: "Not Run",
  unknown: "Unknown"
}

const EXECUTED_TONES: UtResultTone[] = ["passed", "failed", "skipped"]

export interface UtScenarioGroup {
  repository: string
  rows: UtDocsRow[]
}

export function resolveUtResultTone(result: string): UtResultTone {
  return RESULT_TONES[result.trim().toLowerCase()] ?? "unknown"
}

export function formatUtResultLabel(result: string): string {
  const tone = resolveUtResultTone(result)
  if (tone === "unknown") {
    const trimmed = result.trim()
    return trimmed.length > 0 ? trimmed : RESULT_LABELS.unknown
  }
  return RESULT_LABELS[tone]
}

export function hasExecutedUtRows(rows: UtDocsRow[]): boolean {
  return rows.some((row) => EXECUTED_TONES.includes(resolveUtResultTone(row.result)))
}

export function groupUtScenarioRows(rows: UtDocsRow[]): UtScenarioGroup[] {
  const groups: UtScenarioGroup[] = []
  const groupIndex = new Map<string, UtScenarioGroup>()
  for (const row of rows) {
    const existing = groupIndex.get(row.repository)
    if (existing !== undefined) {
      existing.rows.push(row)
      continue
    }
    const group: UtScenarioGroup = { repository: row.repository, rows: [row] }
    groupIndex.set(row.repository, group)
    groups.push(group)
  }
  return groups
}

const escapeHtml = (value: string): string =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")

const renderGroupRow = (repositoryName: string): string =>
  `<tr><td colspan="${UT_SCENARIO_GROUP_COLSPAN}"><strong>Layanan / Microservice: <code>${escapeHtml(
    repositoryName
  )}</code></strong></td></tr>`

const renderScenarioRow = (row: UtDocsRow): string =>
  `<tr><td>${escapeHtml(String(row.no))}</td><td>${escapeHtml(
    row.function
  )}</td><td>${escapeHtml(row.scenario)}</td><td>${escapeHtml(
    row.expected_result
  )}</td><td>${escapeHtml(row.input_data)}</td><td>${escapeHtml(
    formatUtResultLabel(row.result)
  )}</td><td>${escapeHtml(row.screen_capture)}</td></tr>`

export function buildUtMockTestMarkdown(mockTests: UtDocsMockTest[]): string {
  return mockTests
    .filter((mockTest) => mockTest.code.trim().length > 0)
    .map((mockTest) => `\`\`\`go\n${mockTest.code.trimEnd()}\n\`\`\``)
    .join("\n\n")
}

const renderMockTestHtml = (mockTest: UtDocsMockTest): string =>
  `<pre><code class="language-go">${escapeHtml(mockTest.code)}</code></pre>`

const renderMockTestText = (mockTest: UtDocsMockTest): string =>
  `\`\`\`go\n${mockTest.code.trimEnd()}\n\`\`\``

export function buildUtScenarioTableHtml(
  rows: UtDocsRow[],
  mockTests: UtDocsMockTest[] = []
): string {
  const groups = groupUtScenarioRows(rows)
  const hasGroups = groups.length > 1
  const mockTestsHtml = mockTests.map(renderMockTestHtml).join("")

  const head = `<thead><tr>${UT_SCENARIO_HEADERS.map(
    (header) => `<th>${escapeHtml(header)}</th>`
  ).join("")}</tr></thead>`

  if (groups.length === 0) {
    const emptyRow = `<tr><td colspan="${UT_SCENARIO_GROUP_COLSPAN}">${escapeHtml(
      NO_UT_SCENARIO_MESSAGE
    )}</td></tr>`
    return `<table class="wrapped relative-table" style="width: 100%;">${head}<tbody>${emptyRow}</tbody></table>${mockTestsHtml}`
  }

  const body = groups
    .map((group) => {
      const rows: string[] = []
      if (hasGroups) {
        rows.push(renderGroupRow(group.repository))
      }
      for (const row of group.rows) {
        rows.push(renderScenarioRow(row))
      }
      return rows.join("")
    })
    .join("")

  return `<table class="wrapped relative-table" style="width: 100%;">${head}<tbody>${body}</tbody></table>${mockTestsHtml}`
}

export function buildUtScenarioTableText(
  rows: UtDocsRow[],
  mockTests: UtDocsMockTest[] = []
): string {
  const groups = groupUtScenarioRows(rows)
  const hasGroups = groups.length > 1

  if (groups.length === 0) {
    return [
      UT_SCENARIO_HEADERS.join("\t"),
      NO_UT_SCENARIO_MESSAGE,
      ...mockTests.map(renderMockTestText)
    ].join("\n")
  }

  const lines: string[] = [UT_SCENARIO_HEADERS.join("\t")]
  for (const group of groups) {
    if (hasGroups) {
      lines.push(`Layanan / Microservice: ${group.repository}`)
    }
    for (const row of group.rows) {
      lines.push(
        [
          String(row.no),
          row.function,
          row.scenario,
          row.expected_result,
          row.input_data,
          formatUtResultLabel(row.result),
          row.screen_capture
        ].join("\t")
      )
    }
  }
  return [lines.join("\n"), ...mockTests.map(renderMockTestText)].join("\n\n")
}
