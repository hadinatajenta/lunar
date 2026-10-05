package domain

const (
	MaxSelectedFiles    = 500
	MaxSelectedBytes    = int64(2000000)
	MaxSelectedFileSize = int64(512000)
)

type SelectedFile struct {
	RelativePath string
	Content      string
	SizeBytes    int64
}

type Selection struct {
	Service                 string
	Files                   []SelectedFile
	TotalBytes              int64
	SkippedFileCount        int
	HasApplicationDirectory bool
	HasSystemDirectory      bool
	HasPackageJSON          bool
	PackageJSONContent      string
	ExceededFileLimit       bool
	ExceededByteLimit       bool
}

func (s Selection) IsEmpty() bool {
	return len(s.Files) == 0
}

type Framework string

const (
	FrameworkUnknown   Framework = ""
	FrameworkGoNetHTTP Framework = "net/http"
	FrameworkGin       Framework = "gin"
	FrameworkFiber     Framework = "fiber"
	FrameworkEcho      Framework = "echo"
	FrameworkPHPPlain  Framework = "php"
	FrameworkLaravel   Framework = "laravel"
	FrameworkLumen     Framework = "lumen"
	FrameworkNode      Framework = "node"
)

type Classification struct {
	IsExportable bool
	Reason       string
	Framework    Framework
	UsesWire     bool
}

type Stage string

const (
	StageSelecting  Stage = "selecting"
	StagePrompt     Stage = "prompt"
	StageModel      Stage = "model"
	StageValidating Stage = "validating"
)

type StageEvent struct {
	Stage  Stage  `json:"stage"`
	Detail string `json:"detail"`
}
