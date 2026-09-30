import { type Page } from "@playwright/test"

export class PrTable {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get tableRoot() {
    return this.page.getByTestId("pr-table")
  }

  get rows() {
    return this.page.locator("table.pr-table tbody tr")
  }

  get emptyNotice() {
    return this.page.getByTestId("empty-prs-notice")
  }
}
