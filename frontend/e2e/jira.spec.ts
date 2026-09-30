import { test, expect, type Page } from "@playwright/test"
import path from "path"
import { JiraPage } from "./pages/JiraPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/jira/evidence"

const mockAssignedIssues = [
  {
    id: "UT-101",
    key: "UT-101",
    kind: "ut",
    label: "UT",
    title: "Auth middleware unit tests",
    status: "open",
    priority: "low",
    points: 3,
    sub: "Auth · v2 middleware",
    description: "Unit test suite covering token parsing, refresh handling, and error paths for the v2 auth middleware."
  },
  {
    id: "UT-102",
    key: "UT-102",
    kind: "ut",
    label: "UT",
    title: "Payment gateway integration tests",
    status: "open",
    priority: "medium",
    points: 5,
    sub: "Payments · sandbox",
    description: "Integration tests against the sandbox payment gateway, covering successful charges and declines."
  },
  {
    id: "UT-103",
    key: "UT-103",
    kind: "ut",
    label: "UT",
    title: "Notification service tests",
    status: "open",
    priority: "low",
    points: 2,
    sub: "Notifications · templating",
    description: "Tests for notification templating, deduplication, and delivery retries."
  },
  {
    id: "QR-201",
    key: "QR-201",
    kind: "query",
    label: "QR",
    title: "Monthly transaction reconciliation query",
    status: "open",
    priority: "medium",
    points: 2,
    sub: "Reconciliation · monthly close",
    description: "SQL query used to reconcile transaction records against the ledger for monthly close."
  },
  {
    id: "QR-202",
    key: "QR-202",
    kind: "query",
    label: "QR",
    title: "User activity audit query",
    status: "open",
    priority: "high",
    points: 3,
    sub: "Audit · compliance",
    description: "Audit query listing user activity over the last 30 days for compliance verification."
  },
  {
    id: "SOP-301",
    key: "SOP-301",
    kind: "sop",
    label: "SOP",
    title: "Monitoring — Daily health check",
    status: "open",
    priority: "medium",
    points: 1,
    sub: "Monitoring · daily",
    description: "Step-by-step procedure for the daily health check of production services."
  },
  {
    id: "SOP-302",
    key: "SOP-302",
    kind: "sop",
    label: "SOP",
    title: "Monitoring — Alert escalation matrix",
    status: "open",
    priority: "high",
    points: 2,
    sub: "Monitoring · escalation",
    description: "Escalation matrix and paging rules for production alerts and on-call response."
  },
  {
    id: "CRMMS-77911",
    key: "CRMMS-77911",
    kind: "ut",
    label: "UT",
    title: "Sebagai Administrator, I want to melihat daftar user yang dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman detail, serta melihat riwayat perubahannya",
    status: "progress",
    priority: "high",
    points: 5,
    sub: "Core · user management",
    description: "Sebagai Administrator sistem MMS BRI, saya membutuhkan halaman komprehensif untuk memantau data seluruh user berdasarkan PN mereka. Detail mencakup pengelolaan hak akses per modul, histori login dan aktivitas 30 hari terakhir, serta approval workflow untuk perubahan level/jabatan secara terpusat.",
    generated: "describe('CRMMS-77911 user management', () => {\n  it('groups users by personal number', () => {});\n  it('updates role and permissions atomically', () => {});\n});"
  },
  {
    id: "QR-203",
    key: "QR-203",
    kind: "query",
    label: "QR",
    title: "Duplicate transaction detection query",
    status: "progress",
    priority: "medium",
    points: 2,
    sub: "Ops · daily cleanup",
    description: "Query detecting duplicate transactions by hash within a 24-hour window."
  },
  {
    id: "SOP-303",
    key: "SOP-303",
    kind: "sop",
    label: "SOP",
    title: "Operational — Deploy checklist",
    status: "done",
    priority: "low",
    points: 1,
    sub: "Operational · deploy",
    description: "Pre- and post-deploy checklist covering migrations, smoke tests, and rollback triggers."
  },
  {
    id: "BUG-201",
    key: "BUG-201",
    kind: "bug",
    label: "BUG",
    title: "Checkout fails when cart has an expired item",
    status: "open",
    priority: "high",
    points: 3,
    sub: "QA · Sinta · 2h ago",
    description: "Checkout flow fails abruptly when one of the cart items has passed its expiration timestamp."
  },
  {
    id: "BUG-202",
    key: "BUG-202",
    kind: "bug",
    label: "BUG",
    title: "Session timeout does not refresh the auth token",
    status: "progress",
    priority: "high",
    points: 3,
    sub: "QA · Dimas · 4h ago",
    description: "When user session times out after 15 minutes of inactivity, API requests fail with 401 instead of silent refresh."
  },
  {
    id: "SUB-101",
    key: "SUB-101",
    kind: "subtask",
    label: "SUB",
    title: "Write unit tests for auth middleware",
    status: "open",
    priority: "medium",
    points: 2,
    sub: "Parent · LUN-501",
    description: "Add comprehensive unit test coverage for token verification and claims validation."
  },
  {
    id: "SUB-102",
    key: "SUB-102",
    kind: "subtask",
    label: "SUB",
    title: "Update API docs for v2 endpoints",
    status: "done",
    priority: "low",
    points: 1,
    sub: "Parent · LUN-501",
    description: "Document breaking changes and new query parameters in REST API v2."
  }
]

