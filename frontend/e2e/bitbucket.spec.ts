import { test, expect } from "@playwright/test"
import path from "path"
import { BitbucketPage } from "./pages/BitbucketPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/bitbucket/evidence"

const mockPushes = [
  {
    id: "push-1",
    repo: "BRI/mms-frontend",
    repo_mark: "FE",
    branch: "feat/auth-refresh",
    commit_count: 3,
    relative_time: "10m ago",
    status: "ready",
    ai_badge: "AI Ready",
    ai_summary: "Authentication session handling refresh with token rotation.",
    suggested_title: "feat(auth): add session rotation"
  },
  {
    id: "push-2",
    repo: "BRI/mms-backend",
    repo_mark: "BE",
    branch: "fix/cors-preflight",
    commit_count: 1,
    relative_time: "25m ago",
    status: "ready",
    ai_badge: "AI Clean",
    ai_summary: "Fix CORS preflight headers on standard net/http mux.",
    suggested_title: "fix(cors): allow credentials and expose headers"
  },
  {
    id: "push-3",
    repo: "BRI/service-map",
    repo_mark: "SM",
    branch: "chore/update-deps",
    commit_count: 2,
    relative_time: "3d ago",
    status: "stale",
    ai_badge: "AI Stale",
    ai_summary: "Dependency updates pending merge for more than 48 hours.",
    suggested_title: "chore: update dependencies"
  }
]

const mockPRs = [
  {
    id: "#184",
    number: 184,
    repo: "BRI/mms-frontend",
    title: "feat(auth): LUN-421 support biometric and session renewal",
    source_branch: "feat/auth-refresh",
    target_branch: "main",
    status: "open",
    updated_relative: "12m ago",
    author: "Lunar Developer",
    files_count: 4,
    lines_added: 124,
    lines_deleted: 18,
    is_assigned_to_me: true,
    is_ai_flagged: true
  },
  {
    id: "#188",
    number: 188,
    repo: "BRI/mms-backend",
    title: "fix(db): SQLite WAL busy timeout configuration",
    source_branch: "fix/sqlite-wal",
    target_branch: "main",
    status: "open",
    updated_relative: "1h ago",
    author: "Current User",
    files_count: 2,
    lines_added: 42,
    lines_deleted: 6,
    is_assigned_to_me: true,
    is_ai_flagged: true
  },
  {
    id: "#191",
    number: 191,
    repo: "BRI/service-map",
    title: "refactor(fts5): optimize tokenization query speed",
    source_branch: "perf/fts5-tokenize",
    target_branch: "main",
    status: "open",
    updated_relative: "3h ago",
    author: "Alex",
    files_count: 5,
    lines_added: 88,
    lines_deleted: 34,
    is_assigned_to_me: false,
    is_ai_flagged: true
  },
  {
    id: "#195",
    number: 195,
    repo: "BRI/timesheet-core",
    title: "docs: update API contract and swagger specs",
    source_branch: "docs/api-specs",
    target_branch: "main",
    status: "open",
    updated_relative: "5h ago",
    author: "Sarah",
    files_count: 1,
    lines_added: 15,
    lines_deleted: 2,
    is_assigned_to_me: false,
    is_ai_flagged: false
  }
]

const mockDiff = {
  pr_id: "184",
  repo: "BRI/mms-frontend",
  total_added: 124,
  total_deleted: 18,
  files: [
    {
      old_path: "src/auth/session.ts",
      new_path: "src/auth/session.ts",
      status: "modified",
      additions: 124,
      deletions: 18,
      hunks: [
        {
          header: "@@ -40,6 +40,12 @@",
          lines: [
            "- const prev = null",
            "+ const prev = sessionManager.get()",
            "+ The compatibility layer is implemented cleanly"
          ]
        }
      ]
    }
  ]
}

const mockAIReview = {
  pr_id: "184",
  summary:
    "Automated analysis completed. Verified changed files and scope against standard lint and test coverage guidelines.",
  findings: [
    "Authentication session token lifecycle complies with security standards.",
    "Zero code comments constraint preserved.",
    "Unit tests verify negative unauthorized states."
  ],
  generated_comment:
    "Lunar Copilot review: Automated analysis completed. Verified changed files and scope against standard lint and test coverage guidelines."
}

