import type { Page } from "@playwright/test"
import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"
import { contrastRatio, parseColor, type RgbColor } from "../../../helpers/contrast"

type ToggleStyles = {
  iconColor: string
  iconBackground: string
  pageBackground: string
}

const TRANSITION_SETTLE_MS = 350

const readToggleStyles = (page: Page): Promise<ToggleStyles> =>
  page.evaluate(() => {
    const toggle = document.querySelector(".thinking-toggle") as HTMLElement
    const inputArea = document.querySelector(".chat-input-area") as HTMLElement
    const toggleStyle = window.getComputedStyle(toggle)
    const inputAreaStyle = window.getComputedStyle(inputArea)
    return {
      iconColor: toggleStyle.color,
      iconBackground: toggleStyle.backgroundColor,
      pageBackground: inputAreaStyle.backgroundColor
    }
  })

const requireColor = (rawColor: string, label: string): RgbColor => {
  const color = parseColor(rawColor)
  if (color === null) throw new Error(`${label} is not a measurable colour: "${rawColor}"`)
  return color
}

const readToggleContrast = async (page: Page): Promise<{ color: RgbColor; ratio: number }> => {
  await page.waitForTimeout(TRANSITION_SETTLE_MS)
  const styles = await readToggleStyles(page)
  const color = requireColor(styles.iconColor, "thinking icon colour")
  const background = requireColor(styles.iconBackground, "thinking icon background")
  const pageBackground = requireColor(styles.pageBackground, "chat input background")
  return { color, ratio: contrastRatio(color, background, pageBackground) }
}

test.describe("AI Copilot - Composer, Thinking and Effort Controls", () => {
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
    "CP-THINK-001 thinking is an icon-only brain control with an accessible name, an aria state, and a gold or grey icon that meets contrast in both themes",
    {
      tag: ["@copilot", "@positive", "@controls", "@contrast"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      const toggle = copilotPage.thinkingToggle
      await expect(toggle).toBeVisible()
      await expect(toggle).toHaveText("")
      await expect(toggle).toHaveAttribute("aria-label", "Thinking mode")
      await expect(toggle).toHaveAttribute("aria-pressed", "true")

      const darkGold = await readToggleContrast(page)
      expect(darkGold.ratio).toBeGreaterThanOrEqual(4.5)
      expect(darkGold.color.r).toBeGreaterThan(darkGold.color.b)

      await toggle.click()
      await expect(toggle).toHaveAttribute("aria-pressed", "false")

      const darkGrey = await readToggleContrast(page)
      expect(darkGrey.ratio).toBeGreaterThanOrEqual(4.5)
      expect(darkGrey.color.b).toBeGreaterThanOrEqual(darkGrey.color.r)

      await page.evaluate(() => document.documentElement.setAttribute("data-theme", "light"))

      const lightGrey = await readToggleContrast(page)
      expect(lightGrey.ratio).toBeGreaterThanOrEqual(4.5)

      await toggle.click()
      await expect(toggle).toHaveAttribute("aria-pressed", "true")

      const lightGold = await readToggleContrast(page)
      expect(lightGold.ratio).toBeGreaterThanOrEqual(4.5)
      expect(lightGold.color.r).toBeGreaterThan(lightGold.color.b)
    }
  )

  test(
    "CP-REASON-001 reasoning effort is grouped with thinking, reachable by keyboard, and inactive while thinking is off",
    {
      tag: ["@copilot", "@positive", "@controls"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()

      await expect(copilotPage.effortControl).toBeVisible()
      await expect(copilotPage.effortControl).toContainText("Reasoning effort")
      await expect(copilotPage.effortRequirement).toHaveText("Applies only while thinking is on")
      await expect(copilotPage.effortOption("Medium")).toHaveAttribute("aria-pressed", "true")

      await copilotPage.thinkingToggle.focus()
      await page.keyboard.press("Tab")
      await expect(copilotPage.effortOption("Low")).toBeFocused()
      await page.keyboard.press("Tab")
      await expect(copilotPage.effortOption("Medium")).toBeFocused()

      await copilotPage.effortOption("High").click()
      await expect(copilotPage.effortOption("High")).toHaveAttribute("aria-pressed", "true")
      await expect(copilotPage.effortOption("Medium")).toHaveAttribute("aria-pressed", "false")
      await expect.poll(() => page.evaluate(() => localStorage.getItem("copilot_reasoning_effort"))).toBe("high")

      await copilotPage.thinkingToggle.click()
      await expect(copilotPage.effortOptions).toHaveAttribute("aria-disabled", "true")
      await expect(copilotPage.effortOption("High")).toHaveAttribute("aria-disabled", "true")
      await expect.poll(() => page.evaluate(() => getComputedStyle(document.querySelector(".effort-options") as Element).opacity)).not.toBe("1")

      await copilotPage.effortOption("Low").dispatchEvent("click")
      await expect(copilotPage.effortOption("High")).toHaveAttribute("aria-pressed", "true")

      await copilotPage.thinkingToggle.click()
      await expect(copilotPage.effortOptions).toHaveAttribute("aria-disabled", "false")

      await page.reload()
      await expect(copilotPage.effortOption("High")).toHaveAttribute("aria-pressed", "true")
      await expect(copilotPage.thinkingToggle).toHaveAttribute("aria-pressed", "true")
    }
  )
})
