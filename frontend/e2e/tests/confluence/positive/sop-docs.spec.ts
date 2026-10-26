import { test, expect } from "../../../fixtures/authenticated.fixture"
import {
  captureClipboardText,
  routeCurrentUser,
  routeHelperWorkspace,
  routeSecrets,
  routeSopApply,
  routeSopGenerate,
  routeSopHelper,
  routeStandardConfluence
} from "../../../helpers/route-mock"
import {
  buildMockSopScanResult,
  mockSopApplyConflict,
  mockSopApplyResult,
  mockSopDocument,
  mockSopDocumentWithoutIssueKey,
  mockSopGenerateResultScanned
} from "../../../data/confluence.data"

const SOP_ROOT = "/Users/erendt/BRI"

test.describe("Confluence BRI - SOP Deploy and Operasional", () => {
  test.beforeEach(async ({ page }) => {
    await routeCurrentUser(page)
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page, [mockSopDocument])
  })

  test(
    "CONF-SOP-001 the SOP button generates the deploy preview, the operasional children, the warnings, and the copy output",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      const helper = routeSopHelper(page)
      const generate = routeSopGenerate(page)
      await captureClipboardText(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("CRMMS-78064")

      await expect(confluencePage.sopGenerateButton).toBeVisible()
      await expect(confluencePage.sopGenerateButton).toContainText("Generate SOP Deploy & Operasional")
      await expect(confluencePage.sopRoot).toContainText(SOP_ROOT)

      await confluencePage.sopGenerateButton.click()

      await expect(confluencePage.sopResult).toBeVisible()
      await expect(confluencePage.sopIssueKey).toContainText("CRMMS-78064")

      await expect(confluencePage.sopWarnings).toContainText("no Bamboo plan configured for everest")

      await expect(confluencePage.sopDeployTitle).toContainText("CRMMS-78064-SOP Deploy")
      await expect(confluencePage.sopDeployExists).toContainText("Exists")
      await expect(confluencePage.sopDeployPreview).toContainText("BAB I. DEPLOY DATABASE MMS")
      await expect(confluencePage.sopDeployPreview).toContainText("log_program_mdr")
      await expect(confluencePage.sopDeployPreview).not.toContainText(
        "Tidak ada penambahan Table/Database"
      )
      await expect(confluencePage.sopDeployPreview).toContainText("Build everest on release/production")
      await expect(confluencePage.sopDeployPreview.locator("strong")).toHaveCount(0)

      await expect(confluencePage.sopOperasionalTitle).toContainText("CRMMS-78064-SOP Operasional")
      await expect(confluencePage.sopChildren).toHaveCount(3)
      await expect(confluencePage.sopChildStates).toHaveText(["Exists", "Missing", "Missing"])

      await expect(confluencePage.sopRepositories).toHaveCount(2)
      await expect(confluencePage.sopRepositoryNames.nth(0)).toContainText("everest")

      expect(generate.requests).toHaveLength(2)
      expect(JSON.parse(generate.requests[0])).toEqual({ model: "DeepSeek-V4 Pro (Thinking)" })
      const finalGenerate = JSON.parse(generate.requests[1])
      expect(finalGenerate.model).toBe("DeepSeek-V4 Pro (Thinking)")
      expect(finalGenerate.repositories).toEqual([
        buildMockSopScanResult("everest"),
        buildMockSopScanResult("fe-mms")
      ])

      expect(helper.scanRequests).toHaveLength(2)
      expect(JSON.parse(helper.scanRequests[0])).toEqual({
        root: SOP_ROOT,
        repo: "everest",
        branch: "feature/CRMMS-77914",
        base_ref: "master"
      })
      expect(JSON.parse(helper.scanRequests[1])).toEqual({
        root: SOP_ROOT,
        repo: "fe-mms",
        branch: "feature/CRMMS-77914"
      })

      await confluencePage.sopCopyDeployButton.click()
      await expect(confluencePage.globalToast).toContainText("SOP Deploy content copied.")
      await confluencePage.sopCopyOperasionalButton.click()
      await expect(confluencePage.globalToast).toContainText("SOP Operasional plan copied.")

      const copied = await page.evaluate(
        () =>
          (window as unknown as { __capturedClipboardText: string[] }).__capturedClipboardText
      )
      expect(copied[0]).toContain("BAB I. DEPLOY DATABASE MMS")
      expect(copied[0]).not.toContain("<strong>")
      expect(copied[1]).toContain("[x] Maintenance")
      expect(copied[1]).toContain("[ ] Monitoring")
    }
  )

  test(
    "CONF-SOP-002 applying the SOP writes the update mode and toasts the deploy version and action",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeSopHelper(page)
      await routeSopGenerate(page)
      const apply = routeSopApply(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("CRMMS-78064")
      await expect(confluencePage.sopRoot).toContainText(SOP_ROOT)
      await confluencePage.sopGenerateButton.click()
      await expect(confluencePage.sopResult).toBeVisible()

      await confluencePage.sopApplyButton.click()

      await expect(confluencePage.globalToast).toContainText("version 14")
      await expect(confluencePage.globalToast).toContainText("updated")
      await expect(confluencePage.sopApplied).toContainText("updated at version 14")

      expect(apply.requests).toHaveLength(1)
      expect(JSON.parse(apply.requests[0])).toEqual({
        deploy: { mode: "update", managed_block: mockSopGenerateResultScanned.deploy.managed_block },
        operasional: { create_children: true }
      })
    }
  )

  test(
    "CONF-SOP-003 a failed SOP apply shows an in-panel error and the retry succeeds",
    {
      tag: ["@confluence", "@negative", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeSopHelper(page)
      await routeSopGenerate(page)
      const apply = routeSopApply(page, {
        responses: [
          { status: 502, body: mockSopApplyConflict },
          { status: 200, body: mockSopApplyResult }
        ]
      })

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("CRMMS-78064")
      await expect(confluencePage.sopRoot).toContainText(SOP_ROOT)
      await confluencePage.sopGenerateButton.click()
      await expect(confluencePage.sopResult).toBeVisible()

      await confluencePage.sopApplyButton.click()
      await expect(confluencePage.sopApplyError).toContainText("updated elsewhere")

      await confluencePage.sopApplyRetryButton.click()
      await expect(confluencePage.globalToast).toContainText("version 14")
      expect(apply.requests).toHaveLength(2)
    }
  )

  test(
    "CONF-SOP-004 the SOP button is absent when the title carries no issue key",
    {
      tag: ["@confluence", "@edge", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await page.unroute(/\/api\/confluence\//)
      await routeStandardConfluence(page, [mockSopDocumentWithoutIssueKey])

      await page.goto("/confluence/2144721999")

      await expect(confluencePage.detailTitle).toContainText("Architecture Overview")
      await expect(confluencePage.sopGenerateButton).toHaveCount(0)
      await expect(confluencePage.sopPanelHeading).toHaveCount(0)
    }
  )

  test(
    "CONF-SOP-005 a helper scan failure shows the in-panel error with a retry",
    {
      tag: ["@confluence", "@negative", "@actions", "@ai"]
    },
    async ({ page, confluencePage }) => {
      await routeHelperWorkspace(page)
      await routeSopHelper(page, { scanStatus: 500 })
      await routeSopGenerate(page)

      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("CRMMS-78064")
      await expect(confluencePage.sopRoot).toContainText(SOP_ROOT)
      await confluencePage.sopGenerateButton.click()

      await expect(confluencePage.sopError).toContainText("could not scan the repository")
      await expect(confluencePage.sopRetryButton).toBeVisible()
      await expect(confluencePage.sopResult).toHaveCount(0)
    }
  )
})
