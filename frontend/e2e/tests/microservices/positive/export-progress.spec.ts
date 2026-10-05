import { test, expect } from "../../../fixtures/authenticated.fixture"
import {
  CollectionExportPage,
  EXPORT_STAGE_DETAILS,
  EXPORT_STAGE_LABELS
} from "../../../pages/CollectionExportPage"
import {
  DEFAULT_EXPORT_STAGES,
  DEFAULT_FOLDER_COUNT,
  DEFAULT_REQUEST_COUNT,
  buildDefaultResultFrame,
  buildStageFrame,
  installStreamingExport,
  routeCollectionExport
} from "../../../helpers/export-stream"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { BOARD_ROOT_PATH, routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"
const DEFAULT_SELECTED_MODEL_ID = "opencode-go-deepseek-v4.1-flash"
const WORKSPACE_ROOT = BOARD_ROOT_PATH
const STAGE_FRAME_DELAY_MS = 1000
const RESULT_FRAME_DELAY_MS = 3000

const buildStageChunks = () => [
  { delayMs: 0, body: buildStageFrame(DEFAULT_EXPORT_STAGES[0]) },
  { delayMs: STAGE_FRAME_DELAY_MS, body: buildStageFrame(DEFAULT_EXPORT_STAGES[1]) },
  { delayMs: STAGE_FRAME_DELAY_MS, body: buildStageFrame(DEFAULT_EXPORT_STAGES[2]) },
  { delayMs: STAGE_FRAME_DELAY_MS, body: buildStageFrame(DEFAULT_EXPORT_STAGES[3]) },
  { delayMs: RESULT_FRAME_DELAY_MS, body: buildDefaultResultFrame() }
]

test.describe("Microservices Postman export progress", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
    await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
  })

  test(
    "MS-EXP-004 every stage is rendered in the order the frames arrive",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      installStreamingExport(page, { chunks: buildStageChunks() })
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()

      await expect(exportPage.stageRows).toHaveCount(1)
      await expect(exportPage.stageNames.first()).toHaveText(EXPORT_STAGE_LABELS[0])
      await expect(exportPage.stageRows).toHaveCount(2)
      await expect(exportPage.stageRows).toHaveCount(3)
      await expect(exportPage.stageRows).toHaveCount(4)
      await expect(exportPage.stageNames).toHaveText(EXPORT_STAGE_LABELS)
      await expect(exportPage.stageDetails).toHaveText(EXPORT_STAGE_DETAILS)
      await expect(exportPage.downloadModal).toBeHidden()

      await expect(exportPage.downloadModal).toBeVisible()
    }
  )

  test(
    "MS-EXP-005 the export request carries the workspace root, name, model and session and no effort field",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const recorder = routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()
      await expect.poll(() => recorder.requests.length).toBe(1)

      const request = recorder.requests[0]
      expect(request.name).toBe(REPOSITORY_NAME)
      expect(request.root).toBe(WORKSPACE_ROOT)
      expect(request.model).toBe(DEFAULT_SELECTED_MODEL_ID)
      expect(request.fieldNames).toEqual(["backend_url", "model", "name", "root", "session_token"])
      expect(request.backendUrl).toMatch(/^https?:\/\//)
      expect(request.sessionToken.length).toBeGreaterThan(0)
      expect(JSON.stringify(request.fieldNames)).not.toMatch(/effort/i)
      expect(recorder.countFor(REPOSITORY_NAME)).toBe(1)
    }
  )

  test(
    "MS-EXP-006 the result opens the download modal with the request and folder counts",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()

      await expect(exportPage.downloadModal).toBeVisible()
      await expect(exportPage.downloadModal).toContainText("Your Postman collection is ready")
      await expect(exportPage.downloadModal).toContainText("Requests")
      await expect(exportPage.downloadModal).toContainText("Folders")
      await expect(exportPage.downloadStatValues).toHaveText([
        String(DEFAULT_REQUEST_COUNT),
        String(DEFAULT_FOLDER_COUNT)
      ])
      await expect(exportPage.downloadFileName).toHaveText("atlas-postman-collection.json")
      await expect(exportPage.modelPicker).toHaveCount(0)
    }
  )
})