import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"
import { mockConfluenceDocuments } from "../../../data/confluence.data"

test.describe("Confluence BRI - Negative Flows and Error Handling", () => {
  test(
    "CONF-AUTH-001 user cannot see documents when Confluence PAT is unconfigured",
    {
      tag: ["@confluence", "@negative", "@auth", "@p0"]
    },
    async ({ confluencePage, page }) => {
      let documentRequestCount = 0
      page.on("request", (req) => {
        if (req.url().includes("/api/confluence/documents")) {
          documentRequestCount++
        }
      })

      await routeSecrets(page, { hasConfluencePat: false })
      await confluencePage.navigateTo()

      await expect(confluencePage.noPatBanner).toBeVisible()
      await expect(confluencePage.documentCards).toHaveCount(0)
      expect(documentRequestCount).toBe(0)
    }
  )

  test(
    "CONF-AUTH-002 user sees 401 error banner when Confluence PAT is invalid or expired",
    {
      tag: ["@confluence", "@negative", "@auth", "@p1"]
    },
    async ({ page, confluencePage }) => {
      await routeSecrets(page, { hasConfluencePat: true })
      await page.route(/\/api\/confluence\/documents/, async (route) => {
        await route.fulfill({ status: 401 })
      })
      await confluencePage.navigateTo()

      await expect(confluencePage.errorBanner).toBeVisible()
      await expect(confluencePage.vpnBanner).not.toBeVisible()
      await expect(confluencePage.documentCards).toHaveCount(0)
    }
  )

  test(
    "CONF-NET-001 user sees 502 VPN error banner when upstream is unreachable",
    {
      tag: ["@confluence", "@negative", "@network", "@vpn", "@p1"]
    },
    async ({ page, confluencePage }) => {
      await routeSecrets(page, { hasConfluencePat: true })
      await page.route(/\/api\/confluence\/documents/, async (route) => {
        await route.fulfill({ status: 502 })
      })
      await confluencePage.navigateTo()

      await expect(confluencePage.vpnBanner).toBeVisible()
      await expect(confluencePage.errorBanner).not.toBeVisible()
      await expect(confluencePage.documentCards).toHaveCount(0)
    }
  )

  test(
    "CONF-DETAIL-002 detail deep link shows not found back button when API returns 404",
    {
      tag: ["@confluence", "@negative", "@detail", "@404"]
    },
    async ({ page, confluencePage }) => {
      await routeSecrets(page, { hasConfluencePat: true })
      await page.route(/\/api\/confluence\/documents\//, async (route) => {
        await route.fulfill({ status: 404 })
      })
      await page.goto("/confluence/NONEXISTENT")

      await expect(confluencePage.detailBackButton).toBeVisible()
      await expect(confluencePage.detailBackButton).toHaveAttribute("href", "/confluence")
    }
  )

  test(
    "CONF-EXT-002 Open in Confluence BRI shows toast when URL is missing",
    {
      tag: ["@confluence", "@negative", "@external-link"]
    },
    async ({ page, confluencePage }) => {
      const documentsWithoutUrl = mockConfluenceDocuments.map((doc) => ({
        ...doc,
        url: ""
      }))
      await routeSecrets(page, { hasConfluencePat: true })
      await routeStandardConfluence(page, documentsWithoutUrl)
      await confluencePage.navigateTo()

      await confluencePage.clickCardByTitle("DOC-401")
      await confluencePage.detailOpenConfluenceButton.click()

      await expect(confluencePage.globalToast).toContainText("This document has no Confluence URL yet.")
    }
  )
})
