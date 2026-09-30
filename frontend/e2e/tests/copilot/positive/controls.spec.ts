import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

test.describe("AI Copilot - Thinking and Effort Controls", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-THINK-001 engineer can see Thinking ON toggle by default and toggle it off",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ copilotPage }) => {
      await copilotPage.navigateTo()
      await expect(copilotPage.thinkingToggle).toBeVisible()
      await expect(copilotPage.thinkingToggle).toHaveText("Thinking ON")

      await copilotPage.thinkingToggle.click()
      await expect(copilotPage.thinkingToggle).toHaveText("Thinking OFF")

      await copilotPage.thinkingToggle.click()
      await expect(copilotPage.thinkingToggle).toHaveText("Thinking ON")
    }
  )

  test(
    "CP-REASON-001 engineer can change reasoning effort level via dropdown",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ copilotPage }) => {
      await copilotPage.navigateTo()
      await expect(copilotPage.effortTrigger).toBeVisible()
      await expect(copilotPage.effortTrigger).toContainText("Effort: medium")

      await copilotPage.effortTrigger.click()
      await expect(copilotPage.effortMenu).toBeVisible()

      await copilotPage.effortOption("High effort").click()
      await expect(copilotPage.effortTrigger).toContainText("Effort: high")
    }
  )
})
