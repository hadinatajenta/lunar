import type { ToolDomain } from "../types"

export const defaultDomains: ToolDomain[] = [
  { id: "jira", name: "Jira BRI", description: "Search issues, fetch active sprint status and assigned tickets", enabled: true },
  { id: "bitbucket", name: "Bitbucket BRI", description: "Inspect pull requests, fetch git diffs and review commits", enabled: true },
  { id: "confluence", name: "Confluence BRI", description: "Preview technical specifications and architecture documents", enabled: true },
  { id: "queryreview", name: "Query Review Assistant", description: "Extract SQL statements from query review documents", enabled: true },
  { id: "servicemap", name: "ServiceMap & Architecture", description: "Inspect workspace microservices, branches and changed files", enabled: true }
]