const mockAssignedIssuesWithoutSop = mockAssignedIssues.filter(
  (issue) => issue.kind !== "sop"
)

const mockBacklogResponse = {
  sprints: [
    {
      id: "sprint-44",
      name: "Sprint 44 - Fortune Squad",
      is_active: true,
      dates: "Sep 15 - Sep 29",
      issues: [
        {
          id: "CRMMS-77911",
          key: "CRMMS-77911",
          kind: "story",
          title: "Sebagai Administrator, I want to melihat daftar user yang dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman detail, serta melihat riwayat perubahannya",
          status: "progress",
          priority: "high",
          points: 5,
          description: "Sebagai Administrator sistem MMS BRI, saya membutuhkan halaman komprehensif untuk memantau data seluruh user berdasarkan PN mereka."
        }
      ]
    },
    {
      id: "sprint-45",
      name: "Sprint 45 - Next Phase",
      is_active: false,
      dates: "Sep 30 - Oct 14",
      issues: [
        {
          id: "CRMMS-8812",
          key: "CRMMS-8812",
          kind: "task",
          title: "Merchant transaction settlement batch job",
          status: "open",
          priority: "medium",
          points: 3,
          description: "Daily automated settlement batch processing for registered MMS merchants."
        }
      ]
    }
  ],
  total: 2
}

function routeSecrets(page: Page, overrides: { has_jira_pat?: boolean } = {}) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        json: {
          user_id: "test-user-id",
          has_jira_pat: overrides.has_jira_pat ?? true,
          jira_username: "developer",
          has_bitbucket_pat: true,
          bitbucket_username: "developer",
          has_confluence_pat: true,
          has_ai_keys: true,
          configured_ai_providers: ["deepseek"],
          updated_at: "Just now"
        }
      })
      return
    }
    await route.continue()
  })
}

function routeStandardJira(page: Page) {
  return page.route(/\/api\/jira\//, async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.includes("/issues/")) {
      const key = decodeURIComponent(url.pathname.split("/issues/")[1] || "")
      const found = mockAssignedIssues.find(
        (item) => item.key === key || item.id === key
      )
      if (found) {
        await route.fulfill({ status: 200, json: found })
        return
      }
      await route.fulfill({ status: 404, json: { error: "Issue not found" } })
      return
    }
    if (url.pathname.endsWith("/issues")) {
      await route.fulfill({ status: 200, json: mockAssignedIssues })
      return
    }
    if (url.pathname.endsWith("/backlog")) {
      await route.fulfill({ status: 200, json: mockBacklogResponse })
      return
    }
    await route.fulfill({ status: 404, json: { error: "Not found" } })
  })
}

