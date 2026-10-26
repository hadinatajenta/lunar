package domain

const (
	ActionCreate = "CREATE"
	ActionAlter  = "ALTER"
	ActionDrop   = "DROP"
	ActionInsert = "INSERT"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

const (
	DeployDatabaseMarkerStartText = "[[lunar:sop:db:start]]"
	DeployDatabaseMarkerEndText   = "[[lunar:sop:db:end]]"
	DeployRollbackMarkerStartText = "[[lunar:sop:rollback-db:start]]"
	DeployRollbackMarkerEndText   = "[[lunar:sop:rollback-db:end]]"
	OperasionalMarkerStartText    = "[[lunar:sop-operasional:start]]"
	OperasionalMarkerEndText      = "[[lunar:sop-operasional:end]]"
)

const (
	RegionDatabase = "database"
	RegionRollback = "rollback-database"
)

const (
	DeployActionCreated         = "created"
	DeployActionUpdated         = "updated"
	DeployActionAlreadyUpToDate = "already up to date"
)

const (
	DeployModeCreate = "create"
	DeployModeUpdate = "update"
)

const (
	OperasionalMaintenance     = "Maintenance"
	OperasionalMonitoring      = "Monitoring"
	OperasionalTroubleshooting = "Troubleshooting"
)

type Marker struct {
	Start string
	End   string
}

var DeployDatabaseMarker = Marker{
	Start: `<p><em>` + DeployDatabaseMarkerStartText + `</em></p>`,
	End:   `<p><em>` + DeployDatabaseMarkerEndText + `</em></p>`,
}

var DeployRollbackMarker = Marker{
	Start: `<p><em>` + DeployRollbackMarkerStartText + `</em></p>`,
	End:   `<p><em>` + DeployRollbackMarkerEndText + `</em></p>`,
}

var OperasionalMarker = Marker{
	Start: `<p><em>` + OperasionalMarkerStartText + `</em></p>`,
	End:   `<p><em>` + OperasionalMarkerEndText + `</em></p>`,
}

type DatabaseTarget struct {
	IP       string `json:"ip"`
	Database string `json:"database"`
}

type DatabaseStatement struct {
	Action string `json:"action"`
	Table  string `json:"table"`
	SQL    string `json:"sql"`
}

type DatabaseFile struct {
	Path       string              `json:"path"`
	Statements []DatabaseStatement `json:"statements"`
}

type DatabaseTable struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type RepositoryScan struct {
	Repository    string          `json:"repository"`
	Branch        string          `json:"branch"`
	BaseRef       string          `json:"base_ref"`
	DatabaseFiles []DatabaseFile  `json:"database_files"`
	Tables        []DatabaseTable `json:"tables"`
	Warnings      []string        `json:"warnings"`
}

type ScanInput struct {
	RepositoryPath string
	Repository     string
	Branch         string
	BaseRef        string
}

type SopDeploy struct {
	Title        string `json:"title"`
	PageID       string `json:"page_id"`
	Exists       bool   `json:"exists"`
	ManagedBlock string `json:"managed_block"`
}

type SopOperasionalChild struct {
	Title  string `json:"title"`
	Exists bool   `json:"exists"`
}

type SopOperasional struct {
	Title    string                `json:"title"`
	PageID   string                `json:"page_id"`
	Exists   bool                  `json:"exists"`
	Children []SopOperasionalChild `json:"children"`
}

type GenerateRequest struct {
	Model        string           `json:"model"`
	Repositories []RepositoryScan `json:"repositories"`
}

type GenerateResult struct {
	IssueKey          string           `json:"issue_key"`
	DevelopmentPageID string           `json:"development_page_id"`
	Repositories      []RepositoryScan `json:"repositories"`
	Deploy            SopDeploy        `json:"deploy"`
	Operasional       SopOperasional   `json:"operasional"`
	Warnings          []string         `json:"warnings"`
}

type ApplyDeployRequest struct {
	Mode         string `json:"mode"`
	ManagedBlock string `json:"managed_block"`
}

type ApplyOperasionalRequest struct {
	CreateChildren bool `json:"create_children"`
}

type ApplyRequest struct {
	Deploy       ApplyDeployRequest      `json:"deploy"`
	Operasional  ApplyOperasionalRequest `json:"operasional"`
	Repositories []RepositoryScan        `json:"repositories"`
}

type DeployApplyResult struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	URL     string `json:"url"`
	Action  string `json:"action"`
}

type OperasionalApplyResult struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

type ApplyResult struct {
	DeployPage      DeployApplyResult      `json:"deploy_page"`
	OperasionalPage OperasionalApplyResult `json:"operasional_page"`
	CreatedChildren []string               `json:"created_children"`
}
