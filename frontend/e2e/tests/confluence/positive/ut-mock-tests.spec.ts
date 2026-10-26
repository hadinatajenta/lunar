import { test, expect } from "../../../fixtures/authenticated.fixture"
import {
  captureClipboard,
  routeHelperWorkspace,
  routeMigTargets,
  routeSecrets,
  routeStandardConfluence,
  routeUtDocs,
  routeUtRunHelper
} from "../../../helpers/route-mock"
import {
  mockMigTargetsSingle,
  mockUtDocsResultNotRunOnly,
  mockUtDocsResultWithMockCode
} from "../../../data/confluence.data"

test.describe("Confluence BRI - UT mock test code and Not Run rows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-UT-006 a Not Run row keeps its pill, leaves the Screen Capture cell empty, and shows the mock code block",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsSingle }] })
      await routeUtRunHelper(page)
      await routeUtDocs(page, { responses: [{ status: 200, body: mockUtDocsResultWithMockCode }] })
      await captureClipboard(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utDocument).toBeVisible()
      await expect(confluencePage.utRows).toHaveCount(2)
      await expect(confluencePage.utRowResults.nth(1)).toContainText("Not Run")
      await expect(confluencePage.utRowResults.nth(0)).toContainText("Passed")
      await expect(confluencePage.utRows.nth(1).locator("td").last()).toHaveText("")

      await expect(confluencePage.utMockTests).toBeVisible()
      await expect(confluencePage.utMockTests).toContainText("TestRefundPaymentMock")
      await expect(confluencePage.utMockTests.locator("pre code")).toContainText("RefundPayment")
      await expect(confluencePage.utMockTests.locator("code .hljs-keyword").first()).toBeVisible()

      await expect(confluencePage.utScreenshotStrip).toBeVisible()

      await confluencePage.utCopyTableButton.click()
      await expect(confluencePage.globalToast).toContainText("UT scenario table copied for Confluence.")
      const captured = await page.evaluate(
        () =>
          (window as unknown as { __capturedClipboard: Record<string, string>[] })
            .__capturedClipboard
      )
      expect(captured[0]["text/html"]).toContain('<pre><code class="language-go">')
      expect(captured[0]["text/html"]).toContain("TestRefundPaymentMock")
      expect(captured[0]["text/plain"]).toContain("```go")
      expect(captured[0]["text/plain"]).toContain("TestRefundPaymentMock")
    }
  )

  test(
    "CONF-UT-007 the screenshot strip is hidden when no row was executed",
    {
      tag: ["@confluence", "@edge", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeMigTargets(page, { responses: [{ status: 200, body: mockMigTargetsSingle }] })
      await routeUtRunHelper(page)
      await routeUtDocs(page, { responses: [{ status: 200, body: mockUtDocsResultNotRunOnly }] })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()

      await expect(confluencePage.utDocument).toBeVisible()
      await expect(confluencePage.utRows).toHaveCount(1)
      await expect(confluencePage.utRowResults.first()).toContainText("Not Run")
      await expect(confluencePage.utRows.first().locator("td").last()).toHaveText("")
      await expect(confluencePage.utMockTests).toBeVisible()
      await expect(confluencePage.utScreenshotStrip).toHaveCount(0)
      await expect(confluencePage.utScreenshot).toHaveCount(0)
    }
  )
})