test.describe("Jira BRI Negative Scenarios", () => {
  test("user cannot see board when Jira PAT is unconfigured", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: false })
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.noPatBanner).toBeVisible()
    await expect(jiraPage.noPatBanner).toContainText(
      "Jira Personal Access Token (PAT) is not configured yet"
    )
    await expect(jiraPage.kanbanCol("open")).not.toBeVisible()
    const settingsLink = jiraPage.noPatBanner.getByRole("link", {
      name: /Configure in Settings/
    })
    await expect(settingsLink).toBeVisible()
    await expect(settingsLink).toHaveAttribute("href", "/settings")
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "04_jira_negative_unconfigured.png")
    })
  })

  test("user can see 401 error banner when Jira PAT is invalid or expired", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await page.route(/\/api\/jira\//, async (route) => {
      await route.fulfill({
        status: 401,
        json: { error: "unauthorized: invalid or expired token" }
      })
    })
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.errorBanner).toBeVisible()
    await expect(jiraPage.errorBanner).toContainText(
      "Unauthorized access: invalid or expired Jira PAT"
    )
    const updateLink = jiraPage.errorBanner.getByRole("link", {
      name: /Update in Settings/
    })
    await expect(updateLink).toBeVisible()
    await expect(updateLink).toHaveAttribute("href", "/settings")
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "05_jira_negative_401.png")
    })
  })

  test("user can see 502 VPN error banner when upstream is unreachable", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await page.route(/\/api\/jira\//, async (route) => {
      await route.fulfill({
        status: 502,
        json: { error: "bad gateway: upstream unreachable" }
      })
    })
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.vpnBanner).toBeVisible()
    await expect(jiraPage.vpnBanner).toContainText(
      "Unable to reach Jira BRI API. Please ensure you are connected to the BRI VPN."
    )
    await expect(jiraPage.retryButton).toBeVisible()
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "06_jira_negative_502.png")
    })
  })

  test("user can see empty column state when no issues exist for a status", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await page.route(/\/api\/jira\/issues/, async (route) => {
      await route.fulfill({ status: 200, json: [] })
    })
    await page.route(/\/api\/jira\/backlog/, async (route) => {
      await route.fulfill({ status: 200, json: { sprints: [] } })
    })
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.emptyColumnState("open")).toBeVisible()
    await expect(jiraPage.emptyColumnState("open")).toHaveText("Nothing here")
    await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
    await expect(jiraPage.emptyColumnState("progress")).toHaveText("Nothing here")
    await expect(jiraPage.emptyColumnState("done")).toBeVisible()
    await expect(jiraPage.emptyColumnState("done")).toHaveText("Nothing here")
  })

  test("user can see empty state when subfilter yields zero matches", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await page.route(/\/api\/jira\/issues/, async (route) => {
      await route.fulfill({ status: 200, json: mockAssignedIssuesWithoutSop })
    })
    await page.route(/\/api\/jira\/backlog/, async (route) => {
      await route.fulfill({ status: 200, json: mockBacklogResponse })
    })
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.filterDocs.click()
    await jiraPage.docFilterSop.click()
    await expect(jiraPage.emptyColumnState("open")).toBeVisible()
    await expect(jiraPage.emptyColumnState("open")).toHaveText("Nothing here")
    await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
    await expect(jiraPage.emptyColumnState("progress")).toHaveText("Nothing here")
    await expect(jiraPage.emptyColumnState("done")).toBeVisible()
    await expect(jiraPage.emptyColumnState("done")).toHaveText("Nothing here")
  })
})

