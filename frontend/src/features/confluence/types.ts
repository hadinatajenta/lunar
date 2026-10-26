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

export type QueryReviewErrorCode = "NO_COMPARE_LINK"

export interface QueryReviewSource {
  compare_url: string
  project_key?: string
  repo_slug?: string
  source_branch?: string
  target_branch?: string
}

export interface QueryReviewRequest {
  model: string
  source?: QueryReviewSource
}

export interface QueryReviewEntry {
  no: number
  feature: string
  repository: string
  table: string
  action_type: string
  query_statement: string
  index: string
  create_index: string
  file: string
  line: number
  notes: string
}

export interface QueryReviewAnalyzedFile {
  path: string
  added_lines: number
  language: string
}

export interface QueryReviewSkippedFile {
  path: string
  reason: string
}

export type QueryReviewRepositoryStatus = "success" | "no_queries" | "error"

export interface QueryReviewRepository {
  repository: string
  project_key: string
  source_branch: string
  target_branch: string
  compare_url: string
  status: QueryReviewRepositoryStatus
  analyzed_files: QueryReviewAnalyzedFile[]
  skipped_files: QueryReviewSkippedFile[]
  queries: QueryReviewEntry[]
  summary: string
  truncated: boolean
  error: string
}

export interface QueryReviewSkippedRepository {
  repository: string
  reason: string
}

export interface QueryReviewResult {
  document_id: string
  model: string
  prompt_version: string
  mig_page_id: string
  mig_page_title: string
  skipped_repositories: QueryReviewSkippedRepository[]
  repositories: QueryReviewRepository[]
}

export interface QueryReviewApplyQuery {
  no: number
  feature: string
  query_statement: string
  index: string
  create_index: string
}

export interface QueryReviewApplyRepository {
  repository: string
  queries: QueryReviewApplyQuery[]
}

export interface QueryReviewApplyRequest {
  repositories: QueryReviewApplyRepository[]
}

export interface QueryReviewApplyResult {
  page_id: string
  version: number
  url: string
}

export type UtRunPhase = "scan" | "safety" | "run" | "screenshot"

export type UtRunStatus = "passed" | "failed" | "blocked" | "not_run"

export type UtTestClassification = "safe" | "blocked"

export interface UtTestFile {
  path: string
  package: string
  language: string
  classification: UtTestClassification
  reason: string
}

export interface UtRunResult {
  status: UtRunStatus
  executed_packages: string[]
  total: number
  passed: number
  failed: number
  duration_ms: number
  output_excerpt: string
}

export type UtTestStatus = "passed" | "failed" | "skipped"

export interface UtTestResult {
  name: string
  package: string
  file: string
  status: UtTestStatus
  duration_ms: number
  output: string
}

export interface UtUntestedFunction {
  name: string
  file: string
  package: string
}

export interface UtRunDone {
  run_id: string
  repo: string
  branch: string
  base_ref: string
  changed_files: string[]
  test_files: UtTestFile[]
  blocked_files: UtTestFile[]
  run: UtRunResult
  stashed: boolean
  restored: boolean
  tests: UtTestResult[]
  untested_functions: UtUntestedFunction[]
  screenshot_url: string
}

export interface UtDocsRepositoryRun {
  repository: string
  branch: string
  base_ref: string
  test_files: UtTestFile[]
  blocked_files: UtTestFile[]
  run: UtRunResult
  tests: UtTestResult[]
  untested_functions: UtUntestedFunction[]
}

export interface UtDocsRequest {
  model: string
  repositories: UtDocsRepositoryRun[]
}

export interface UtDocsRow {
  no: number
  repository: string
  function: string
  scenario: string
  expected_result: string
  input_data: string
  result: string
  screen_capture: string
  test_function_name: string
}

export interface UtDocsMockTest {
  test_function_name: string
  function: string
  scenario: string
  expected_result: string
  input_data: string
  code: string
}

export interface UtDocsResult {
  document_id: string
  model: string
  prompt_version: string
  title: string
  summary: string
  table: string
  rows: UtDocsRow[]
  mock_tests: UtDocsMockTest[]
  markdown: string
  blocked_notice: string
  screenshot_caption: string
}

export interface UtDocsApplyRow {
  function: string
  scenario: string
  expected_result: string
  input_data: string
  result: string
}

export interface UtDocsApplyMockTest {
  function: string
  scenario: string
  expected_result: string
  input_data: string
  code: string
}

export interface UtDocsApplyScreenshot {
  filename: string
  content_base64: string
}

export interface UtDocsApplyRepository {
  repository: string
  screenshot: UtDocsApplyScreenshot
  rows: UtDocsApplyRow[]
  mock_tests: UtDocsApplyMockTest[]
}

export interface UtDocsApplyRequest {
  repositories: UtDocsApplyRepository[]
}

export interface UtDocsApplyResult {
  page_id: string
  version: number
  url: string
}

export interface MigTarget {
  repository: string
  branch: string
  base_ref?: string
}

export interface MigSkippedRepository {
  repository: string
  reason: string
}

export interface MigTargetsResult {
  mig_page_title: string
  targets: MigTarget[]
  skipped_repositories: MigSkippedRepository[]
}

export interface SopScanStatement {
  action: string
  table: string
  sql: string
}

export interface SopScanDatabaseFile {
  path: string
  statements: SopScanStatement[]
}

export interface SopScanTable {
  name: string
  action: string
}

export interface SopScanResult {
  repository: string
  branch: string
  base_ref: string
  database_files: SopScanDatabaseFile[]
  tables: SopScanTable[]
  warnings: string[]
}

export interface SopScanRequest {
  root: string
  repo: string
  branch: string
  base_ref?: string
}

export interface SopScanTarget {
  repository: string
  branch: string
  base_ref?: string
}

export interface SopGenerateRepository {
  repository: string
  branch: string
  base_ref?: string
  database_files: SopScanDatabaseFile[]
  warnings: string[]
}

export interface SopDeployPlan {
  title: string
  page_id: string
  exists: boolean
  managed_block: string
}

export interface SopOperasionalChild {
  title: string
  exists: boolean
}

export interface SopOperasionalPlan {
  title: string
  page_id: string
  exists: boolean
  children: SopOperasionalChild[]
}

export interface SopGenerateRequest {
  model: string
  repositories?: SopScanResult[]
}

export interface SopGenerateResult {
  issue_key: string
  development_page_id: string
  repositories: SopGenerateRepository[]
  deploy: SopDeployPlan
  operasional: SopOperasionalPlan
  warnings: string[]
}

export type SopDeployMode = "create" | "update"

export interface SopApplyRequest {
  deploy: { mode: SopDeployMode; managed_block: string }
  operasional: { create_children: boolean }
}

export interface SopApplyPage {
  id: string
  version: number
  url: string
}

export interface SopApplyDeployPage extends SopApplyPage {
  action: string
}

export interface SopApplyResult {
  deploy_page: SopApplyDeployPage
  operasional_page: SopApplyPage
  created_children: string[]
}
