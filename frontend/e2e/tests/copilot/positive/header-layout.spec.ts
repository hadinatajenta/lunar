import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"
import { captureEvidence } from "../../../helpers/evidence"

const SLIM_HEADER_HEIGHT_PX = 48

test.describe("AI Copilot - Slim Contextual Header and Collapsed Main Menu", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-HEADER-001 copilot uses a slim contextual header with a breadcrumb and a single new chat button instead of the global topbar",
    {
      tag: ["@copilot", "@positive", "@header"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      const header = page.getByTestId("copilot-header")
      await expect(header).toBeVisible()

      const headerBox = await header.boundingBox()
      expect(headerBox).not.toBeNull()
      expect(Math.round(headerBox!.height)).toBe(SLIM_HEADER_HEIGHT_PX)

      await expect(page.locator(".topbar")).toHaveCount(0)
      await expect(header.locator(".breadcrumb")).toContainText("Workspace")
      await expect(header.locator(".breadcrumb")).toContainText("AI Copilot")

      const newChatButton = page.getByTestId("copilot-header-new-chat")
      await expect(newChatButton).toBeVisible()
      await expect(newChatButton).toHaveAttribute("aria-label", "New chat")

      await expect(header.locator(".theme-toggle")).toHaveCount(0)
      await expect(header.locator('[title="Chat History"]')).toHaveCount(0)

      const indexChip = page.getByTestId("code-index-status")
      await expect(header.getByTestId("code-index-status")).toHaveCount(1)
      await expect(page.locator(".chat-input-area .control-bar")).toHaveCount(0)

      const chipBox = await indexChip.boundingBox()
      expect(chipBox).not.toBeNull()
      const headerCenter = headerBox!.x + headerBox!.width / 2
      const chipCenter = chipBox!.x + chipBox!.width / 2
      expect(Math.abs(chipCenter - headerCenter)).toBeLessThanOrEqual(2)

      await captureEvidence(page, "copilot", "11_copilot_slim_header.png")
    }
  )

  test(
    "CP-HEADER-002 the header new chat button clears the conversation back to the empty state",
    {
      tag: ["@copilot", "@positive", "@header"]
    },
    async ({ page, copilotPage }) => {
      await page.route(/\/api\/copilot\/chat/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockChatSuccessResponse)
        })
      })

      await copilotPage.navigateTo()

      await copilotPage.composerInput.fill("Keep this turn")
      await copilotPage.composerSendButton.click()

      await expect(page.locator(".message.user")).toHaveCount(1)
      await expect(page.locator(".empty-title")).toHaveCount(0)

      await page.getByTestId("copilot-header-new-chat").click()

      await expect(page.locator(".empty-title")).toBeVisible()
      await expect(page.locator(".message.user")).toHaveCount(0)
    }
  )

  test(
    "CP-SIDEBAR-001 collapsing the main menu hides the Lunar brand and the user account, keeping the icon rail",
    {
      tag: ["@copilot", "@positive", "@sidebar", "@layout"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await expect(page.locator(".sidebar .brand")).toBeVisible()
      await expect(page.locator(".sidebar .account")).toBeVisible()

      await page.locator(".sidebar .collapse-btn").click()

      await expect(page.locator(".sidebar .brand")).toBeHidden()
      await expect(page.locator(".sidebar .account")).toBeHidden()
      await expect(page.locator(".sidebar .collapse-btn")).toBeVisible()
      await expect(page.locator(".sidebar .nav-item").first()).toBeVisible()

      await captureEvidence(page, "copilot", "12_copilot_collapsed_menu.png")

      await page.locator(".sidebar .collapse-btn").click()
      await expect(page.locator(".sidebar .brand")).toBeVisible()
      await expect(page.locator(".sidebar .account")).toBeVisible()
    }
  )
})
