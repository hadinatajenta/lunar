import type { Page } from "@playwright/test"

export const BOARD_ROOT_PATH = "/Users/erendt/BRI"
export const BOARD_REPOSITORY_NAMES = ["atlas", "aurora"]

const BOARD_COMMIT = {
  hash: "6a1f0c9d2b7e4f8a1c3d5e7f9a0b2c4d6e8f0a1b",
  author: "Board Fixture",
  relative_time: "2 hours ago",
  subject: "exercise the export flow"
}

const buildBoardService = (repositoryName: string) => ({
  name: repositoryName,
  branch: "master",
  updated_relative: "2 hours ago",
  commit: { ...BOARD_COMMIT },
  ahead: 0,
  behind: 0,
  files: [],
  files_truncated: false,
  error: ""
})

const buildBoardRepository = (repositoryName: string) => ({
  name: repositoryName,
  is_git: true,
  branch: "master",
  dirty_count: 0,
  updated_relative: "2 hours ago",
  error: ""
})

export const BOARD_TALL_REPOSITORY_NAMES = [
  ...BOARD_REPOSITORY_NAMES,
  "bandung",
  "cendana",
  "dahlia",
  "eclipse",
  "flamboyan",
  "garuda",
  "himalaya",
  "indigo",
  "jasmine",
  "kenanga",
  "lotus",
  "meranti"
]

export const buildWorkspaceSettings = (selected: string[] = BOARD_REPOSITORY_NAMES) => ({
  source: "server",
  root_path: BOARD_ROOT_PATH,
  helper_url: "http://127.0.0.1:5199",
  has_helper_token: false,
  selected: [...selected]
})

export const BOARD_CURRENT_BRANCH = "master"
export const BOARD_BRANCH_NAMES = [
  BOARD_CURRENT_BRANCH,
  "development",
  "staging",
  "release/v1.13.0-program-mdr",
  "feature/CRMMS-69759-maintenance-mdr",
  "feature/CRMMS-73952-inbox-cm",
  "feature/CRMMS-75734-manage-wa-notif",
  "bugfix/62302-fixing-conflict"
]

const buildBoardBranch = (name: string, index: number) => ({
  name,
  is_current: index === 0,
  updated_relative: index === 0 ? "2 hours ago" : "3 days ago"
})

export function routeWorkspaceBranches(page: Page): Promise<unknown[]> {
  const registration = page.route(/\/api\/workspace\/branches/, (route) =>
    route.fulfill({
      json: { branches: BOARD_BRANCH_NAMES.map(buildBoardBranch) }
    })
  )
  return Promise.all([registration])
}

export function routeWorkspaceBoard(
  page: Page,
  repositoryNames: string[] = BOARD_REPOSITORY_NAMES
): Promise<unknown[]> {
  const workspaceRegistration = page.route(/\/api\/workspace$/, (route) =>
    route.fulfill({ json: buildWorkspaceSettings(repositoryNames) })
  )
  const repositoriesRegistration = page.route(/\/api\/workspace\/repositories$/, (route) =>
    route.fulfill({
      json: {
        root_path: BOARD_ROOT_PATH,
        repositories: repositoryNames.map(buildBoardRepository)
      }
    })
  )
  const servicesRegistration = page.route(/\/api\/workspace\/services$/, (route) =>
    route.fulfill({
      json: {
        root_path: BOARD_ROOT_PATH,
        services: repositoryNames.map(buildBoardService)
      }
    })
  )
  return Promise.all([workspaceRegistration, repositoriesRegistration, servicesRegistration])
}