test.describe("Jira BRI Positive Kanban & Backlog", () => {
  test("user can view assigned issues Kanban board with default 5 items per column", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.kanbanCol("open")).toBeVisible()
    await expect(jiraPage.colCount("open")).toHaveText("7")
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    await expect(jiraPage.colCount("progress")).toHaveText("2")
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(2)
    await expect(jiraPage.colCount("done")).toHaveText("1")
    await expect(jiraPage.kanbanCards("done")).toHaveCount(1)
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "01_jira_board_overview.png"),
      fullPage: true
    })
  })

  test("user can click load more button to expand column when more than 5 cards exist", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    await expect(jiraPage.loadMoreButton("open")).toBeVisible()
    await expect(jiraPage.loadMoreButton("open")).toContainText("Load more (2 more)")
    await jiraPage.loadMoreButton("open").click()
    await expect(jiraPage.kanbanCards("open")).toHaveCount(7)
    await expect(jiraPage.loadMoreButton("open")).not.toBeVisible()
  })

  test("user can click card to open centered modal dialog", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.kanbanCards("open").first().click()
    await expect(jiraPage.issueModal).toBeVisible()
    await expect(jiraPage.issueModal).toHaveAttribute("role", "dialog")
    await expect(jiraPage.issueModal).toHaveAttribute("aria-modal", "true")
    await expect(jiraPage.modalScrim).toBeVisible()
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "02_jira_modal_open.png")
    })
  })

  test("user can verify modal contains Jira code, user story title, status, and priority", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await page.getByTestId("kanban-card-CRMMS-77911").click()
    await expect(jiraPage.issueModal).toBeVisible()
    await expect(jiraPage.modalKey).toHaveText("CRMMS-77911")
    await expect(jiraPage.modalTitle).toContainText("Sebagai Administrator")
    await expect(page.getByTestId("modal-issue-status")).toContainText("In Progress")
    await expect(page.getByTestId("modal-issue-priority")).toContainText("High")
    await expect(page.getByTestId("modal-issue-points")).toContainText("5 pts")
  })

  test("user can toggle read more and read less description button", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await page.getByTestId("kanban-card-CRMMS-77911").click()
    await expect(jiraPage.issueModal).toBeVisible()
    await expect(jiraPage.btnReadMore).toBeVisible()
    await expect(jiraPage.btnReadMore).toHaveText("Read more")
    await jiraPage.btnReadMore.click()
    await expect(jiraPage.btnReadMore).toHaveText("Read less")
    await jiraPage.btnReadMore.click()
    await expect(jiraPage.btnReadMore).toHaveText("Read more")
  })

  test("user can close modal via close button", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.kanbanCards("open").first().click()
    await expect(jiraPage.issueModal).toBeVisible()
    await jiraPage.btnCloseModal.click()
    await expect(jiraPage.issueModal).not.toBeVisible()
  })

  test("user can close modal via backdrop click", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.kanbanCards("open").first().click()
    await expect(jiraPage.issueModal).toBeVisible()
    await jiraPage.modalScrim.click({ position: { x: 10, y: 10 } })
    await expect(jiraPage.issueModal).not.toBeVisible()
  })

  test("user can close modal via escape key", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.kanbanCards("open").first().click()
    await expect(jiraPage.issueModal).toBeVisible()
    await page.keyboard.press("Escape")
    await expect(jiraPage.issueModal).not.toBeVisible()
  })

  test("user can filter by category", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.filterBugs.click()
    await expect(jiraPage.filterBugs).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(1)
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
    await expect(jiraPage.emptyColumnState("done")).toBeVisible()
    await jiraPage.filterSubtasks.click()
    await expect(jiraPage.filterSubtasks).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(1)
    await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
    await expect(jiraPage.kanbanCards("done")).toHaveCount(1)
    await jiraPage.filterDocs.click()
    await expect(jiraPage.filterDocs).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
  })

  test("user can filter dev documents by sub-type", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.docFilterAll).toHaveClass(/is-active/)
    await jiraPage.docFilterUt.click()
    await expect(jiraPage.docFilterUt).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(3)
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
    await expect(jiraPage.emptyColumnState("done")).toBeVisible()
    await jiraPage.docFilterQuery.click()
    await expect(jiraPage.docFilterQuery).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(2)
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
    await expect(jiraPage.emptyColumnState("done")).toBeVisible()
    await jiraPage.docFilterSop.click()
    await expect(jiraPage.docFilterSop).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(2)
    await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
    await expect(jiraPage.kanbanCards("done")).toHaveCount(1)
    await jiraPage.docFilterAll.click()
    await expect(jiraPage.docFilterAll).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
  })

  test("user can switch to backlog tab and view active sprint, future sprint, and backlog issues", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.tabBacklog.click()
    await expect(jiraPage.tabBacklog).toHaveClass(/is-active/)
    await expect(jiraPage.backlogContainer).toBeVisible()
    await expect(jiraPage.backlogSprints).toHaveCount(2)
    await expect(jiraPage.backlogContainer).toContainText("ACTIVE")
    await expect(jiraPage.backlogIssues).toHaveCount(2)
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "03_jira_backlog_view.png"),
      fullPage: true
    })
  })

  test("user can click backlog issue to open issue detail modal", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.tabBacklog.click()
    await expect(jiraPage.backlogContainer).toBeVisible()
    await jiraPage.backlogIssues.first().click()
    await expect(jiraPage.issueModal).toBeVisible()
    await expect(jiraPage.modalKey).toHaveText("CRMMS-77911")
    await expect(jiraPage.modalTitle).toContainText("Sebagai Administrator")
    await jiraPage.btnCloseModal.click()
    await expect(jiraPage.issueModal).not.toBeVisible()
  })

  test("user sees skeleton loader instead of dummy data while Jira data is loading", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    let fulfillIssues: () => void = () => {}
    const issuesPromise = new Promise<void>((resolve) => {
      fulfillIssues = resolve
    })

    await page.route(/\/api\/jira\/issues/, async (route) => {
      await issuesPromise
      await route.fulfill({ status: 200, json: mockAssignedIssues })
    })
    await page.route(/\/api\/jira\/backlog/, async (route) => {
      await issuesPromise
      await route.fulfill({ status: 200, json: mockBacklogResponse })
    })

    const jiraPage = new JiraPage(page)
    await jiraPage.goto()

    await expect(jiraPage.kanbanSkeleton).toBeVisible()
    await expect(jiraPage.kanbanCards()).toHaveCount(0)

    fulfillIssues()

    await expect(jiraPage.kanbanSkeleton).not.toBeVisible()
    await expect(jiraPage.kanbanCol("open")).toBeVisible()
  })

  test("user can switch to all filter and view all assigned issues", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.filterAll.click()
    await expect(jiraPage.filterAll).toHaveClass(/is-active/)
    await expect(jiraPage.kanbanCol("open")).toBeVisible()
    await expect(jiraPage.colCount("open")).toHaveText("9")
  })

  test("user sees squad filter pills and can filter sprints by squad", async ({
    page
  }) => {
    await routeSecrets(page, { has_jira_pat: true })
    const multiSquadBacklog = {
      sprints: [
        {
          id: "sprint-44",
          name: "Sprint 44 - Fortune Squad",
          is_active: true,
          dates: "Sep 15 - Sep 29",
          issues: [
            {
              id: "CRMMS-77911",
              key: "CRMMS-77911",
              kind: "story",
              title: "Fortune story",
              status: "progress",
              priority: "high",
              points: 5
            }
          ]
        }
      ],
      total_issues: 1,
      detected_squad: "Fortune Squad",
      available_squads: ["Fortune Squad", "Azzuri"]
    }

    await page.route(/\/api\/jira\/issues/, async (route) => {
      await route.fulfill({ status: 200, json: mockAssignedIssues })
    })
    await page.route(/\/api\/jira\/backlog/, async (route) => {
      await route.fulfill({ status: 200, json: multiSquadBacklog })
    })

    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await jiraPage.tabBacklog.click()

    await expect(jiraPage.squadFilterBar).toBeVisible()
    await expect(page.getByTestId("squad-pill-fortune-squad")).toHaveClass(/is-active/)
    await expect(page.getByTestId("squad-pill-fortune-squad")).toContainText("Yours")
    await expect(page.getByTestId("squad-pill-azzuri")).toBeVisible()
  })

  test("returning to jira keeps the board without skeleton", async ({ page }) => {
    await routeSecrets(page, { has_jira_pat: true })
    await routeStandardJira(page)
    const jiraPage = new JiraPage(page)
    await jiraPage.goto()
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(2)
    await expect(jiraPage.kanbanCards("done")).toHaveCount(1)

    const mainNavigation = page.getByRole("navigation", {
      name: "Main Navigation"
    })
    await mainNavigation.getByRole("link", { name: "Settings" }).click()
    await expect(page).toHaveURL(/\/settings/)
    await mainNavigation.getByRole("link", { name: "Jira BRI" }).click()
    await expect(page).toHaveURL(/\/jira/)

    await expect(jiraPage.kanbanCol("open")).toBeVisible()
    await expect(jiraPage.kanbanSkeleton).not.toBeVisible()
    await expect(jiraPage.colCount("open")).toHaveText("7")
    await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    await expect(jiraPage.colCount("progress")).toHaveText("2")
    await expect(jiraPage.kanbanCards("progress")).toHaveCount(2)
    await expect(jiraPage.colCount("done")).toHaveText("1")
    await expect(jiraPage.kanbanCards("done")).toHaveCount(1)
    await page.screenshot({
      path: path.join(EVIDENCE_DIR, "07_jira_cached_revisit.png"),
      fullPage: true
    })
  })
})

