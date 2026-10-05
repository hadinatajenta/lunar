import { type Locator, type Page, expect } from "@playwright/test"

export function escapeRegExpPattern(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

export class MicroservicesPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get pageRoot() {
    return this.page.getByTestId("microservices-page")
  }

  get pageHeading() {
    return this.page.getByRole("heading", { level: 1, name: "Microservices BRI" })
  }

  get serviceRows() {
    return this.page.locator("tr.service-row")
  }

  get branchPopup() {
    return this.page.locator(".branch-popup")
  }

  get branchList() {
    return this.page.locator(".branch-list")
  }

  get branchSearchInput() {
    return this.page.getByRole("searchbox", { name: "Search branches" })
  }

  get actionMenu() {
    return this.page.getByRole("menu")
  }

  get actionMenuItems() {
    return this.actionMenu.getByRole("menuitem")
  }

  get noticeBanner() {
    return this.page.locator(".notice-banner")
  }

  get discardDialog() {
    return this.page.getByRole("alertdialog")
  }

  get confirmBackdrop() {
    return this.page.locator(".confirm-backdrop")
  }

  serviceRow(repositoryName: string): Locator {
    const exactName = new RegExp(`^${escapeRegExpPattern(repositoryName)}$`)
    return this.serviceRows.filter({ has: this.page.locator(".svc-title", { hasText: exactName }) })
  }

  branchPill(repositoryName: string): Locator {
    return this.serviceRow(repositoryName).locator(".branch-pill")
  }

  kebabTrigger(repositoryName: string): Locator {
    return this.serviceRow(repositoryName).locator(".kebab-trigger")
  }

  branchRows(): Locator {
    return this.branchList.locator(".branch-row")
  }

  branchRow(branchName: string): Locator {
    const exactName = new RegExp(`^${escapeRegExpPattern(branchName)}$`)
    return this.branchRows().filter({ has: this.page.locator(".branch-row-name", { hasText: exactName }) })
  }

  actionMenuItem(label: string): Locator {
    return this.actionMenu.getByRole("menuitem", { name: label, exact: true })
  }

  discardDialogButton(label: string): Locator {
    return this.discardDialog.getByRole("button", { name: label, exact: true })
  }

  expandPanel(repositoryName: string): Locator {
    const exactName = new RegExp(`^${escapeRegExpPattern(repositoryName)}$`)
    return this.page
      .locator(".expand-row")
      .filter({ has: this.page.locator(".block-count", { hasText: exactName }) })
  }

  async boundingBoxOf(locator: Locator): Promise<{ x: number; y: number; width: number; height: number }> {
    const box = await locator.boundingBox()
    if (box === null) {
      throw new Error("the element has no rendered bounding box")
    }
    return box
  }

  async navigateTo(): Promise<void> {
    await this.page.goto("/microservices")
    await expect(this.page).toHaveURL(/\/microservices/)
    await expect(this.serviceRows.first()).toBeVisible()
  }

  async openBranchPicker(repositoryName: string): Promise<void> {
    await this.branchPill(repositoryName).click()
    await this.waitForBranchList()
  }

  async waitForBranchList(): Promise<void> {
    await expect(this.branchList).toBeVisible()
  }

  async openRepositoryActions(repositoryName: string): Promise<void> {
    await this.kebabTrigger(repositoryName).click()
    await expect(this.actionMenu).toBeVisible()
  }

  async openDiscardConfirmation(repositoryName: string): Promise<void> {
    await this.openRepositoryActions(repositoryName)
    await this.actionMenuItem("Discard changes").click()
    await expect(this.discardDialog).toBeVisible()
  }

  async currentBranchName(repositoryName: string): Promise<string> {
    return this.branchPill(repositoryName).locator(".branch-pill-name").innerText()
  }

  async branchRowNames(): Promise<string[]> {
    return this.branchRows().locator(".branch-row-name").allTextContents()
  }

  async countVisibleBranchRows(): Promise<number> {
    return this.branchList.evaluate((listElement) => {
      const listRect = listElement.getBoundingClientRect()
      const rowElements = Array.from(listElement.querySelectorAll("button"))
      return rowElements.filter((rowElement) => {
        const rowRect = rowElement.getBoundingClientRect()
        return rowRect.top >= listRect.top - 1 && rowRect.bottom <= listRect.bottom + 1
      }).length
    })
  }

  async scrollToPageBottom(): Promise<void> {
    await this.page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
  }
}
