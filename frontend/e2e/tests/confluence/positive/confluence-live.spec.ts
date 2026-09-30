import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

test.describe.configure({ mode: "serial" })

test.describe("Confluence BRI - Live Backend Real Data Verification", () => {
  test(
    "CONF-LIVE-001 engineer views real Confluence documents and spaces from live BRI server",
    {
      tag: ["@confluence", "@live", "@p0"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()

      await expect(confluencePage.documentCards.first()).toBeVisible({ timeout: 30000 })
      const cardCount = await confluencePage.documentCards.count()
      expect(cardCount).toBeGreaterThan(0)

      const firstCardTitle = await confluencePage.documentCards.first().locator(".doc-card-title, h3, [data-testid='doc-title']").textContent()
      expect(firstCardTitle?.length).toBeGreaterThan(0)

      await captureEvidence(page, "confluence", "01_confluence_live_overview.png", { fullPage: true })
    }
  )

  test(
    "CONF-LIVE-002 engineer can click a real Confluence document and view live details",
    {
      tag: ["@confluence", "@live", "@detail", "@p0"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await expect(confluencePage.documentCards.first()).toBeVisible({ timeout: 15000 })

      await confluencePage.clickCard(0)
      await expect(page).toHaveURL(/\/confluence\/\d+/)

      await expect(confluencePage.detailTitle).toBeVisible()
      const title = await confluencePage.detailTitle.textContent()
      expect(title?.trim().length).toBeGreaterThan(0)

      await captureEvidence(page, "confluence", "02_confluence_live_document_detail.png", { fullPage: true })

      await expect(confluencePage.detailBackButton).toBeVisible()
      await confluencePage.detailBackButton.click()
      await expect(page).toHaveURL("/confluence")
    }
  )
})
