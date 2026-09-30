import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"

test.describe("Confluence BRI - Edge Cases and Empty States", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
  })

  test(
    "CONF-VIEW-002 user sees empty state when no documents exist",
    {
      tag: ["@confluence", "@edge", "@empty-state"]
    },
    async ({ page, confluencePage }) => {
      await routeStandardConfluence(page, [])
      await confluencePage.navigateTo()

      await expect(confluencePage.emptyState).toBeVisible()
      await expect(confluencePage.statCards.filter({ hasText: /All/ })).toContainText("0")
      await expect(confluencePage.documentCards).toHaveCount(0)
    }
  )

  test(
    "CONF-SEARCH-002 user sees empty state when search yields zero matches",
    {
      tag: ["@confluence", "@edge", "@search", "@empty-state"]
    },
    async ({ page, confluencePage }) => {
      await routeStandardConfluence(page)
      await confluencePage.navigateTo()

      await confluencePage.search("nonexistent-document-xyz-query")
      await expect(confluencePage.emptyState).toBeVisible()
      await expect(confluencePage.documentCards).toHaveCount(0)
    }
  )
})
