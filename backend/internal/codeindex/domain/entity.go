package domain

type ChunkType string

const (
	ChunkTypeRoute      ChunkType = "route"
	ChunkTypeKafkaTopic ChunkType = "kafka_topic"
	ChunkTypeHTTPClient ChunkType = "http_client"
	ChunkTypeCodeBlock  ChunkType = "code_block"
	ChunkTypeEnumDef    ChunkType = "enum_definition"
	ChunkTypeDependency ChunkType = "dependency"
	ChunkTypeKnowledge  ChunkType = "knowledge"
)

type CallType string

const (
	CallTypeHTTPClient     CallType = "http_client"
	CallTypeKafkaPublish   CallType = "kafka_publish"
	CallTypeKafkaSubscribe CallType = "kafka_subscribe"
)

type CodeChunk struct {
	RepoName   string  `json:"repo_name"`
	FilePath   string  `json:"file_path"`
	ChunkType  string  `json:"chunk_type"`
	RawContent string  `json:"raw_content"`
	Content    string  `json:"-"`
	StartLine  int     `json:"start_line,omitempty"`
	EndLine    int     `json:"end_line,omitempty"`
	Score      float64 `json:"score,omitempty"`
}

type ServiceNode struct {
	RepoName   string `json:"repo_name"`
	Language   string `json:"language"`
	ChunkCount int    `json:"chunk_count"`
	IndexedAt  string `json:"indexed_at"`
}

type ServiceDependency struct {
	FromService string `json:"from_service"`
	ToService   string `json:"to_service"`
	CallType    string `json:"call_type"`
	Endpoint    string `json:"endpoint,omitempty"`
	Topic       string `json:"topic,omitempty"`
	SourceFile  string `json:"source_file,omitempty"`
}

type IndexStatus struct {
	RepoName   string `json:"repo_name"`
	Status     string `json:"status"`
	ChunkCount int    `json:"chunk_count"`
	IndexedAt  string `json:"indexed_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

type DomainKnowledgeItem struct {
	Term        string `json:"term"`
	Explanation string `json:"explanation"`
}

type GlossaryEnum struct {
	Description string            `json:"description"`
	Values      map[string]string `json:"values"`
}

type GlossaryNote struct {
	Term        string `json:"term"`
	Explanation string `json:"explanation"`
}

type Glossary struct {
	Enums map[string]GlossaryEnum `json:"enums"`
	Notes []GlossaryNote          `json:"notes"`
}

type AgentQuestion struct {
	Question string            `json:"question"`
	Model    string            `json:"model,omitempty"`
	Domains  []string          `json:"domains,omitempty"`
	History  []ChatHistoryItem `json:"history,omitempty"`
}

type ChatHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AgentAnswer struct {
	Answer      string        `json:"answer"`
	Rounds      int           `json:"rounds"`
	IsTruncated bool          `json:"is_truncated"`
	Sources     []AgentSource `json:"sources,omitempty"`
}

type AgentSource struct {
	Repo      string `json:"repo"`
	File      string `json:"file"`
	Type      string `json:"type"`
	Snippet   string `json:"snippet"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}
