import { test, expect } from "../../../fixtures/authenticated.fixture"
import {
  buildUtRunStream,
  captureClipboard,
  routeHelperWorkspace,
  routeMigTargets,
  routeQueryReview,
  routeQueryReviewApply,
  routeSecrets,
  routeStandardConfluence,
  routeSystemConfig,
  routeUtDocs,
  routeUtDocsApply,
  routeUtRunHelper
} from "../../../helpers/route-mock"
import {
  buildMockUtRunDone,
  mockMigTargetsEmpty,
  mockMigTargetsSingle,
  mockQueryReviewApplyConflict,
  mockQueryReviewApplyResult,
  mockQueryReviewEmpty,
  mockQueryReviewMixed,
  mockQueryReviewNoQueries,
  mockUtDocsApplyConflict,
  mockUtDocsApplyResult,
  mockUtDocsResultGrouped
} from "../../../data/confluence.data"
import { captureEvidence } from "../../../helpers/evidence"

const buildPerRepositoryStream = (repository: string): string =>
  buildUtRunStream(buildMockUtRunDone(repository))

test.describe("Confluence BRI - Document Actions Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-ACT-002 UT generate reads the MIG targets, shows the skipped repositories, and runs one mock-only pass per repository",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page)
      const utRun = routeUtRunHelper(page, {
        buildStream: (request) => buildPerRepositoryStream(request.repo ?? "")
      })
      const utDocs = routeUtDocs(page, { responses: [{ status: 200, body: mockUtDocsResultGrouped }] })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utDocument).toBeVisible()
      await expect(confluencePage.utMigPage).toContainText("CRMMS-76972-MIG")
      await expect(confluencePage.utTargets).toHaveCount(2)
      await expect(confluencePage.utSkippedRepositories).toContainText("fe-mms")
      await expect(confluencePage.utSkippedRepositories).toContainText("mms")
      await expect(confluencePage.utSkippedRepositories).toContainText("excluded repository")

      expect(utRun.runRequests).toHaveLength(2)
      expect(JSON.parse(utRun.runRequests[0])).toEqual({
        root: "/Users/erendt/BRI",
        repo: "micro-payment",
        branch: "feat/CRMMS-76972"
      })
      expect(JSON.parse(utRun.runRequests[1])).toEqual({
        root: "/Users/erendt/BRI",
        repo: "micro-notification",
        branch: "feat/CRMMS-76972"
      })

      expect(utDocs.requests).toHaveLength(1)
      const documentPayload = JSON.parse(utDocs.requests[0])
      expect(documentPayload.model).toBe("DeepSeek-V4 Pro (Thinking)")
      expect(documentPayload.repositories).toHaveLength(2)
      expect(documentPayload.repositories[0].repository).toBe("micro-payment")
      expect(documentPayload.repositories[0].tests[0].name).toBe("TestCreatePayment")
      expect(documentPayload.repositories[1].repository).toBe("micro-notification")
    }
  )

  test(
    "CONF-ACT-003 UT run generates the document and enables the gated actions without a stub",
    {
      tag: ["@confluence", "@positive", "@actions", "@toolbar"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsSingle }] })
      await routeUtRunHelper(page, { streamDelayMs: 800 })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")

      await expect(confluencePage.instantApplyButton).toBeDisabled()
      await expect(confluencePage.copyDescriptionButton).toBeDisabled()
      await captureEvidence(page, "confluence", "05_confluence_actions_disabled_initial.png")

      await routeUtDocs(page)
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.generateAiButton).toBeDisabled()
      await expect(confluencePage.generateAiButton).toHaveAttribute("aria-busy", "true")
      await expect(confluencePage.generateAiButton).toContainText("Generating UT docs")

      await expect(confluencePage.utDocument).toBeVisible()
      await expect(confluencePage.utDocumentTitle).toContainText("UT documentation for micro-payment")

      await expect(confluencePage.instantApplyButton).toBeEnabled()
      await expect(confluencePage.copyDescriptionButton).toBeEnabled()
      await page.waitForTimeout(200)
      await captureEvidence(page, "confluence", "06_confluence_actions_enabled_after_ai.png")

      await page.getByRole("button", { name: "Edit" }).click()
      await expect(confluencePage.globalToast).toContainText("Edit is ready for backend wiring.")

      await page.getByRole("button", { name: "Share" }).click()
      await expect(confluencePage.globalToast).toContainText("Share is ready for backend wiring.")

      await page.getByRole("button", { name: "More" }).click()
      await expect(confluencePage.globalToast).toContainText("More actions are ready for backend wiring.")
    }
  )

  test(
    "CONF-ACT-005 generate is disabled and the Settings link is shown when no AI provider is configured",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeSecrets(page, { hasConfluencePat: true, hasAiKeys: false, configuredAiProviders: [] })
      await routeSystemConfig(page, [])

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")

      await expect(confluencePage.noAiKeysBanner).toBeVisible()
      await expect(confluencePage.noAiKeysBanner.getByRole("link", { name: /Settings/ })).toBeVisible()
      await expect(confluencePage.generateAiButton).toBeDisabled()
    }
  )

  test(
    "CONF-ACT-004 Commenting action triggers toast and validates input",
    {
      tag: ["@confluence", "@positive", "@actions", "@comments"]
    },
    async ({ confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")

      await confluencePage.commentInput.fill("Test comment")
      await confluencePage.commentPostButton.click()
      await expect(confluencePage.globalToast).toContainText("Commenting is ready for backend wiring.")
      await expect(confluencePage.commentItems).toHaveCount(0)

      await confluencePage.commentInput.clear()
      await confluencePage.commentPostButton.click()
      await expect(confluencePage.commentInput).toBeFocused()
    }
  )

  test(
    "CONF-QUERY-001 query document extracts the branch queries, renders the capture table, and copies a Confluence-ready table",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      const queryReview = routeQueryReview(page)
      await captureClipboard(page)
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")

      await expect(confluencePage.queryIdleState).toBeVisible()
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.queryResult).toBeVisible()
      expect(JSON.parse(queryReview.requests[0])).toMatchObject({
        model: "DeepSeek-V4 Pro (Thinking)"
      })

      await expect(confluencePage.queryMigPage).toContainText("CRMMS-76972-MIG")
      await expect(confluencePage.querySkippedRepositories).toContainText("fe-mms")
      await expect(confluencePage.querySkippedRepositories).toContainText("excluded repository")

      await expect(confluencePage.queryRepositories).toHaveCount(2)
      await expect(confluencePage.queryRepositoryNames.nth(0)).toContainText("[micro-payment]")
      await expect(confluencePage.queryRepositoryNames.nth(1)).toContainText("[micro-notification]")
      await expect(confluencePage.queryRepositoryStatuses.nth(0)).toContainText("Success")
      await expect(confluencePage.queryRepositoryStatuses.nth(1)).toContainText("Success")
      await expect(confluencePage.queryRepositoryBranches.nth(0)).toContainText("feat/CRMMS-123")
      await expect(confluencePage.queryTables).toHaveCount(2)
      await expect(confluencePage.queryRows).toHaveCount(2)

      await expect(confluencePage.queryTableHeaders).toHaveText([
        "No",
        "Fitur",
        "Query Statement",
        "Index",
        "Capture Explain / Execution Plan"
      ])
      await expect(confluencePage.queryRows.first()).toContainText("[micro-payment] Get user by PN")
      await expect(confluencePage.queryRows.first().locator("pre").first()).toContainText(
        "select id, username"
      )
      await expect(confluencePage.queryIndexCells.first()).toContainText("pn")
      await expect(confluencePage.queryIndexCells.first()).toContainText(
        "CREATE INDEX idx_user_data_pn ON user_data (pn);"
      )
      await expect(confluencePage.queryIndexCells.nth(1)).toContainText("PRIMARY")
      await expect(confluencePage.queryIndexCells.nth(1)).not.toContainText("CREATE INDEX")
      await expect(confluencePage.queryCopyStatementButtons).toHaveCount(0)
      await expect(confluencePage.queryRows.first().locator("td").last()).toHaveText("")

      await confluencePage.queryCopyTableButton.click()
      await expect(confluencePage.globalToast).toContainText("Query table copied for Confluence.")
      const captured = await page.evaluate(
        () =>
          (window as unknown as { __capturedClipboard: Record<string, string>[] })
            .__capturedClipboard
      )
      expect(captured[0]["text/html"]).toContain("Capture Explain / Execution Plan")
      expect(captured[0]["text/html"]).toContain("select id, username")
      expect(captured[0]["text/html"]).toContain(
        "CREATE INDEX idx_user_data_pn ON user_data (pn);"
      )
      expect(captured[0]["text/plain"]).toContain("Query Statement")
      expect(captured[0]["text/plain"]).toContain(
        "CREATE INDEX idx_user_data_pn ON user_data (pn);"
      )

      await confluencePage.queryCopyJsonButton.click()
      await expect(confluencePage.globalToast).toContainText("Raw query review copied as JSON.")

      await captureEvidence(page, "confluence", "07_confluence_query_review_result.png")
    }
  )

  test(
    "CONF-QUERY-006 Instant apply writes the capture table and toasts the new version",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page)
      const apply = routeQueryReviewApply(page, {
        responses: [{ status: 200, body: mockQueryReviewApplyResult }]
      })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.queryResult).toBeVisible()

      await expect(confluencePage.instantApplyButton).toBeEnabled()
      await confluencePage.instantApplyButton.click()

      await expect(confluencePage.globalToast).toContainText("New version 12")
      expect(apply.requests).toHaveLength(1)
      const applyPayload = JSON.parse(apply.requests[0])
      expect(applyPayload.repositories).toHaveLength(2)
      expect(applyPayload.repositories[0]).toMatchObject({
        repository: "micro-payment",
        queries: [
          {
            no: 1,
            feature: "[micro-payment] Get user by PN",
            query_statement: "select id, username from user_data where pn = '[input_here]'",
            index: "pn"
          }
        ]
      })
    }
  )

  test(
    "CONF-QUERY-007 a failed Instant apply shows an in-panel error and retry succeeds",
    {
      tag: ["@confluence", "@negative", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page)
      const apply = routeQueryReviewApply(page, {
        responses: [
          { status: 502, body: mockQueryReviewApplyConflict },
          { status: 200, body: mockQueryReviewApplyResult }
        ]
      })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.queryResult).toBeVisible()

      await confluencePage.instantApplyButton.click()
      await expect(confluencePage.queryApplyError).toContainText("updated elsewhere")

      await confluencePage.queryApplyRetryButton.click()
      await expect(confluencePage.globalToast).toContainText("New version 12")
      expect(apply.requests).toHaveLength(2)
    }
  )

  test(
    "CONF-QUERY-002 a repository with no changed query renders the fallback row",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page, { responses: [{ status: 200, body: mockQueryReviewNoQueries }] })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.queryResult).toBeVisible()
      await expect(confluencePage.queryRepositories).toHaveCount(1)
      await expect(confluencePage.queryRepositoryStatuses.first()).toContainText("No queries")
      await expect(confluencePage.queryNoQueriesNote).toBeVisible()
      await expect(confluencePage.queryNoQueriesNote).toContainText(
        "Tidak ada penambahan atau perubahan query pada pengembangan kali ini."
      )
      await expect(confluencePage.queryTables).toHaveCount(1)
      await expect(confluencePage.queryRows).toHaveCount(0)
    }
  )

  test(
    "CONF-QUERY-003 a failing repository shows the reason and does not hide the successful one",
    {
      tag: ["@confluence", "@negative", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page, { responses: [{ status: 200, body: mockQueryReviewMixed }] })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.queryResult).toBeVisible()
      await expect(confluencePage.queryRepositories).toHaveCount(2)
      await expect(confluencePage.queryRepositoryStatuses.nth(0)).toContainText("Success")
      await expect(confluencePage.queryRepositoryStatuses.nth(1)).toContainText("Error")
      await expect(confluencePage.queryRepositoryError).toContainText("upstream returned 502")
      await expect(confluencePage.queryNoQueriesNote).toHaveCount(0)
      await expect(confluencePage.queryRows).toHaveCount(1)
    }
  )

  test(
    "CONF-QUERY-004 the loading state is shown while the extraction is in flight",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page, { delayMs: 800 })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.queryLoading).toBeVisible()
      await expect(confluencePage.queryResult).toBeVisible()
      await expect(confluencePage.queryRepositories).toHaveCount(2)
    }
  )

  test(
    "CONF-QUERY-005 an empty repository list shows the empty state",
    {
      tag: ["@confluence", "@edge", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page, { responses: [{ status: 200, body: mockQueryReviewEmpty }] })
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.queryEmptyState).toBeVisible()
      await expect(confluencePage.queryResult).toHaveCount(0)
      await expect(confluencePage.queryRepositories).toHaveCount(0)
    }
  )

  test(
    "CONF-UT-001 UT run renders phases, streamed output, the run summary, the blocked note, the seven-column table, and the screenshot",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsSingle }] })
      await routeUtRunHelper(page)
      await routeUtDocs(page)
      await captureClipboard(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utPhases).toBeVisible()
      await expect(confluencePage.utStatus).toContainText("Passed")
      await expect(confluencePage.utCounts).toContainText("2/2 passed")
      await expect(confluencePage.utWorktree).toContainText("Local changes restored")
      await expect(confluencePage.utOutput).toContainText("=== RUN   TestDao")
      await expect(confluencePage.utBlocked).toContainText("internal/seed/reset_test.go")
      await expect(confluencePage.utBlocked).toContainText("Not executed")
      await expect(confluencePage.utBlockedNotice).toContainText("can reset a database")
      await expect(confluencePage.utDocumentTitle).toContainText("UT documentation for micro-payment")

      await expect(confluencePage.utTableHeaders).toHaveText([
        "No",
        "Function",
        "Scenario",
        "Expected Result",
        "Input Data",
        "Result",
        "Screen Capture"
      ])
      await expect(confluencePage.utGroups).toHaveCount(0)
      await expect(confluencePage.utRows).toHaveCount(2)
      await expect(confluencePage.utRowFunctions.nth(0)).toContainText("CreatePayment")
      await expect(confluencePage.utRowResults.nth(0)).toContainText("Passed")
      await expect(confluencePage.utRowResults.nth(1)).toContainText("Failed")
      await expect(confluencePage.utRows.first().locator("td").last()).toContainText("run-micro-payment.png")
      await expect(confluencePage.utScreenshot).toBeVisible()

      await confluencePage.utCopyTableButton.click()
      await expect(confluencePage.globalToast).toContainText("UT scenario table copied for Confluence.")
      const captured = await page.evaluate(
        () =>
          (window as unknown as { __capturedClipboard: Record<string, string>[] })
            .__capturedClipboard
      )
      expect(captured[0]["text/html"]).toContain("Function")
      expect(captured[0]["text/html"]).toContain("Expected Result")
      expect(captured[0]["text/html"]).toContain("CreatePayment")
      expect(captured[0]["text/plain"]).toContain("Expected Result")

      await captureEvidence(page, "confluence", "08_confluence_ut_docs_result.png")
    }
  )

  test(
    "CONF-UT-002 a two-repository run groups the table with a Layanan / Microservice row per repository",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page)
      await routeUtRunHelper(page, {
        buildStream: (request) => buildPerRepositoryStream(request.repo ?? "")
      })
      await routeUtDocs(page, { responses: [{ status: 200, body: mockUtDocsResultGrouped }] })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utDocument).toBeVisible()
      await expect(confluencePage.utSummaries).toHaveCount(2)
      await expect(confluencePage.utGroups).toHaveCount(2)
      await expect(confluencePage.utGroups.nth(0)).toContainText("Layanan / Microservice: micro-payment")
      await expect(confluencePage.utGroups.nth(1)).toContainText("Layanan / Microservice: micro-notification")
      await expect(confluencePage.utRows).toHaveCount(4)
      await expect(confluencePage.utRowResults.nth(3)).toContainText("Skipped")

      const paymentGroup = confluencePage.utTable.locator('[data-testid="ai-ut-group"]').nth(0)
      await expect(paymentGroup.locator("xpath=following-sibling::tr[1]").locator("td").first()).toHaveText("1")
      const notificationGroup = confluencePage.utTable.locator('[data-testid="ai-ut-group"]').nth(1)
      await expect(notificationGroup.locator("xpath=following-sibling::tr[1]").locator("td").first()).toHaveText("1")
    }
  )

  test(
    "CONF-UT-003 Instant apply uploads each repository screenshot and its rows, then toasts the new version",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page)
      await routeUtRunHelper(page, {
        buildStream: (request) => buildPerRepositoryStream(request.repo ?? "")
      })
      await routeUtDocs(page, { responses: [{ status: 200, body: mockUtDocsResultGrouped }] })
      const apply = routeUtDocsApply(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.utDocument).toBeVisible()

      await expect(confluencePage.instantApplyButton).toBeEnabled()
      await confluencePage.instantApplyButton.click()

      await expect(confluencePage.globalToast).toContainText("New version 13")
      expect(apply.requests).toHaveLength(1)
      const applyPayload = JSON.parse(apply.requests[0])
      expect(applyPayload.repositories).toHaveLength(2)
      expect(applyPayload.repositories[0].repository).toBe("micro-payment")
      expect(applyPayload.repositories[0].screenshot.filename).toBe("micro-payment-run-micro-payment.png")
      expect(applyPayload.repositories[0].screenshot.content_base64.length).toBeGreaterThan(0)
      expect(applyPayload.repositories[0].rows).toHaveLength(2)
      expect(applyPayload.repositories[0].rows[0]).toEqual({
        function: "CreatePayment",
        scenario: "Creates a payment with a valid payload",
        expected_result: "The payment is stored",
        input_data: "valid payload",
        result: "Passed"
      })
      expect(applyPayload.repositories[1].rows).toHaveLength(2)
    }
  )

  test(
    "CONF-UT-004 a failed Instant apply shows an in-panel error and the retry succeeds",
    {
      tag: ["@confluence", "@negative", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsSingle }] })
      await routeUtRunHelper(page)
      await routeUtDocs(page)
      const apply = routeUtDocsApply(page, {
        responses: [
          { status: 502, body: mockUtDocsApplyConflict },
          { status: 200, body: mockUtDocsApplyResult }
        ]
      })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.utDocument).toBeVisible()

      await confluencePage.instantApplyButton.click()
      await expect(confluencePage.utApplyError).toContainText("updated elsewhere")

      await confluencePage.utApplyRetryButton.click()
      await expect(confluencePage.globalToast).toContainText("New version 13")
      expect(apply.requests).toHaveLength(2)
    }
  )

  test(
    "CONF-UT-005 a MIG page with no repository shows the empty state and runs nothing",
    {
      tag: ["@confluence", "@edge", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsEmpty }] })
      const utRun = routeUtRunHelper(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utEmptyState).toBeVisible()
      await expect(confluencePage.utEmptyState).toContainText("No repository was found")
      await expect(confluencePage.utDocument).toHaveCount(0)
      expect(utRun.runRequests).toHaveLength(0)
    }
  )

  test(
    "CONF-CLIP-001 Copy button is gated until the query review result exists, then copies the description",
    {
      tag: ["@confluence", "@positive", "@clipboard"]
    },
    async ({ page, confluencePage }) => {
      await routeQueryReview(page)
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("QR-201")
      await page.context().grantPermissions(["clipboard-read", "clipboard-write"])

      await expect(confluencePage.copyDescriptionButton).toBeDisabled()
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.queryResult).toBeVisible()
      await expect(confluencePage.copyDescriptionButton).toBeEnabled()

      await confluencePage.copyDescriptionButton.click()
      await expect(confluencePage.globalToast).toContainText("Description copied to clipboard.")
    }
  )
})
