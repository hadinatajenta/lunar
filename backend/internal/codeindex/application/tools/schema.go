package tools

const (
	ToolNameSearchCode             = "search_code"
	ToolNameGetFileContent         = "get_file_content"
	ToolNameGetRouteDetails        = "get_route_details"
	ToolNameGetServiceContract     = "get_service_contract"
	ToolNameFindSymbolReferences   = "find_symbol_references"
	ToolNameGetServiceRoutes       = "get_service_routes"
	ToolNameGetServiceDependencies = "get_service_dependencies"
	ToolNameListServices           = "list_services"
	ToolNameGetDomainKnowledge     = "get_domain_knowledge"
	ToolNameGetGitDiff             = "get_git_diff"
	ToolNameGrepRawFallback        = "grep_raw_fallback"
)

const searchCodeDescription = "Search indexed source chunks (routes, message broker exchanges and queues, Kafka topics, HTTP client calls) across the indexed repositories using full-text search. Use concrete keywords such as an endpoint path, function name, service name, exchange or queue name, or Kafka topic."

const getFileContentDescription = "Read a bounded slice of a source file from a local repository with line numbers. Use it to read the full implementation of a handler, use case, repository, or model after locating it with search_code or get_route_details."

const getRouteDetailsDescription = "Return in-depth technical details about one route or endpoint of a service: the handler or controller file, the implementation snippet, and the downstream HTTP and messaging dependencies."

const getServiceContractDescription = "Return the complete contract summary of a service: every inbound HTTP route it exposes, its outbound HTTP dependencies, and its published or consumed messaging events."

const findSymbolReferencesDescription = "Find every usage of a symbol (function, struct, method, interface, or constant) across the indexed repositories, separating definitions from call sites."

const getServiceRoutesDescription = "List every HTTP route registered by one service."

const getServiceDependenciesDescription = "Return the dependency graph of one service: which services it calls over HTTP and which exchanges, queues, or Kafka topics it publishes or subscribes to."

const listServicesDescription = "List every indexed service with its programming language and the number of indexed code chunks."

const getDomainKnowledgeDescription = "Look up business terms, enum or status meanings, historical design decisions, business rules, and architecture context that is not present in the source code."

const getGitDiffDescription = "Read a bounded, read-only Git diff of a local repository, including branch and HEAD metadata plus the list of untracked files without reading their contents. Use it to verify whether an implementation is complete."

const grepRawFallbackDescription = "Last-resort fallback for a case-insensitive literal text search over the local repositories, used only after search_code returned no result. Restrict the pattern to a specific keyword such as a field name or constant."

const searchCodeSchema = `{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Search keyword. It can be an endpoint path, a function name, a Kafka topic, or a feature description."
    },
    "repo_filter": {
      "type": "string",
      "description": "Optional single repository name. Leave empty to search every repository."
    },
    "limit": {
      "type": "integer",
      "description": "Maximum number of results. Defaults to 8 and is capped at 20.",
      "default": 8
    }
  },
  "required": ["query"]
}`

const getFileContentSchema = `{
  "type": "object",
  "properties": {
    "repo_name": {
      "type": "string",
      "description": "Repository name, for example 'settlement-service'."
    },
    "file_path": {
      "type": "string",
      "description": "Path relative to the repository root, for example 'internal/handler/payment.go'."
    },
    "start_line": {
      "type": "integer",
      "description": "First line to read, 1-indexed. Zero or omitted starts at line 1."
    },
    "end_line": {
      "type": "integer",
      "description": "Last line to read, 1-indexed. Zero or omitted reads a window of at most 80 lines from start_line."
    }
  },
  "required": ["repo_name", "file_path"]
}`

const getRouteDetailsSchema = `{
  "type": "object",
  "properties": {
    "service_name": {
      "type": "string",
      "description": "Repository or service name, for example 'settlement-service'."
    },
    "endpoint_path": {
      "type": "string",
      "description": "HTTP endpoint path, for example '/api/v1/payments'."
    },
    "method": {
      "type": "string",
      "description": "Optional HTTP method such as GET, POST, PUT, or DELETE. Leave empty when unsure."
    }
  },
  "required": ["service_name", "endpoint_path"]
}`

const getServiceContractSchema = `{
  "type": "object",
  "properties": {
    "service_name": {
      "type": "string",
      "description": "Repository or service name whose contract should be summarized."
    }
  },
  "required": ["service_name"]
}`

const findSymbolReferencesSchema = `{
  "type": "object",
  "properties": {
    "symbol": {
      "type": "string",
      "description": "Symbol name of a function, struct, method, interface, or constant, for example 'ProcessPayment'."
    },
    "repo_filter": {
      "type": "string",
      "description": "Optional single repository name to restrict the search."
    },
    "limit": {
      "type": "integer",
      "description": "Maximum number of references per section. Defaults to 15 and is capped at 30.",
      "default": 15
    }
  },
  "required": ["symbol"]
}`

const getServiceRoutesSchema = `{
  "type": "object",
  "properties": {
    "service_name": {
      "type": "string",
      "description": "Repository or service name whose routes should be listed."
    }
  },
  "required": ["service_name"]
}`

const getServiceDependenciesSchema = `{
  "type": "object",
  "properties": {
    "service_name": {
      "type": "string",
      "description": "Repository or service name whose dependencies should be listed."
    }
  },
  "required": ["service_name"]
}`

const listServicesSchema = `{
  "type": "object",
  "properties": {}
}`

const getDomainKnowledgeSchema = `{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Business term, enum code, registration flow name, or architecture topic to look up."
    }
  },
  "required": ["query"]
}`

const grepRawFallbackSchema = `{
  "type": "object",
  "properties": {
    "pattern": {
      "type": "string",
      "description": "Literal text or specific keyword to find, for example 'app_jenis' or 'STATUS_APPROVED'."
    },
    "repo_filter": {
      "type": "string",
      "description": "Optional single repository name. Leave empty to search every repository."
    }
  },
  "required": ["pattern"]
}`

const getGitDiffSchema = `{
  "type": "object",
  "properties": {
    "repo_name": {
      "type": "string",
      "description": "Repository name to verify."
    },
    "base_ref": {
      "type": "string",
      "description": "Optional Git baseline. Defaults to HEAD for staged and unstaged changes. Use a branch, tag, or commit SHA to inspect committed changes since that baseline."
    },
    "include_uncommitted": {
      "type": "boolean",
      "description": "Defaults to true. Include staged and unstaged changes against HEAD.",
      "default": true
    },
    "file_path": {
      "type": "string",
      "description": "Optional relative path of a single file to narrow the diff."
    }
  },
  "required": ["repo_name"]
}`
