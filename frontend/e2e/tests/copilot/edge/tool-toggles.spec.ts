import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

test.describe("AI Copilot - Active Tool Toggles", () => {
  test(
    "CP-TOOL-001 engineer can open Active Copilot Tools modal and toggle a tool",
    {
      tag: ["@copilot", "@edge", "@tools"]
    },
    async ({ page, copilotPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
      await routeSystemConfig(page, [])

      await copilotPage.navigateTo()
      await copilotPage.settingsGearButton.click()
      await expect(copilotPage.activeToolsModalTitle).toHaveText("Active Copilot Tools")

      await copilotPage.domainRow("Query Review Assistant").click()
      await copilotPage.doneButton.click()
      await expect(copilotPage.activeToolsModalTitle).not.toBeVisible()
    }
  )
})
