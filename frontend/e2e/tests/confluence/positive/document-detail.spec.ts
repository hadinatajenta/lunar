import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"

test.describe("Confluence BRI - Document Detail Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-DETAIL-001 clicking card navigates to detail with correct title, badges, description, and meta",
    {
      tag: ["@confluence", "@positive", "@detail", "@p0"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-card"]').filter({ hasText: "UT-101" }).click()
      await expect(page).toHaveURL(/\/confluence\/UT-101/)

      await expect(page.getByTestId("detail-title")).toContainText("UT-101 User Authentication Flow")
      await expect(page.getByTestId("detail-type-badge")).toContainText("UT")
      await expect(page.getByTestId("detail-status")).toContainText("Open")
      await expect(page.getByTestId("detail-description")).toContainText("Unit test specification")
      await expect(page.getByTestId("detail-open-confluence")).toBeVisible()

      const metaText = await page.getByTestId("detail-meta").textContent()
      expect(metaText).toContain("Engineering / Auth")
      expect(metaText).toContain("Eren")
      expect(metaText).not.toContain("Version")
    }
  )

  test(
    "CONF-NAV-001 detail back button returns to list preserving state",
    {
      tag: ["@confluence", "@positive", "@navigation"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await page.getByTestId("doc-filter").filter({ hasText: /UT/ }).click()
      await page.locator('[data-testid="doc-card"]').first().click()
      await page.getByTestId("detail-back").click()

      await expect(page).toHaveURL("/confluence")
      await expect(page.locator('[data-testid="doc-filter"]').filter({ hasText: /UT/ })).toHaveAttribute("aria-current", "true")
    }
  )

  test(
    "CONF-NAV-002 deep link loads detail directly without list loaded first",
    {
      tag: ["@confluence", "@positive", "@navigation"]
    },
    async ({ page }) => {
      await page.goto("/confluence/QR-202")
      await expect(page).toHaveURL(/\/confluence\/QR-202/)
      await expect(page.getByTestId("detail-title")).toContainText("QR-202 Index Suggestion")
    }
  )

  test(
    "CONF-EXT-001 Open in Confluence BRI opens the document URL when present",
    {
      tag: ["@confluence", "@positive", "@external-link"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-card"]').filter({ hasText: "UT-101" }).click()

      const openBtn = page.getByTestId("detail-open-confluence")
      await expect(openBtn).toHaveAttribute("href", "https://confluence.bri.co.id/pages/viewpage.action?pageId=UT-101")
      await expect(openBtn).toHaveAttribute("target", "_blank")
      await expect(openBtn).toHaveAttribute("rel", /noopener/)
    }
  )

  test(
    "CONF-CACHE-001 revisit list does not refetch when already loaded",
    {
      tag: ["@confluence", "@positive", "@cache"]
    },
    async ({ page, confluencePage }) => {
      let callCount = 0
      await page.route(/\/api\/confluence\/documents$/, async (route) => {
        callCount++
        await route.continue()
      })

      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-card"]').filter({ hasText: "UT-101" }).click()
      await page.getByTestId("detail-back").click()

      await expect(page.locator('[data-testid="doc-card"]')).toHaveCount(6)
      expect(callCount).toBe(1)
    }
  )
})
