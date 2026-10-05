import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import {
  DEFAULT_EXPORT_STAGES,
  buildDefaultExportStream,
  buildErrorFrame,
  buildExportStream,
  buildStageFrame,
  routeCollectionExport
} from "../../../helpers/export-stream"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"
const PROVIDER_KEY_MESSAGE = "no provider key is configured for the selected model"
const NOT_EXPORTABLE_MESSAGE = "this repository does not expose HTTP routes that can be exported"
const MID_STREAM_MESSAGE = "the provider did not respond in time"

test.describe("Microservices Postman export failures", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
    await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
  })

  test(
    "MS-EXP-010 a 409 response shows the readable reason and opens no download modal",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const recorder = routeCollectionExport(page, {
        status: 409,
        responseBody: { error: PROVIDER_KEY_MESSAGE }
      })
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()

      await expect(exportPage.errorAlert).toBeVisible()
      await expect(exportPage.errorAlert).toHaveText(PROVIDER_KEY_MESSAGE)
      await expect(exportPage.downloadModal).toHaveCount(0)
      await expect(exportPage.exportButton).toHaveText("Retry export")
      expect(recorder.requests.length).toBe(1)
    }
  )

  test(
    "MS-EXP-011 a 422 response shows the reason the service cannot be exported",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page, {
        status: 422,
        responseBody: { error: NOT_EXPORTABLE_MESSAGE }
      })
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()

      await expect(exportPage.errorAlert).toBeVisible()
      await expect(exportPage.errorAlert).toHaveText(NOT_EXPORTABLE_MESSAGE)
      await expect(exportPage.downloadModal).toHaveCount(0)
      await expect(exportPage.stageRows).toHaveCount(0)
    }
  )

  test(
    "MS-EXP-012 an error frame mid-stream shows the reason and the retry button completes the export",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const failingStream = buildExportStream(
        ...DEFAULT_EXPORT_STAGES.map(buildStageFrame),
        buildErrorFrame(MID_STREAM_MESSAGE)
      )
      const recorder = routeCollectionExport(page, {
        streamBodies: [failingStream, buildDefaultExportStream()]
      })
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.startExport()

      await expect(exportPage.errorAlert).toBeVisible()
      await expect(exportPage.errorAlert).toHaveText(MID_STREAM_MESSAGE)
      await expect(exportPage.stageRows).toHaveCount(DEFAULT_EXPORT_STAGES.length)
      await expect(exportPage.downloadModal).toHaveCount(0)
      await expect(exportPage.exportButton).toHaveText("Retry export")
      await expect(exportPage.exportButton).toBeEnabled()

      await exportPage.exportButton.click()

      await expect(exportPage.downloadModal).toBeVisible()
      await expect(exportPage.downloadStatValues.first()).toHaveText("29")
      await expect.poll(() => recorder.requests.length).toBe(2)
    }
  )
})