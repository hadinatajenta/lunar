import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

test.describe("AI Copilot - Access Gate", () => {
  test(
    "CP-GATE-001 engineer cannot access Copilot chat when no AI provider API key is configured",
    {
      tag: ["@copilot", "@negative", "@authorization", "@p0"]
    },
    async ({ page, copilotPage }) => {
      await routeSecrets(page, { hasAiKeys: false, configuredAiProviders: [] })
      await routeSystemConfig(page, [])

      await copilotPage.navigateTo()

      await expect(copilotPage.noAiKeysGate).toBeVisible()
      await expect(copilotPage.noAiKeysGate).toContainText("AI Provider API Key Required")
      await expect(copilotPage.goToSettingsButton).toBeVisible()
      await captureEvidence(page, "copilot", "01_copilot_gate_unconfigured.png")

      await copilotPage.goToSettingsButton.click()
      await expect(page).toHaveURL(/\/settings/)
    }
  )
})
