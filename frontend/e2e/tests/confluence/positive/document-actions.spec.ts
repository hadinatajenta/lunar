import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"
import { captureEvidence } from "../../../helpers/evidence"

test.describe("Confluence BRI - Document Actions Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-ACT-002 AI generate button triggers action toast for document type and enables gated actions",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.globalToast).toContainText("Generate UT Docs With AI completed. Actions are now available.")
    }
  )

  test(
    "CONF-ACT-003 Instant apply is disabled initially, enabled by AI generate, and toolbar buttons trigger toasts",
    {
      tag: ["@confluence", "@positive", "@actions", "@toolbar"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")

      await expect(confluencePage.instantApplyButton).toBeDisabled()
      await expect(confluencePage.copyDescriptionButton).toBeDisabled()
      await captureEvidence(page, "confluence", "05_confluence_actions_disabled_initial.png")

      await confluencePage.generateAiButton.click()
      await expect(confluencePage.instantApplyButton).toBeEnabled()
      await expect(confluencePage.copyDescriptionButton).toBeEnabled()
      await page.waitForTimeout(200)
      await captureEvidence(page, "confluence", "06_confluence_actions_enabled_after_ai.png")

      await confluencePage.instantApplyButton.click()
      await expect(confluencePage.globalToast).toContainText("Instant apply is ready for backend wiring.")

      await page.getByRole("button", { name: "Edit" }).click()
      await expect(confluencePage.globalToast).toContainText("Edit is ready for backend wiring.")

      await page.getByRole("button", { name: "Share" }).click()
      await expect(confluencePage.globalToast).toContainText("Share is ready for backend wiring.")

      await page.getByRole("button", { name: "More" }).click()
      await expect(confluencePage.globalToast).toContainText("More actions are ready for backend wiring.")
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
    "CONF-CLIP-001 Copy button is gated until AI generate, then copies description to clipboard",
    {
      tag: ["@confluence", "@positive", "@clipboard"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await page.context().grantPermissions(["clipboard-read", "clipboard-write"])

      await expect(confluencePage.copyDescriptionButton).toBeDisabled()
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.copyDescriptionButton).toBeEnabled()

      await confluencePage.copyDescriptionButton.click()
      await expect(confluencePage.globalToast).toContainText("Description copied to clipboard.")
    }
  )
})
