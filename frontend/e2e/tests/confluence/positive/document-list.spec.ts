import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"
import { mockConfluenceDocuments } from "../../../data/confluence.data"

test.describe("Confluence BRI - Document List Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-VIEW-001 list renders stats grid, cards, visible count, and show more",
    {
      tag: ["@confluence", "@positive", "@list", "@p0"]
    },
    async ({ confluencePage }) => {
      await confluencePage.navigateTo()

      await expect(confluencePage.statCards.filter({ hasText: /UT/ })).toContainText("2")
      await expect(confluencePage.statCards.filter({ hasText: /QR/ })).toContainText("2")
      await expect(confluencePage.statCards.filter({ hasText: /SOP/ })).toContainText("1")
      await expect(confluencePage.statCards.filter({ hasText: /All/ })).toContainText("6")
      await expect(confluencePage.documentCards).toHaveCount(6)

      await expect(confluencePage.visibleCount).toContainText("Showing 6 of 6")
      await expect(confluencePage.showMoreButton).not.toBeVisible()
    }
  )

  test(
    "CONF-PAGE-001 show more reveals six more cards when total exceeds visible count",
    {
      tag: ["@confluence", "@positive", "@pagination"]
    },
    async ({ page, confluencePage }) => {
      const largeFixture = [...mockConfluenceDocuments]
      for (let i = 0; i < 8; i++) {
        largeFixture.push({
          ...mockConfluenceDocuments[i % 6],
          id: `${mockConfluenceDocuments[i % 6].id}-${i + 7}`,
          title: `${mockConfluenceDocuments[i % 6].title} (${i + 7})`
        })
      }
      await routeStandardConfluence(page, largeFixture)
      await confluencePage.navigateTo()

      await expect(confluencePage.showMoreButton).toBeVisible()
      await expect(confluencePage.visibleCount).toContainText("Showing 6 of 14")
      await confluencePage.showMoreButton.click()
      await expect(confluencePage.visibleCount).toContainText("Showing 12 of 14")
      await confluencePage.showMoreButton.click()
      await expect(confluencePage.visibleCount).toContainText("Showing 14 of 14")
      await expect(confluencePage.showMoreButton).not.toBeVisible()
    }
  )

  test(
    "CONF-SEARCH-001 search filters cards by title",
    {
      tag: ["@confluence", "@positive", "@search"]
    },
    async ({ confluencePage }) => {
      await confluencePage.navigateTo()

      await confluencePage.search("UT-101")
      await expect(confluencePage.documentCards).toHaveCount(1)
      await expect(confluencePage.documentCards.first()).toContainText("UT-101")

      await confluencePage.clearSearch()
      await expect(confluencePage.documentCards).toHaveCount(6)
    }
  )

  test(
    "CONF-FILTER-001 filter chip switches type and shows correct counts",
    {
      tag: ["@confluence", "@positive", "@filter"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()

      await page.getByTestId("doc-filter").filter({ hasText: /Query/ }).click()
      await expect(confluencePage.documentCards).toHaveCount(2)
      await expect(confluencePage.visibleCount).toContainText("Showing 2 of 2")

      await page.getByTestId("doc-filter").filter({ hasText: /Unit Test/ }).click()
      await expect(confluencePage.documentCards).toHaveCount(2)
      await expect(confluencePage.visibleCount).toContainText("Showing 2 of 2")

      await page.getByTestId("doc-filter").filter({ hasText: /SOP/ }).click()
      await expect(confluencePage.documentCards).toHaveCount(1)
      await expect(confluencePage.visibleCount).toContainText("Showing 1 of 1")

      await page.getByTestId("doc-filter").filter({ hasText: /All/ }).click()
      await expect(confluencePage.documentCards).toHaveCount(6)
      await expect(confluencePage.visibleCount).toContainText("Showing 6 of 6")
    }
  )
})
