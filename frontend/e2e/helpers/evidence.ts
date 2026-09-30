import path from "path"
import fs from "fs"
import { fileURLToPath } from "url"
import type { Page } from "@playwright/test"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const QA_EVIDENCE_ROOT = path.resolve(currentDir, "../../../docs/qa/evidence")

export async function captureEvidence(
  page: Page,
  feature: "auth" | "dashboard" | "copilot" | "jira" | "bitbucket" | "confluence" | "settings",
  filename: string,
  options?: boolean | { fullPage?: boolean }
): Promise<void> {
  const targetDir = path.join(QA_EVIDENCE_ROOT, feature)
  if (!fs.existsSync(targetDir)) {
    fs.mkdirSync(targetDir, { recursive: true })
  }
  const isFullPage = typeof options === "boolean" ? options : (options?.fullPage ?? false)
  await page.screenshot({
    path: path.join(targetDir, filename),
    fullPage: isFullPage
  })
}

export function getEvidencePath(
  feature: "auth" | "dashboard" | "copilot" | "jira" | "bitbucket" | "confluence" | "settings",
  filename: string
): string {
  return path.join(QA_EVIDENCE_ROOT, feature, filename)
}
