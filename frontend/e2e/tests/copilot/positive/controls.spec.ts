import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"

const VIEWPORT_HEIGHT_RATIO = 0.4
const LENGTH_TOLERANCE_PX = 2

test.describe("AI Copilot - Composer, Model and Effort Controls", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-COMPOSER-001 Enter sends, Shift + Enter inserts a newline, and no shortcut hint or workspace header remains",
    {
      tag: ["@copilot", "@positive", "@composer"]
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

      await expect(page.locator(".chat-hint")).toHaveCount(0)
      await expect(page.locator(".chat-header")).toHaveCount(0)
      await expect(page.locator(".composer-actions")).toBeVisible()

      await copilotPage.composerInput.fill("First line")
      await copilotPage.composerInput.press("Shift+Enter")
      await copilotPage.composerInput.type("Second line")
      await expect(copilotPage.composerInput).toHaveValue("First line\nSecond line")

      await copilotPage.composerInput.press("Enter")
      await expect(copilotPage.composerInput).toHaveValue("")
      await expect(page.locator(".message.user").last()).toContainText("First line")
      await expect(page.locator(".message.user").last()).toContainText("Second line")
    }
  )

  test(
    "CP-COMPOSER-002 the composer wraps long input instead of scrolling sideways and never grows past four tenths of the viewport",
    {
      tag: ["@copilot", "@positive", "@composer"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      const initialHeight = await copilotPage.composerInput.evaluate(
        (element) => element.getBoundingClientRect().height
      )

      const longWord = "wrapping".repeat(90)
      await copilotPage.composerInput.fill(longWord)

      const metrics = await copilotPage.composerInput.evaluate((element) => {
        const style = window.getComputedStyle(element)
        return {
          height: element.getBoundingClientRect().height,
          scrollWidth: element.scrollWidth,
          clientWidth: element.clientWidth,
          maxHeight: Number.parseFloat(style.maxHeight),
          overflowY: style.overflowY,
          whiteSpace: style.whiteSpace
        }
      })

      expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + LENGTH_TOLERANCE_PX)
      expect(metrics.height).toBeGreaterThan(initialHeight)
      expect(metrics.overflowY).toBe("auto")
      expect(metrics.whiteSpace).not.toBe("nowrap")

      const expectedMaxHeight = page.viewportSize()!.height * VIEWPORT_HEIGHT_RATIO
      expect(metrics.maxHeight).toBeGreaterThan(expectedMaxHeight - LENGTH_TOLERANCE_PX)
      expect(metrics.maxHeight).toBeLessThan(expectedMaxHeight + LENGTH_TOLERANCE_PX)
    }
  )

  test(
    "CP-THINK-001 thinking mode is always on and offers no control to switch it off",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await expect(page.locator(".thinking-toggle")).toHaveCount(0)
      await expect(page.locator('[aria-label="Thinking mode"]')).toHaveCount(0)

      await page.reload()
      await expect(page.locator(".thinking-toggle")).toHaveCount(0)
    }
  )

  test(
    "CP-PICKER-MODEL-001 opening the picker shows Model and Effort, the model view searches and returns, and a choice applies",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await copilotPage.modelTrigger.click()
      await expect(copilotPage.modelRootOptionModel).toBeVisible()
      await expect(copilotPage.modelRootOptionEffort).toBeVisible()
      await expect(copilotPage.modelSearchInput).toHaveCount(0)

      await copilotPage.modelRootOptionModel.click()
      await expect(copilotPage.modelSearchInput).toBeVisible()
      await expect(copilotPage.modelBackButton).toBeVisible()
      await expect(copilotPage.modelOption("DeepSeek Flash")).toBeVisible()
      await expect(copilotPage.modelOption("DeepSeek-V4 Pro")).toBeVisible()

      await copilotPage.modelSearchInput.fill("Flash")
      await expect(copilotPage.modelOption("DeepSeek Flash")).toBeVisible()
      await expect(copilotPage.modelOption("DeepSeek-V4 Pro")).toHaveCount(0)

      await copilotPage.modelSearchInput.fill("no-such-model")
      await expect(page.locator(".model-option.is-selected")).toHaveCount(0)

      await copilotPage.modelBackButton.click()
      await expect(copilotPage.modelRootOptionModel).toBeVisible()

      await copilotPage.modelRootOptionModel.click()
      await copilotPage.modelSearchInput.fill("Flash")
      await copilotPage.modelOption("DeepSeek Flash").click()

      await expect(copilotPage.modelRootOptionModel).toBeVisible()
      await expect(copilotPage.modelTriggerLabel).toHaveText("DeepSeek Flash")
      await expect.poll(() => page.evaluate(() => localStorage.getItem("copilot_selected_model"))).toBe("DeepSeek Flash")
    }
  )

  test(
    "CP-EFFORT-001 the effort view lists Low, Med and High, and a choice applies and returns to the picker root",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await copilotPage.modelTrigger.click()
      await copilotPage.modelRootOptionEffort.click()

      await expect(copilotPage.effortOption("low")).toBeVisible()
      await expect(copilotPage.effortOption("medium")).toBeVisible()
      await expect(copilotPage.effortOption("high")).toBeVisible()
      await expect(copilotPage.effortOption("medium")).toHaveAttribute("aria-pressed", "true")

      await copilotPage.effortOption("high").click()
      await expect(copilotPage.modelRootOptionEffort).toBeVisible()
      await expect.poll(() => page.evaluate(() => localStorage.getItem("copilot_reasoning_effort"))).toBe("high")

      await copilotPage.modelRootOptionEffort.click()
      await expect(copilotPage.effortOption("high")).toHaveAttribute("aria-pressed", "true")
      await expect(copilotPage.effortOption("medium")).toHaveAttribute("aria-pressed", "false")

      await copilotPage.effortOption("low").click()
      await expect.poll(() => page.evaluate(() => localStorage.getItem("copilot_reasoning_effort"))).toBe("low")

      await page.reload()
      await copilotPage.modelTrigger.click()
      await copilotPage.modelRootOptionEffort.click()
      await expect(copilotPage.effortOption("low")).toHaveAttribute("aria-pressed", "true")
    }
  )

  test(
    "CP-PICKER-ESCAPE-001 Escape closes the picker menu from either view",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await copilotPage.modelTrigger.click()
      await expect(copilotPage.modelRootOptionModel).toBeVisible()
      await page.keyboard.press("Escape")
      await expect(copilotPage.modelRootOptionModel).toHaveCount(0)

      await copilotPage.modelTrigger.click()
      await copilotPage.modelRootOptionModel.click()
      await page.keyboard.press("Escape")
      await expect(copilotPage.modelSearchInput).toHaveCount(0)
    }
  )
})
