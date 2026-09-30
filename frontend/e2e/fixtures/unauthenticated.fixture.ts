import { test as base, expect } from "@playwright/test"
import { AuthPage } from "../pages/AuthPage"
import { RegisterPage } from "../pages/RegisterPage"
import { DashboardPage } from "../pages/DashboardPage"
import { CopilotPage } from "../pages/CopilotPage"
import { JiraPage } from "../pages/JiraPage"
import { BitbucketPage } from "../pages/BitbucketPage"
import { ConfluencePage } from "../pages/ConfluencePage"
import { SettingsPage } from "../pages/SettingsPage"

type UnauthFixtures = {
  authPage: AuthPage
  registerPage: RegisterPage
  dashboardPage: DashboardPage
  copilotPage: CopilotPage
  jiraPage: JiraPage
  bitbucketPage: BitbucketPage
  confluencePage: ConfluencePage
  settingsPage: SettingsPage
}

export const test = base.extend<UnauthFixtures>({
  storageState: async ({}, use) => {
    await use({ cookies: [], origins: [] })
  },
  authPage: async ({ page }, use) => {
    await use(new AuthPage(page))
  },
  registerPage: async ({ page }, use) => {
    await use(new RegisterPage(page))
  },
  dashboardPage: async ({ page }, use) => {
    await use(new DashboardPage(page))
  },
  copilotPage: async ({ page }, use) => {
    await use(new CopilotPage(page))
  },
  jiraPage: async ({ page }, use) => {
    await use(new JiraPage(page))
  },
  bitbucketPage: async ({ page }, use) => {
    await use(new BitbucketPage(page))
  },
  confluencePage: async ({ page }, use) => {
    await use(new ConfluencePage(page))
  },
  settingsPage: async ({ page }, use) => {
    await use(new SettingsPage(page))
  }
})

export { expect }