function routeStandardPushes(page: import("@playwright/test").Page) {
  return page.route(/\/api\/bitbucket\/pushes/, async (route) => {
    const url = new URL(route.request().url())
    const filter = url.searchParams.get("filter")
    if (filter === "ready") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "ready") })
    } else if (filter === "stale") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "stale") })
    } else {
      await route.fulfill({ json: mockPushes })
    }
  })
}

function routeStandardPRs(page: import("@playwright/test").Page) {
  return page.route(/\/api\/bitbucket\/prs/, async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() === "POST" && url.pathname.endsWith("/prs")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "#196",
          number: 196,
          repo: payload.repo,
          title: payload.title,
          source_branch: payload.source_branch,
          target_branch: payload.target_branch,
          status: "open",
          updated_relative: "Just now",
          author: "Lunar Developer",
          files_count: 2,
          lines_added: 45,
          lines_deleted: 8,
          is_assigned_to_me: false,
          is_ai_flagged: false
        }
      })
      return
    }

    if (url.pathname.includes("/diff")) {
      await route.fulfill({ json: mockDiff })
      return
    }

    if (url.pathname.includes("/ai-review")) {
      await route.fulfill({ json: mockAIReview })
      return
    }

    if (url.pathname.includes("/action")) {
      await route.fulfill({ status: 200, json: { status: "success", action: "approve", pr_id: "188" } })
      return
    }

    if (url.pathname.includes("/comments")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "comm-101",
          pr_id: "191",
          repo: "BRI/service-map",
          author: "Lunar Developer",
          content: payload.content,
          created_at: "Just now"
        }
      })
      return
    }

    const filter = url.searchParams.get("filter")
    if (filter === "ai") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_ai_flagged) })
    } else if (filter === "mine") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_assigned_to_me) })
    } else {
      await route.fulfill({ json: mockPRs })
    }
  })
}

function routeSecretsConfigured(page: import("@playwright/test").Page, providers = ["deepseek"]) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        json: {
          user_id: "test-user-id",
          has_jira_pat: true,
          jira_username: "developer",
          has_bitbucket_pat: true,
          bitbucket_username: "developer",
          has_confluence_pat: true,
          has_ai_keys: providers.length > 0,
          configured_ai_providers: providers,
          updated_at: "Just now"
        }
      })
      return
    }
    await route.continue()
  })
}

