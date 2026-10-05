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

export const buildWorkspaceSettings = () => ({
  source: "server",
  root_path: BOARD_ROOT_PATH,
  helper_url: "http://127.0.0.1:5199",
  has_helper_token: false,
  selected: [...BOARD_REPOSITORY_NAMES]
})

export function routeWorkspaceBoard(page: Page): Promise<unknown[]> {
  const workspaceRegistration = page.route(/\/api\/workspace$/, (route) =>
    route.fulfill({ json: buildWorkspaceSettings() })
  )
  const repositoriesRegistration = page.route(/\/api\/workspace\/repositories$/, (route) =>
    route.fulfill({
      json: {
        root_path: BOARD_ROOT_PATH,
        repositories: BOARD_REPOSITORY_NAMES.map(buildBoardRepository)
      }
    })
  )
  const servicesRegistration = page.route(/\/api\/workspace\/services$/, (route) =>
    route.fulfill({
      json: {
        root_path: BOARD_ROOT_PATH,
        services: BOARD_REPOSITORY_NAMES.map(buildBoardService)
      }
    })
  )
  return Promise.all([workspaceRegistration, repositoriesRegistration, servicesRegistration])
}
