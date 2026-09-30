import { type Page, type Locator } from "@playwright/test"

export class BacklogList {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get backlogRoot() {
    return this.page.getByTestId("backlog-view")
  }

  get sprintSections() {
    return this.page.locator(".sprint-section")
  }

  get backlogIssues() {
    return this.page.locator(".backlog-issue-row")
  }

  sprint(id: string): Locator {
    return this.page.locator(`[data-testid="sprint-${id}"]`)
  }
}
