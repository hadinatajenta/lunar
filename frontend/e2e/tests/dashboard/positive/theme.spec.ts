import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

test.describe("Global Theme Switcher", () => {
  test(
    "THEME-TOGGLE-001 user can toggle light and dark modes from the header button",
    {
      tag: ["@theme", "@positive", "@header"]
    },
    async ({ page }) => {
      await page.goto("/dashboard")
      await expect(page).toHaveURL(/\/dashboard/)

      const themeToggle = page.getByTestId("theme-toggle")
      await expect(themeToggle).toBeVisible()

      const initialTheme = await page.evaluate(() => document.documentElement.getAttribute("data-theme"))
      expect(initialTheme === "dark" || initialTheme === null).toBe(true)

      await themeToggle.click()
      await expect(page.locator("html")).toHaveAttribute("data-theme", "light")
      const lightStored = await page.evaluate(() => localStorage.getItem("lunar_theme"))
      expect(lightStored).toBe("light")

      await page.waitForFunction(() => {
        const tile = document.querySelector(".action-tile")
        return tile && getComputedStyle(tile).backgroundColor === "rgb(241, 245, 249)"
      })
      await captureEvidence(page, "dashboard", "03_dashboard_light_theme.png")

      await themeToggle.click()
      await expect(page.locator("html")).toHaveAttribute("data-theme", "dark")
      const darkStored = await page.evaluate(() => localStorage.getItem("lunar_theme"))
      expect(darkStored).toBe("dark")
    }
  )

  test(
    "THEME-TOGGLE-002 persisted theme is respected across navigation",
    {
      tag: ["@theme", "@positive", "@persistence"]
    },
    async ({ page }) => {
      await page.addInitScript(() => {
        localStorage.setItem("lunar_theme", "light")
      })

      await page.goto("/confluence")
      await expect(page.locator("html")).toHaveAttribute("data-theme", "light")

      const themeToggle = page.getByTestId("theme-toggle")
      await expect(themeToggle).toHaveAttribute("aria-label", "Switch to dark mode")
    }
  )
})
