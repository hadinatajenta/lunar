import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"
import { captureEvidence } from "../../../helpers/evidence"

function getLuminance(r: number, g: number, b: number): number {
  const [rs, gs, bs] = [r, g, b].map((c) => {
    const val = c / 255
    return val <= 0.03928 ? val / 12.92 : Math.pow((val + 0.055) / 1.055, 2.4)
  })
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs
}

function getContrast(rgb1: [number, number, number], rgb2: [number, number, number]): number {
  const l1 = getLuminance(...rgb1)
  const l2 = getLuminance(...rgb2)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

function parseRgb(colorStr: string): [number, number, number] | null {
  const match = colorStr.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  if (!match) return null
  return [parseInt(match[1], 10), parseInt(match[2], 10), parseInt(match[3], 10)]
}

test.describe("AI Copilot - Visual Layout & Contrast Audit", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-VISUAL-001 verify composer layout, thought process contrast, and non-overlapping controls in light mode",
    {
      tag: ["@copilot", "@visual", "@layout", "@contrast"]
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

      await page.evaluate(() => {
        localStorage.setItem("lunar_theme", "light")
        document.documentElement.setAttribute("data-theme", "light")
      })
      await page.waitForTimeout(300)

      await copilotPage.messageTextarea.fill("Audit pull request #184 layout")
      await copilotPage.sendButton.click()

      await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 10000 })
      await page.waitForTimeout(300)
      await captureEvidence(page, "copilot", "10_copilot_resolved_layout_contrast.png")

      const layoutAudit = await page.evaluate(() => {
        const composer = document.querySelector(".chat-composer") as HTMLElement | null
        const composerActions = document.querySelector(".composer-actions") as HTMLElement | null
        const textarea = document.querySelector(".chat-composer textarea") as HTMLElement | null
        const sendBtn = document.querySelector(".send-btn") as HTMLElement | null
        const thinkingToggle = document.querySelector(".thinking-toggle") as HTMLElement | null
        const thoughtSection = document.querySelector(".thought-section") as HTMLElement | null
        const thoughtToggle = document.querySelector(".thought-toggle") as HTMLElement | null
        const thoughtDuration = document.querySelector(".thought-duration") as HTMLElement | null

        const errors: string[] = []

        if (!composer) {
          errors.push("Missing .chat-composer")
        } else {
          const cs = window.getComputedStyle(composer)
          if (cs.padding === "0px") {
            errors.push("chat-composer padding is collapsed to 0px")
          }
        }

        if (!composerActions) {
          errors.push("Missing .composer-actions")
        } else {
          const cs = window.getComputedStyle(composerActions)
          if (cs.display !== "flex") {
            errors.push(`composer-actions display is "${cs.display}", expected "flex"`)
          }
        }

        if (!thoughtToggle) {
          errors.push("Missing .thought-toggle")
        } else {
          const cs = window.getComputedStyle(thoughtToggle)
          if (cs.display !== "flex") {
            errors.push(`thought-toggle display is "${cs.display}", expected "flex"`)
          }
        }

        if (thinkingToggle) {
          const cs = window.getComputedStyle(thinkingToggle)
          if (cs.display !== "flex") {
            errors.push(`thinking-toggle display is "${cs.display}", expected "flex"`)
          }
        }

        if (textarea && sendBtn) {
          const rText = textarea.getBoundingClientRect()
          const rSend = sendBtn.getBoundingClientRect()
          const isOverlapping = !(
            rText.right < rSend.left ||
            rText.left > rSend.right ||
            rText.bottom < rSend.top ||
            rText.top > rSend.bottom
          )
          if (isOverlapping) {
            errors.push("Textarea physically overlaps with send button")
          }
          if (rSend.left < rText.left + 50) {
            errors.push(`Send button is positioned at left (${rSend.left}px) instead of right flex container`)
          }
        }

        if (thoughtDuration && thoughtSection) {
          const csDuration = window.getComputedStyle(thoughtDuration)
          const csSection = window.getComputedStyle(thoughtSection)
          return {
            errors,
            durationColor: csDuration.color,
            sectionBg: csSection.backgroundColor
          }
        }

        return {
          errors,
          durationColor: null,
          sectionBg: null
        }
      })

      expect(layoutAudit.errors).toEqual([])

      if (layoutAudit.durationColor && layoutAudit.sectionBg) {
        const fg = parseRgb(layoutAudit.durationColor)
        const bg = parseRgb(layoutAudit.sectionBg)
        if (fg && bg) {
          const ratio = getContrast(fg, bg)
          expect(ratio).toBeGreaterThanOrEqual(4.5)
        }
      }
    }
  )
})
