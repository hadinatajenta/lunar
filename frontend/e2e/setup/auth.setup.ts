import { test as setup, expect } from "@playwright/test"
import fs from "fs"
import path from "path"
import { fileURLToPath } from "url"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const AUTH_STATE_PATH = path.join(currentDir, "../../playwright/.auth/user.json")

const envCandidates = [
  path.join(currentDir, "../../.env"),
  path.join(currentDir, "../../../backend/.env"),
  path.join(currentDir, "../../../.env")
]

for (const envCandidate of envCandidates) {
  if (fs.existsSync(envCandidate) && typeof process.loadEnvFile === "function") {
    try {
      process.loadEnvFile(envCandidate)
    } catch {
      continue
    }
  }
}

const email = process.env.E2E_USER_EMAIL || process.env.SEED_USER_EMAIL || "developer@lunar.dev"
const password = process.env.E2E_USER_PASSWORD || process.env.SEED_USER_PASSWORD || "12345678"

setup("authenticate and persist session state", async ({ page }) => {
  fs.mkdirSync(path.dirname(AUTH_STATE_PATH), { recursive: true })
  await page.goto("/login")
  await page.fill("#email", email)
  await page.fill("#password", password)
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL(/\/dashboard/)
  await page.context().storageState({ path: AUTH_STATE_PATH })
})
