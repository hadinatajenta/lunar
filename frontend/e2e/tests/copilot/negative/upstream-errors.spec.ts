import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

test.describe("AI Copilot - Upstream Provider Errors", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-ERR-001 engineer receives clear error message when upstream AI provider returns 401 Unauthorized",
    {
      tag: ["@copilot", "@negative", "@authorization"]
    },
    async ({ page, copilotPage }) => {
      await page.route(/\/api\/copilot\/chat/, async (route) => {
        await route.fulfill({
          status: 401,
          contentType: "application/json",
          body: JSON.stringify({ error: "unauthorized: invalid or expired deepseek API key" })
        })
      })

      await copilotPage.navigateTo()
      await copilotPage.messageTextarea.fill("Review database migration indexes")
      await copilotPage.sendButton.click()

      await expect(copilotPage.lastAssistantMessage).toBeVisible({ timeout: 8000 })
      await expect(copilotPage.lastAssistantMessage).toContainText("invalid or expired deepseek API key")
      await captureEvidence(page, "copilot", "03_copilot_error_401_invalid_key.png")
    }
  )

  test(
    "CP-ERR-002 engineer receives clear error message when upstream AI provider returns 502 Bad Gateway",
    {
      tag: ["@copilot", "@negative", "@network"]
    },
    async ({ page, copilotPage }) => {
      await page.route(/\/api\/copilot\/chat/, async (route) => {
        await route.fulfill({
          status: 502,
          contentType: "application/json",
          body: JSON.stringify({ error: "failed to contact AI provider (deepseek): connection timeout" })
        })
      })

      await copilotPage.navigateTo()
      await copilotPage.messageTextarea.fill("Analyze service dependencies")
      await copilotPage.sendButton.click()

      await expect(copilotPage.lastAssistantMessage).toBeVisible({ timeout: 8000 })
      await expect(copilotPage.lastAssistantMessage).toContainText("connection timeout")
      await captureEvidence(page, "copilot", "04_copilot_error_502_timeout.png")
    }
  )
})
