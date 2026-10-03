import { defineConfig } from "@playwright/test"
import fs from "fs"
import path from "path"
import { fileURLToPath } from "url"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const AUTH_STATE_PATH = path.join(currentDir, "playwright/.auth/user.json")

const envCandidates = [
  path.join(currentDir, ".env"),
  path.join(currentDir, "../backend/.env"),
  path.join(currentDir, "../.env")
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

export default defineConfig({
  testDir: "./e2e/tests",
  outputDir: "./test-results",
  timeout: process.env.SLOW_MO ? 60000 : 30000,
  expect: {
    timeout: 8000
  },
  fullyParallel: true,
  workers: process.env.SLOW_MO ? 1 : (process.env.CI ? 2 : 4),
  retries: process.env.CI ? 2 : 0,
  reporter: [
    ["list"],
    ["html", { open: "never", outputFolder: "playwright-report" }],
    ["json", { outputFile: "test-results/results.json" }]
  ],
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:5174",
    headless: true,
    viewport: { width: 1360, height: 860 },
    screenshot: "only-on-failure",
    trace: "on-first-retry",
    video: "off",
    launchOptions: {
      slowMo: process.env.SLOW_MO ? parseInt(process.env.SLOW_MO, 10) : 0,
      channel: process.env.CHROME ? "chrome" : undefined,
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH || undefined
    }
  },
  projects: [
    {
      name: "setup",
      testDir: "./e2e/setup",
      testMatch: /.*\.setup\.ts/
    },
    {
      name: "unauthenticated",
      testMatch: ["**/auth/**", "**/unauthenticated-*.spec.ts"],
      use: {
        storageState: undefined
      }
    },
    {
      name: "authenticated",
      testIgnore: ["**/auth/**", "**/unauthenticated-*.spec.ts"],
      dependencies: ["setup"],
      use: {
        storageState: AUTH_STATE_PATH
      }
    }
  ]
})
