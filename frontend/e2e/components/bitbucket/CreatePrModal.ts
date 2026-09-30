import { type Page } from "@playwright/test"

export class CreatePrModal {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get modalRoot() {
    return this.page.getByTestId("create-pr-modal")
  }

  get sourceBranchSelect() {
    return this.page.getByTestId("input-source-branch")
  }

  get targetBranchSelect() {
    return this.page.getByTestId("input-target-branch")
  }

  get titleInput() {
    return this.page.getByTestId("input-pr-title")
  }

  get submitButton() {
    return this.page.getByTestId("btn-submit-pr")
  }

  get toastSuccess() {
    return this.page.getByTestId("toast-create-pr-success")
  }

  async createPR(title: string) {
    await this.titleInput.fill(title)
    await this.submitButton.click()
  }
}
