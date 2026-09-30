import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

test.describe("AI Copilot - Model Selection", () => {
  test(
    "CP-MODEL-001 engineer only sees and can select configured models in model picker",
    {
      tag: ["@copilot", "@positive", "@models"]
    },
    async ({ page, copilotPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
      await routeSystemConfig(page, [])

      await copilotPage.navigateTo()

      await expect(copilotPage.modelTrigger).toBeVisible()
      await copilotPage.modelTrigger.click()
      await expect(copilotPage.modelMenu).toBeVisible()

      await expect(copilotPage.modelOption("DeepSeek-V4 Pro (Thinking)")).toBeVisible()
      await expect(copilotPage.modelOption("DeepSeek Flash")).toBeVisible()

      await expect(copilotPage.disabledModelOptions).toHaveCount(8)
      await captureEvidence(page, "copilot", "02_copilot_model_picker_filtered.png")

      await copilotPage.modelOption("DeepSeek Flash").click()
      await expect(copilotPage.modelTriggerLabel).toHaveText("DeepSeek Flash")
    }
  )
})
