import { type Page, expect, type Locator } from "@playwright/test"

export class JiraPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  async goto(): Promise<void> {
    await this.page.goto("/jira")
    await expect(this.page).toHaveURL(/\/jira/)
  }

  get tabAssigned(): Locator {
    return this.page.getByTestId("tab-assigned")
  }

  get tabBacklog(): Locator {
    return this.page.getByTestId("tab-backlog")
  }

  get assignedCount(): Locator {
    return this.page.locator('[data-testid="count-assigned"], [data-testid="assigned-count"]')
  }

  get backlogCount(): Locator {
    return this.page.locator('[data-testid="count-backlog"], [data-testid="backlog-count"]')
  }

  get filterAll(): Locator {
    return this.page.getByTestId("filter-all")
  }

  get filterDocs(): Locator {
    return this.page.getByTestId("filter-docs")
  }

  get filterBugs(): Locator {
    return this.page.getByTestId("filter-bugs")
  }

  get filterSubtasks(): Locator {
    return this.page.getByTestId("filter-subtasks")
  }

  get docFilterAll(): Locator {
    return this.page.getByTestId("doc-filter-all")
  }

  get kanbanSkeleton(): Locator {
    return this.page.getByTestId("kanban-skeleton")
  }

  get backlogSkeleton(): Locator {
    return this.page.getByTestId("backlog-skeleton")
  }

  get squadFilterBar(): Locator {
    return this.page.getByTestId("squad-filter-bar")
  }

  get docFilterUt(): Locator {
    return this.page.getByTestId("doc-filter-ut")
  }

  get docFilterQuery(): Locator {
    return this.page.getByTestId("doc-filter-query")
  }

  get docFilterSop(): Locator {
    return this.page.getByTestId("doc-filter-sop")
  }

  kanbanCol(status: "open" | "progress" | "done"): Locator {
    return this.page.getByTestId(`kanban-col-${status}`)
  }

  colCount(status: "open" | "progress" | "done"): Locator {
    return this.page.getByTestId(`col-count-${status}`)
  }

  kanbanCards(status?: "open" | "progress" | "done"): Locator {
    if (status) {
      return this.kanbanCol(status).locator('[data-testid="kanban-card"]')
    }
    return this.page.locator('[data-testid="kanban-card"]')
  }

  loadMoreButton(status: "open" | "progress" | "done"): Locator {
    return this.page.getByTestId(`btn-load-more-${status}`)
  }

  emptyColumnState(status: "open" | "progress" | "done"): Locator {
    return this.page.getByTestId(`kanban-empty-${status}`)
  }

  get issueModal(): Locator {
    return this.page.getByTestId("issue-detail-modal")
  }

  get modalScrim(): Locator {
    return this.page.getByTestId("modal-scrim")
  }

  get modalKey(): Locator {
    return this.page.getByTestId("modal-issue-key")
  }

  get modalTitle(): Locator {
    return this.page.getByTestId("modal-issue-title")
  }

  get modalDesc(): Locator {
    return this.page.getByTestId("modal-issue-desc")
  }

  get btnReadMore(): Locator {
    return this.page.getByTestId("btn-read-more")
  }

  get btnCloseModal(): Locator {
    return this.page.getByTestId("btn-close-modal")
  }

  get backlogContainer(): Locator {
    return this.page.getByTestId("backlog-container")
  }

  get backlogSprints(): Locator {
    return this.page.getByTestId("backlog-sprint")
  }

  get backlogIssues(): Locator {
    return this.page.getByTestId("backlog-issue")
  }

  get noPatBanner(): Locator {
    return this.page.getByTestId("banner-no-pat")
  }

  get errorBanner(): Locator {
    return this.page.getByTestId("banner-jira-error")
  }

  get vpnBanner(): Locator {
    return this.page.getByTestId("banner-jira-vpn")
  }

  get retryButton(): Locator {
    return this.page.locator('[data-testid="btn-retry-jira"], [data-testid="btn-retry"]')
  }
}
