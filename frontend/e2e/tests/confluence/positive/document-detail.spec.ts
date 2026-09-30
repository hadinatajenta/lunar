import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardConfluence } from "../../../helpers/route-mock"
import { mockConfluenceDocuments } from "../../../data/confluence.data"
import { captureEvidence } from "../../../helpers/evidence"

test.describe("Confluence BRI - Document Detail Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasConfluencePat: true })
    await routeStandardConfluence(page)
  })

  test(
    "CONF-DETAIL-001 clicking card navigates to detail with correct title, badges, and meta without description",
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
      await expect(page.getByTestId("detail-description")).toHaveCount(0)
      await expect(page.getByTestId("detail-open-confluence")).toBeVisible()

      const metaText = await page.getByTestId("detail-meta").textContent()
      expect(metaText).toContain("Engineering / Auth")
      expect(metaText).toContain("Eren")
      expect(metaText).not.toContain("Version")

      await expect(page.getByTestId("detail-last-editor")).toBeVisible()
      await expect(page.getByTestId("detail-last-editor")).toContainText("Eren")
    }
  )

  test(
    "CONF-DETAIL-003 displays last editor in meta grid, renders body container, and ensures description section is omitted",
    {
      tag: ["@confluence", "@positive", "@detail"]
    },
    async ({ page }) => {
      const mockDocWithBody = {
        ...mockConfluenceDocuments[0],
        id: "UT-BODY-01",
        last_editor: "Satria",
        body: '<div class="test-body-content"><h1>Architecture</h1><p>Test document content</p></div>'
      }

      await routeStandardConfluence(page, [mockDocWithBody])
      await page.goto("/confluence/UT-BODY-01")

      await expect(page.getByTestId("detail-last-editor")).toBeVisible()
      await expect(page.getByTestId("detail-last-editor")).toContainText("Satria")

      await expect(page.getByTestId("detail-description")).toHaveCount(0)
      await expect(page.getByTestId("detail-read-more")).toHaveCount(0)

      const bodyContainer = page.getByTestId("detail-body")
      await expect(bodyContainer).toBeVisible()
      await expect(bodyContainer.locator("h1")).toHaveText("Architecture")
      await expect(bodyContainer.locator("p")).toHaveText("Test document content")
    }
  )

  test(
    "CONF-DETAIL-004 table and long queries do not overflow canvas and sticky bar remains below header on scroll",
    {
      tag: ["@confluence", "@positive", "@detail", "@visual"]
    },
    async ({ page }) => {
      const mockDocWithLongTable = {
        ...mockConfluenceDocuments[0],
        id: "2144720692",
        title: "CRMMS-78056-[2026MMDD] Dokumen Query Review",
        description: "Dokumen query review with wide tables and queries",
        last_editor: "Hadinata Jenta",
        body: '<table class="wrapped relative-table" style="width: 131.439%;"><thead><tr><th>No</th><th>Fitur</th><th>Query Statement</th><th>Index</th><th>Capture Explain</th></tr></thead><tbody><tr><td>1</td><td>Create New User</td><td><pre>explain insert into user_data (username, password, email, nama_lengkap, jabatan, uker_main, uker_branch, uker, level_id, active_status) values (\'1234567890123456\', \'5f4dcc3b5aa765d61d8327deb882cf99\', \'user@bri.co.id\', \'Budi Santoso\', \'Staff\', \'0001\', \'0001\', \'0001\', 1, 1)</pre></td><td><code>PRIMARY</code></td><td>Execution plan data</td></tr></tbody></table><div style="height: 1200px;">Filler scroll content</div>'
      }

      await routeStandardConfluence(page, [mockDocWithLongTable])
      await page.goto("/confluence/2144720692")

      const content = page.locator(".content")
      const table = page.locator(".detail-body table")
      const preElement = page.locator(".detail-body pre")
      await expect(table).toBeVisible()
      await expect(preElement).toBeVisible()

      const contentBox = await content.boundingBox()
      const tableBox = await table.boundingBox()
      const preBox = await preElement.boundingBox()

      expect(contentBox).not.toBeNull()
      expect(tableBox).not.toBeNull()
      expect(preBox).not.toBeNull()

      if (contentBox && tableBox) {
        expect(tableBox.width).toBeLessThanOrEqual(contentBox.width + 2)
        expect(tableBox.x + tableBox.width).toBeLessThanOrEqual(contentBox.x + contentBox.width + 2)
      }

      const isPreContained = await page.evaluate(() => {
        const contentEl = document.querySelector(".content")
        const preEl = document.querySelector(".detail-body pre")
        if (!contentEl || !preEl) return false
        return preEl.getBoundingClientRect().right <= contentEl.getBoundingClientRect().right + 2
      })
      expect(isPreContained).toBe(true)

      const preWhiteSpace = await preElement.evaluate((el) => window.getComputedStyle(el).whiteSpace)
      expect(preWhiteSpace).toBe("pre-wrap")

      await captureEvidence(page, "confluence", "03_confluence_table_query_wrapped.png")

      const stickyContainer = page.locator(".detail-actions-sticky-container")
      const appHeader = page.locator("header.topbar")
      await expect(stickyContainer).toBeVisible()
      await expect(appHeader).toBeVisible()

      const stickyTopBefore = await stickyContainer.evaluate((el) => window.getComputedStyle(el).top)
      expect(stickyTopBefore).toBe("64px")

      await page.evaluate(() => window.scrollTo(0, 500))
      await page.waitForTimeout(100)

      const headerBoxAfterScroll = await appHeader.boundingBox()
      const stickyBoxAfterScroll = await stickyContainer.boundingBox()

      expect(headerBoxAfterScroll).not.toBeNull()
      expect(stickyBoxAfterScroll).not.toBeNull()

      if (headerBoxAfterScroll && stickyBoxAfterScroll) {
        expect(stickyBoxAfterScroll.y).toBeGreaterThanOrEqual(headerBoxAfterScroll.y + headerBoxAfterScroll.height - 2)
      }

      const stickyRect = await stickyContainer.evaluate((el) => el.getBoundingClientRect().top)
      expect(Math.round(stickyRect)).toBe(64)

      await captureEvidence(page, "confluence", "04_confluence_sticky_bar_below_header.png")
    }
  )

  test(
    "CONF-NAV-001 detail back button returns to list preserving state",
    {
      tag: ["@confluence", "@positive", "@navigation"]
    },
    async ({ page, confluencePage }) => {
      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-filter"][data-filter="ut"]').click()
      await page.locator('[data-testid="doc-card"]').first().click()
      await page.getByTestId("detail-back").click()

      await expect(page).toHaveURL("/confluence")
      await expect(page.locator('[data-testid="doc-filter"][data-filter="ut"]')).toHaveClass(/is-active/)
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
      await page.context().route(/confluence\.bri\.co\.id/, async (route) => {
        await route.fulfill({ status: 200, contentType: "text/html", body: "<html><body>Confluence</body></html>" })
      })

      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-card"]').filter({ hasText: "UT-101" }).click()

      const openBtn = page.getByTestId("detail-open-confluence")
      await expect(openBtn).toBeVisible()
      const [popup] = await Promise.all([
        page.waitForEvent("popup"),
        openBtn.click()
      ])
      expect(popup.url()).toContain("confluence.bri.co.id")
      await popup.close()
    }
  )

  test(
    "CONF-CACHE-001 revisit list does not refetch when already loaded",
    {
      tag: ["@confluence", "@positive", "@cache"]
    },
    async ({ page, confluencePage }) => {
      let callCount = 0
      page.on("request", (req) => {
        if (req.url().includes("/api/confluence/documents") && !req.url().includes("/api/confluence/documents/")) {
          callCount++
        }
      })

      await confluencePage.navigateTo()
      await page.locator('[data-testid="doc-card"]').filter({ hasText: "UT-101" }).click()
      await page.getByTestId("detail-back").click()

      await expect(page.locator('[data-testid="doc-card"]')).toHaveCount(6)
      expect(callCount).toBe(1)
    }
  )
})
