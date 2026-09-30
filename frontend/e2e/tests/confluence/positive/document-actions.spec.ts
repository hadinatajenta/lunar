import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"

test.describe("Confluence BRI - Document Actions Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-ACT-001 New page button triggers placeholder toast",
    {
      tag: ["@confluence", "@positive", "@actions", "@p1"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await page.getByRole("button", { name: "New page" }).click()
      await expect(confluencePage.globalToast).toContainText("New page is ready for backend wiring.")
    }
  )

  test(
    "CONF-ACT-002 AI generate button triggers action toast for document type",
    {
      tag: ["@confluence", "@positive", "@actions", "@ai"]
    },
    async ({ confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await confluencePage.generateAiButton.click()
      await expect(confluencePage.globalToast).toContainText("AI generation is ready for backend wiring.")
    }
  )

  test(
    "CONF-ACT-003 Instant apply, Edit, Share, and More buttons trigger placeholder toasts",
    {
      tag: ["@confluence", "@positive", "@actions", "@toolbar"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")

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

      await confluencePage.commentPostButton.click()
      await expect(confluencePage.commentInput).toBeFocused()
    }
  )

  test(
    "CONF-CLIP-001 Copy button copies description to clipboard on non-SOP doc",
    {
      tag: ["@confluence", "@positive", "@clipboard"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await confluencePage.clickCardByTitle("UT-101")
      await page.context().grantPermissions(["clipboard-read", "clipboard-write"])
      await confluencePage.copyDescriptionButton.click()
      await expect(confluencePage.globalToast).toContainText("Description copied to clipboard.")
    }
  )
})
