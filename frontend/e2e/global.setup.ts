import { test as setup, expect } from "@playwright/test"
import path from "path"
import { fileURLToPath } from "url"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const AUTH_STATE_PATH = path.join(currentDir, "../playwright/.auth/user.json")

setup("authenticate and persist session state", async ({ page }) => {
  await page.goto("/login")
  await page.fill("#email", "developer@lunar.dev")
  await page.fill("#password", "12345678")
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL(/\/dashboard/)
  await page.context().storageState({ path: AUTH_STATE_PATH })
})
