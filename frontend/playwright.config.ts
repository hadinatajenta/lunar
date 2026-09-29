import { defineConfig } from "@playwright/test"
import path from "path"
import { fileURLToPath } from "url"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const AUTH_STATE_PATH = path.join(currentDir, "playwright/.auth/user.json")

export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  expect: {
    timeout: 8000
  },
  fullyParallel: true,
  workers: 4,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:5173",
    headless: true,
    viewport: { width: 1360, height: 860 },
    screenshot: "only-on-failure",
    video: "off"
  },
  projects: [
    {
      name: "setup",
      testMatch: "**/global.setup.ts"
    },
    {
      name: "authenticated",
      testIgnore: "**/global.setup.ts",
      use: {
        storageState: AUTH_STATE_PATH
      },
      dependencies: ["setup"]
    }
  ]
})
