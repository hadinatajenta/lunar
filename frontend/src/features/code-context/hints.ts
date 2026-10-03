import type { ToolDomain } from "./types"

export const SUPPORTED_TOOL_DOMAINS: ToolDomain[] = [
  "servicemap",
  "jira",
  "bitbucket",
  "queryreview",
  "confluence"
]

export const HELPER_OFFLINE_HINT =
  "The local helper is not connected, so this answer was produced without code context. Start the helper to receive code citations."

export const INDEX_MISSING_HINT =
  "The code index is not built yet, so this answer was produced without code context. Build the index to receive code citations."

export const INDEX_BUILDING_HINT =
  "The code index is still building, so this answer was produced without code context. Citations appear when the build finishes."

export const INDEX_EMPTY_HINT =
  "The code index holds no indexed code yet, so this answer was produced without code context. Rebuild the index to receive code citations."

export const EMPTY_ANSWER_ERROR = "The local helper returned an empty answer"

export const AGENT_FALLBACK_ERROR = "The local helper could not answer this question"

export const TRANSCRIPT_FALLBACK_ERROR = "The answer could not be saved to the chat history"

export const INDEX_STATUS_FALLBACK_ERROR = "The code index status could not be loaded"

export const INDEX_BUILD_FALLBACK_ERROR = "The code index build could not be started"
