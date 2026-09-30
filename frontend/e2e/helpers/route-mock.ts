import type { Page } from "@playwright/test"
import { mockAssignedIssues, mockBacklogResponse } from "../data/jira.data"
import { mockPushes, mockPRs, mockDiff, mockAIReview } from "../data/bitbucket.data"
import { mockConfluenceDocuments } from "../data/confluence.data"

export function routeSecrets(
  page: Page,
  options: {
    hasJiraPat?: boolean
    hasBitbucketPat?: boolean
    hasConfluencePat?: boolean
    hasAiKeys?: boolean
    configuredAiProviders?: string[]
  } = {}
) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          user_id: "test-user-id",
          has_jira_pat: options.hasJiraPat ?? true,
          jira_username: "developer",
          has_bitbucket_pat: options.hasBitbucketPat ?? true,
          bitbucket_username: "developer",
          has_confluence_pat: options.hasConfluencePat ?? true,
          has_ai_keys: options.hasAiKeys ?? true,
          configured_ai_providers: options.configuredAiProviders ?? ["deepseek"],
          updated_at: "Just now"
        })
      })
      return
    }
    await route.continue()
  })
}

export function routeSystemConfig(page: Page, systemAi: string[] = []) {
  return page.route(/\/api\/config/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        jira_base_url: "https://jira.bri.co.id",
        bitbucket_base_url: "https://bitbucket.bri.co.id",
        confluence_base_url: "https://confluence.bri.co.id",
        system_ai_providers: systemAi
      })
    })
  })
}

export function routeStandardJira(page: Page) {
  return page.route(/\/api\/jira\//, async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.includes("/issues/")) {
      const key = decodeURIComponent(url.pathname.split("/issues/")[1] || "")
      const found = mockAssignedIssues.find(
        (item) => item.key === key || item.id === key
      )
      if (found) {
        await route.fulfill({ status: 200, json: found })
        return
      }
      await route.fulfill({ status: 404, json: { error: "Issue not found" } })
      return
    }
    if (url.pathname.endsWith("/issues")) {
      await route.fulfill({ status: 200, json: mockAssignedIssues })
      return
    }
    if (url.pathname.endsWith("/backlog")) {
      await route.fulfill({ status: 200, json: mockBacklogResponse })
      return
    }
    await route.fulfill({ status: 404, json: { error: "Not found" } })
  })
}

export function routeStandardPushes(page: Page) {
  return page.route(/\/api\/bitbucket\/pushes/, async (route) => {
    const url = new URL(route.request().url())
    const filter = url.searchParams.get("filter")
    if (filter === "ready") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "ready") })
    } else if (filter === "stale") {
      await route.fulfill({ json: mockPushes.filter((p) => p.status === "stale") })
    } else {
      await route.fulfill({ json: mockPushes })
    }
  })
}

export function routeStandardPRs(page: Page) {
  return page.route(/\/api\/bitbucket\/prs/, async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() === "POST" && url.pathname.endsWith("/prs")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "#196",
          number: 196,
          repo: payload.repo,
          title: payload.title,
          source_branch: payload.source_branch,
          target_branch: payload.target_branch,
          status: "open",
          updated_relative: "Just now",
          author: "Lunar Developer",
          files_count: 2,
          lines_added: 45,
          lines_deleted: 8,
          is_assigned_to_me: false,
          is_ai_flagged: false
        }
      })
      return
    }

    if (url.pathname.includes("/diff")) {
      await route.fulfill({ json: mockDiff })
      return
    }

    if (url.pathname.includes("/ai-review")) {
      await route.fulfill({ json: mockAIReview })
      return
    }

    if (url.pathname.includes("/action")) {
      await route.fulfill({ status: 200, json: { status: "success", action: "approve", pr_id: "188" } })
      return
    }

    if (url.pathname.includes("/comments")) {
      const payload = JSON.parse(request.postData() || "{}")
      await route.fulfill({
        status: 201,
        json: {
          id: "comm-101",
          pr_id: "191",
          repo: "BRI/service-map",
          author: "Lunar Developer",
          content: payload.content,
          created_at: "Just now"
        }
      })
      return
    }

    const filter = url.searchParams.get("filter")
    if (filter === "ai") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_ai_flagged) })
    } else if (filter === "mine") {
      await route.fulfill({ json: mockPRs.filter((p) => p.is_assigned_to_me) })
    } else {
      await route.fulfill({ json: mockPRs })
    }
  })
}

export function routeStandardConfluence(page: Page, documents = mockConfluenceDocuments) {
  return page.route(/\/api\/confluence\//, async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.includes("/documents/")) {
      const id = decodeURIComponent(url.pathname.split("/documents/")[1] || "")
      const found = documents.find((d) => d.id === id)
      if (found) {
        await route.fulfill({ status: 200, json: found })
        return
      }
      await route.fulfill({ status: 404, json: { error: "Document not found" } })
      return
    }
    if (url.pathname.endsWith("/documents")) {
      await route.fulfill({ status: 200, json: documents })
      return
    }
    await route.fulfill({ status: 404, json: { error: "Not found" } })
  })
}