test.describe("Bitbucket Code Review Workspace", () => {
  test("engineer can view pushed branches and open pull requests on load", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await expect(bitbucketPage.pushCards).toHaveCount(3)
    await expect(bitbucketPage.prTableRows).toHaveCount(4)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_bitbucket_overview.png"), fullPage: true })
  })

  test("engineer can filter pushed branches by ready status", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.pushFilterButton("ready").click()
    await expect(bitbucketPage.pushCards).toHaveCount(2)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_pushes_filter_ready.png") })
  })

  test("engineer can filter pushed branches by stale status", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.pushFilterButton("stale").click()
    await expect(bitbucketPage.pushCards).toHaveCount(1)
  })

  test("engineer can restore all pushed branches after applying a filter", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.pushFilterButton("ready").click()
    await expect(bitbucketPage.pushCards).toHaveCount(2)
    await bitbucketPage.pushFilterButton("all").click()
    await expect(bitbucketPage.pushCards).toHaveCount(3)
  })

  test("engineer can filter pull requests by AI flagged", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.prFilterButton("ai").click()
    await expect(bitbucketPage.prTableRows).toHaveCount(3)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_prs_filter_ai_flagged.png") })
  })

  test("engineer can filter pull requests by assigned to me", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.prFilterButton("mine").click()
    await expect(bitbucketPage.prTableRows).toHaveCount(2)
  })

  test("engineer can restore all pull requests after applying a filter", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.prFilterButton("mine").click()
    await expect(bitbucketPage.prTableRows).toHaveCount(2)
    await bitbucketPage.prFilterButton("all").click()
    await expect(bitbucketPage.prTableRows).toHaveCount(4)
  })

  test("engineer can open create PR modal and submit a new pull request with toast confirmation", async ({
    page
  }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.createPrButton("push-1").click()
    await expect(bitbucketPage.createPrModal).toBeVisible()
    await expect(bitbucketPage.createPrSourceBranch).toContainText("feat/auth-refresh")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_create_pr_modal.png") })
    await bitbucketPage.submitPrButton.click()
    await expect(bitbucketPage.globalToast).toBeVisible()
    await expect(bitbucketPage.globalToast).toContainText("Pull request created: feat/auth-refresh → main")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "05_create_pr_toast.png") })
    await expect(bitbucketPage.createPrModal).not.toBeVisible()
  })

  test("engineer can see model selector in PR review modal populated with configured models", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    await routeSecretsConfigured(page, ["deepseek"])

    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(184).click()
    await expect(bitbucketPage.reviewModal).toBeVisible()

    await expect(bitbucketPage.prReviewModelSelect).toBeVisible()
    await expect(bitbucketPage.prReviewModelSelect).toContainText("DeepSeek-V4 Pro (Thinking)")
    await expect(bitbucketPage.prReviewModelSelect).toContainText("DeepSeek Flash")

    await bitbucketPage.prReviewModelSelect.selectOption("DeepSeek Flash")
    await expect(bitbucketPage.prReviewModelSelect).toHaveValue("DeepSeek Flash")
  })

  test("engineer sees warning when no AI provider is configured in PR review modal", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    await page.route(/\/api\/auth\/secrets/, async (route) => {
      await route.fulfill({
        status: 200,
        json: {
          has_ai_keys: false,
          configured_ai_providers: []
        }
      })
    })

    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(184).click()
    await expect(bitbucketPage.reviewModal).toBeVisible()

    await expect(bitbucketPage.noAiKeysBanner).toBeVisible()
    await expect(bitbucketPage.noAiKeysBanner).toContainText("No AI provider configured yet")
  })

  test("engineer can open PR review modal and inspect diff then generate AI review", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    await routeSecretsConfigured(page, ["deepseek"])

    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(184).click()
    await expect(bitbucketPage.reviewModal).toBeVisible()
    await expect(bitbucketPage.reviewModalTitle).toContainText("LUN-421")
    await expect(bitbucketPage.aiEmptyState).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "06_pr_review_modal_empty.png") })

    await bitbucketPage.toggleDiffButton.click()
    await expect(bitbucketPage.diffViewer).toBeVisible()
    await bitbucketPage.triggerAiReviewButton.click()

    await expect(bitbucketPage.aiFindings).toBeVisible({ timeout: 5000 })
    await expect(bitbucketPage.aiSummary).toContainText("Automated analysis completed")
    await expect(bitbucketPage.reviewCommentInput).toHaveValue(/Lunar Copilot review/)
    await expect(bitbucketPage.globalToast).toBeVisible()
    await expect(bitbucketPage.globalToast).toContainText("AI review generated")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "07_pr_review_ai_generated.png") })
  })

  test("engineer receives clear error toast when AI review request fails with upstream 401 Unauthorized", async ({
    page
  }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    await routeSecretsConfigured(page, ["deepseek"])
    await page.route(/\/api\/bitbucket\/prs\/.*ai-review/, async (route) => {
      await route.fulfill({
        status: 401,
        json: { error: "unauthorized: invalid or expired deepseek API key" }
      })
    })

    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(184).click()
    await expect(bitbucketPage.reviewModal).toBeVisible()

    await bitbucketPage.triggerAiReviewButton.click()
    await expect(bitbucketPage.globalToast).toBeVisible({ timeout: 5000 })
    await expect(bitbucketPage.globalToast).toContainText("invalid or expired deepseek API key")
  })

  test("engineer receives clear error toast when AI review request fails with upstream 502 Bad Gateway", async ({
    page
  }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    await routeSecretsConfigured(page, ["deepseek"])
    await page.route(/\/api\/bitbucket\/prs\/.*ai-review/, async (route) => {
      await route.fulfill({
        status: 502,
        json: { error: "failed to contact AI provider (deepseek): connection timeout" }
      })
    })

    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(184).click()
    await expect(bitbucketPage.reviewModal).toBeVisible()

    await bitbucketPage.triggerAiReviewButton.click()
    await expect(bitbucketPage.globalToast).toBeVisible({ timeout: 5000 })
    await expect(bitbucketPage.globalToast).toContainText("connection timeout")
  })

  test("engineer can approve a pull request and see review status toast", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(188).click()
    await bitbucketPage.approveButton().click()
    await expect(bitbucketPage.globalToast).toBeVisible()
    await expect(bitbucketPage.globalToast).toContainText("Review marked as approve.")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "08_pr_review_status_action.png") })
  })

  test("engineer can post a comment to a pull request and see confirmation toast", async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.reviewButton(191).click()
    await bitbucketPage.reviewCommentInput.fill("LGTM! Verified hook signature change with unit test coverage.")
    await bitbucketPage.sendCommentButton.click()
    await expect(bitbucketPage.globalToast).toBeVisible()
    await expect(bitbucketPage.globalToast).toContainText("Review posted to the pull request as a comment.")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "09_pr_review_comment_posted.png") })
    await expect(bitbucketPage.reviewModal).not.toBeVisible()
  })

  test("engineer can see empty state message when no pushes or pull requests exist", async ({ page }) => {
    await page.route(/\/api\/bitbucket\/pushes/, (route) => route.fulfill({ json: [] }))
    await page.route(/\/api\/bitbucket\/prs/, (route) => route.fulfill({ json: [] }))
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await expect(bitbucketPage.emptyStates.first()).toContainText("No pushed branches match this filter.")
    await expect(bitbucketPage.emptyStates.nth(1)).toContainText("No pull requests match this filter.")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "10_bitbucket_empty_state.png") })
  })

  test("engineer cannot access Bitbucket data when PAT is invalid or expired (401 Unauthorized)", async ({
    page
  }) => {
    await routeSecretsConfigured(page)
    await page.route(/\/api\/bitbucket\/pushes/, (route) =>
      route.fulfill({ status: 401, json: { error: "unauthorized access: invalid or expired Bitbucket PAT" } })
    )
    await page.route(/\/api\/bitbucket\/prs/, (route) =>
      route.fulfill({ status: 401, json: { error: "unauthorized access: invalid or expired Bitbucket PAT" } })
    )
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.assertErrorBannerContains("invalid or expired Bitbucket PAT")
    await expect(bitbucketPage.errorBanner.locator("a")).toContainText("Update in Settings →")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "11_bitbucket_invalid_pat_negative.png") })
  })

  test("engineer sees VPN guidance when server cannot reach Bitbucket due to network failure (502)", async ({
    page
  }) => {
    await routeSecretsConfigured(page)
    const vpnError =
      "failed to contact Bitbucket Server (https://bitbucket.bri.co.id): please verify your BRI VPN connection"
    await page.route(/\/api\/bitbucket\/pushes/, (route) =>
      route.fulfill({ status: 502, json: { error: vpnError } })
    )
    await page.route(/\/api\/bitbucket\/prs/, (route) =>
      route.fulfill({ status: 502, json: { error: vpnError } })
    )
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.assertErrorBannerContains("verify your BRI VPN connection")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "12_bitbucket_vpn_disconnect_negative.png") })
  })

  test("engineer cannot access Bitbucket data when PAT lacks repository permissions (403 Forbidden)", async ({
    page
  }) => {
    await routeSecretsConfigured(page)
    await page.route(/\/api\/bitbucket\/pushes/, (route) =>
      route.fulfill({
        status: 403,
        json: { error: "access forbidden: your PAT does not have permission for this repository" }
      })
    )
    await page.route(/\/api\/bitbucket\/prs/, (route) =>
      route.fulfill({
        status: 403,
        json: { error: "access forbidden: your PAT does not have permission for this repository" }
      })
    )
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await bitbucketPage.assertErrorBannerContains("access forbidden")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "13_bitbucket_access_forbidden_negative.png") })
  })

  test("engineer sees configuration warning when Bitbucket PAT has not been set up yet", async ({ page }) => {
    await page.route(/\/api\/auth\/secrets/, (route) =>
      route.fulfill({
        status: 200,
        json: { has_bitbucket_pat: false, has_jira_pat: false, has_confluence_pat: false }
      })
    )
    await page.route(/\/api\/bitbucket\/pushes/, (route) => route.fulfill({ json: [] }))
    await page.route(/\/api\/bitbucket\/prs/, (route) => route.fulfill({ json: [] }))
    const bitbucketPage = new BitbucketPage(page)
    await bitbucketPage.navigateTo()
    await expect(bitbucketPage.noPatWarningBanner).toBeVisible()
    await expect(bitbucketPage.noPatWarningBanner).toContainText(
      "Bitbucket Personal Access Token (PAT) is not configured yet"
    )
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "14_bitbucket_no_pat_configured.png") })
  })
})
