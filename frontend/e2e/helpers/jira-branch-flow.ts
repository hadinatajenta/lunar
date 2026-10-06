import { expect, type Page, type Route } from "@playwright/test"
import { routeStandardJira, routeSecrets, routeSystemConfig } from "./route-mock"

export const ISSUE_KEY = "CRMMS-77911"
export const BRANCH_NAME = "feature/CRMMS-77911-Enhancement-Manage-User"
export const AUTH_TOKEN_KEY = "lunar_auth_token"

const SCOPE = {
  local_repository_count: 36,
  remote_repository_count: 65,
  coverage_percent: 55,
  verified_remotely: true,
  remote_repositories_checked: 5
}

interface MatchInput {
  repo: string
  branch: string
  source: "local" | "remote" | "both"
  lastCommitRelative?: string | null
}

export const match = (input: MatchInput) => ({
  repo: input.repo,
  branch: input.branch,
  source: input.source,
  is_current: false,
  is_cloned: input.source !== "remote",
  bitbucket_project: "NME",
  last_commit: "6a1f0c9d2b7e4f8a1c3d5e7f9a0b2c4d6e8f0a1b",
  last_commit_relative: input.lastCommitRelative ?? (input.source === "remote" ? null : "3 days ago"),
  ahead: input.source === "remote" ? null : 4,
  behind: input.source === "remote" ? null : 0
})

interface SuggestionInput {
  repo: string
  isCloned: boolean
  reason: string
}

export const suggestion = (input: SuggestionInput) => ({
  repo: input.repo,
  role: "backend",
  score: 9,
  is_cloned: input.isCloned,
  bitbucket_project: "NME",
  reasons: [{ signal: "epic_siblings", weight: 5, strength: 0.8, text: input.reason }]
})

const baseOptions = (repo: string, isCloned: boolean) => ({
  repo,
  is_cloned: isCloned,
  remembered: null,
  bitbucket_default: "release/production",
  options: [
    { name: "release/production", last_commit_relative: "23 Sep 2026", source: "bitbucket_default" },
    { name: "pre/development", last_commit_relative: "2 Oct 2026", source: "trusted" },
    {
      name: "feature/CRMMS-77745-Input-MDR-QRIS-Statis",
      last_commit_relative: "1 Oct 2026",
      source: "other"
    }
  ],
  other_branch_count: 0
})

interface FlowMocks {
  matches: ReturnType<typeof match>[]
  suggestions: ReturnType<typeof suggestion>[]
  unavailable?: { signal: string; reason: string }[]
  created?: unknown
}

export interface CreateRequest {
  branch: string
  targets: { repo: string; base_branch: string; remember_base: boolean }[]
}

export interface BranchFlow {
  createRequests: CreateRequest[]
}

export async function routeBranchFlow(page: Page, mocks: FlowMocks): Promise<BranchFlow> {
  const flow: BranchFlow = { createRequests: [] }
  await routeSecrets(page, { hasJiraPat: true })
  await routeSystemConfig(page)
  await routeStandardJira(page)

  const scope = SCOPE
  const suggestionsResponse = {
    key: ISSUE_KEY,
    confidence: mocks.suggestions.length > 0 ? "high" : "low",
    suggestions: mocks.suggestions,
    catalogue: { local_repository_count: 36, remote_repository_count: 65 },
    signals_unavailable: mocks.unavailable ?? [],
    scope
  }

  await page.route(/\/api\/jira\/issues\/[^/]+\/branches/, async (route) => {
    await route.fulfill({
      status: 200,
      json: { key: ISSUE_KEY, matches: mocks.matches, scope }
    })
  })

  await page.route(/\/api\/jira\/issues\/[^/]+\/repository-suggestions/, async (route) => {
    await route.fulfill({ status: 200, json: suggestionsResponse })
  })

  await page.route(/\/api\/jira\/issues\/[^/]+\/branch-name/, async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        key: ISSUE_KEY,
        branch: BRANCH_NAME,
        category: "Enhancement",
        core: "Manage-User",
        source: "model",
        evidence: "Halaman Utama Manage User (Grouping by PN)"
      }
    })
  })

  await page.route(/\/api\/jira\/issues\/[^/]+\/repository-feedback/, async (route) => {
    await route.fulfill({ status: 202, json: { recorded: true } })
  })

  await page.route(/\/api\/jira\/sprint\/branches/, async (route) => {
    await route.fulfill({
      status: 200,
      json: { matches: { [ISSUE_KEY]: mocks.matches }, scope }
    })
  })

  await page.route(/\/api\/jira\/repositories/, async (route) => {
    const search = new URL(route.request().url()).searchParams.get("search") ?? ""
    const repositories = [
      { repo: "atlas", project: "NME", is_cloned: true },
      { repo: "aurora", project: "NME", is_cloned: false },
      { repo: "micro-cm-qris", project: "NME", is_cloned: true }
    ].filter((entry) => entry.repo.includes(search))
    await route.fulfill({
      status: 200,
      json: { repositories, local_repository_count: 36, remote_repository_count: 65 }
    })
  })

  await page.route(/\/api\/workspace\/repos\/[^/]+\/base-branch-options/, async (route) => {
    const path = new URL(route.request().url()).pathname
    const repo = decodeURIComponent(path.split("/repos/")[1].replace("/base-branch-options", ""))
    await route.fulfill({ status: 200, json: baseOptions(repo, repo !== "fe-mms") })
  })

  await page.route(/\/api\/workspace\/branches/, async (route: Route) => {
    flow.createRequests.push(route.request().postDataJSON() as CreateRequest)
    await route.fulfill({
      status: 207,
      json: mocks.created ?? { key: ISSUE_KEY, branch: BRANCH_NAME, created: [], failed: [], skipped: [] }
    })
  })

  return flow
}

export async function openIssue(page: Page): Promise<void> {
  await page.getByTestId("kanban-card-CRMMS-77911").click()
  await expect(page.getByTestId("issue-detail-modal")).toBeVisible()
  await expect(page.getByTestId("issue-branch-panel")).toBeVisible()
}

export async function chooseBase(page: Page, repo: string, base: string): Promise<void> {
  const row = page.getByTestId(`branch-target-${repo}`)
  await row.getByRole("button", { name: new RegExp(`^Base branch for ${repo}`) }).click()
  await page.getByTestId(`base-branch-option-${base}`).click()
}
