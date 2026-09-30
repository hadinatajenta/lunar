import { type Page, type Locator } from "@playwright/test"

export class KanbanBoard {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get boardRoot() {
    return this.page.getByTestId("kanban-board")
  }

  get columns() {
    return this.page.locator(".kanban-column")
  }

  get cards() {
    return this.page.locator(".kanban-card")
  }

  column(status: string): Locator {
    return this.page.locator(`[data-testid="kanban-col-${status}"]`)
  }

  columnCards(status: string): Locator {
    return this.column(status).locator(".kanban-card")
  }

  columnCount(status: string): Locator {
    return this.column(status).locator(".column-count")
  }

  loadMoreButton(status: string): Locator {
    return this.column(status).locator('[data-testid="btn-load-more"]')
  }

  emptyColumnState(status: string): Locator {
    return this.column(status).locator(".empty-column")
  }
}
