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

      await copilotPage.modelRootOptionModel.click()
      await expect(copilotPage.modelSearchInput).toBeVisible()

      await expect(copilotPage.modelOption("DeepSeek-V4 Pro (Thinking)")).toBeVisible()
      await expect(copilotPage.modelOption("DeepSeek Flash")).toBeVisible()

      await expect(copilotPage.disabledModelOptions).toHaveCount(11)
      await captureEvidence(page, "copilot", "02_copilot_model_picker_filtered.png")

      await copilotPage.modelOption("DeepSeek Flash").click()
      await expect(copilotPage.modelTriggerLabel).toHaveText("DeepSeek Flash")
    }
  )

  test(
    "CP-MODEL-002 OpenCode Go models appear once the provider key is saved and apply when chosen",
    {
      tag: ["@copilot", "@positive", "@models", "@opencode-go"]
    },
    async ({ page, copilotPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["opencode-go"] })
      await routeSystemConfig(page, [])

      await copilotPage.navigateTo()

      await copilotPage.modelTrigger.click()
      await copilotPage.modelRootOptionModel.click()

      await expect(copilotPage.modelOption("DeepSeek V4.1 Flash (Go)")).toBeVisible()
      await expect(copilotPage.modelOption("MiMo V2.6 Flash (Go)")).toBeVisible()
      await expect(copilotPage.modelOption("Space Bunny Free (Go)")).toBeVisible()

      await copilotPage.modelOption("Space Bunny Free (Go)").click()
      await expect(copilotPage.modelTriggerLabel).toHaveText("Space Bunny Free (Go)")
      await expect
        .poll(() => page.evaluate(() => localStorage.getItem("copilot_selected_model")))
        .toBe("Space Bunny Free (Go)")
    }
  )
})
