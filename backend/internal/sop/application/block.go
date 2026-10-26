package application

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"strconv"
	"strings"

	"lunar/backend/internal/sop/domain"
)

const (
	babIDatabaseHeading     = "BAB I. DEPLOY DATABASE MMS"
	babIIApplicationHeading = "BAB II. DEPLOY APLIKASI"
	babIIIRollbackHeading   = "BAB III. ROLLBACK APLIKASI"
	babIVRollbackHeading    = "BAB IV ROLLBACK DATABASE"

	noDatabaseChangeMessage = "Tidak ada penambahan Table/Database"
	noCreatedTableMessage   = "Tidak ada table yang dibuat"

	manualScreenshotNote = "Screenshot setiap langkah ditambahkan secara manual pada bab ini."

	bambooPlaceholder   = "(konfigurasi LUNAR_SOP_BAMBOO_MAP)"
	databasePlaceholder = "(konfigurasi LUNAR_SOP_DATABASE_MAP)"

	releaseBranch = "release/production"
)

var operasionalChildTitles = []string{
	domain.OperasionalMaintenance,
	domain.OperasionalMonitoring,
	domain.OperasionalTroubleshooting,
}

type DeployBlockInput struct {
	IssueKey    string
	Scans       []domain.RepositoryScan
	BambooPlans map[string]string
	Databases   map[string]domain.DatabaseTarget
}

type DeployBlock struct {
	DatabaseRegion string
	Middle         string
	RollbackRegion string
	DatabaseHash   string
	RollbackHash   string
	Warnings       []string
}

func (b DeployBlock) FullBody() string {
	return b.DatabaseRegion + "\n" + b.Middle + "\n" + b.RollbackRegion
}

func BuildDeployBlock(input DeployBlockInput) DeployBlock {
	warnings := make([]string, 0)

	databaseBuilder := strings.Builder{}
	databaseBuilder.WriteString(domain.DeployDatabaseMarker.Start)
	databaseBuilder.WriteString("\n")
	writeDatabaseSection(&databaseBuilder, input, &warnings)
	databaseBuilder.WriteString("\n")
	databaseBuilder.WriteString(domain.DeployDatabaseMarker.End)
	databaseRegion := databaseBuilder.String()

	middleBuilder := strings.Builder{}
	writeApplicationSection(&middleBuilder, input, &warnings)
	writeRollbackApplicationSection(&middleBuilder)
	middle := strings.TrimRight(middleBuilder.String(), "\n")

	rollbackBuilder := strings.Builder{}
	rollbackBuilder.WriteString(domain.DeployRollbackMarker.Start)
	rollbackBuilder.WriteString("\n")
	writeRollbackDatabaseSection(&rollbackBuilder, input.Scans)
	rollbackBuilder.WriteString("\n")
	rollbackBuilder.WriteString(domain.DeployRollbackMarker.End)
	rollbackRegion := rollbackBuilder.String()

	return DeployBlock{
		DatabaseRegion: databaseRegion,
		Middle:         middle,
		RollbackRegion: rollbackRegion,
		DatabaseHash:   HashBlock(databaseRegion),
		RollbackHash:   HashBlock(rollbackRegion),
		Warnings:       warnings,
	}
}

func ParseDeployBlock(body string) (DeployBlock, bool) {
	databaseRegion, databaseFound := extractRegion(body, domain.DeployDatabaseMarker)
	rollbackRegion, rollbackFound := extractRegion(body, domain.DeployRollbackMarker)
	if !databaseFound || !rollbackFound {
		return DeployBlock{}, false
	}

	middleStart := strings.Index(body, domain.DeployDatabaseMarker.End)
	middleEnd := strings.Index(body, domain.DeployRollbackMarker.Start)
	middle := ""
	if middleStart >= 0 && middleEnd > middleStart {
		middle = strings.Trim(body[middleStart+len(domain.DeployDatabaseMarker.End):middleEnd], "\n")
	}

	return DeployBlock{
		DatabaseRegion: databaseRegion,
		Middle:         middle,
		RollbackRegion: rollbackRegion,
		DatabaseHash:   HashBlock(databaseRegion),
		RollbackHash:   HashBlock(rollbackRegion),
	}, true
}

func extractRegion(storage string, marker domain.Marker) (string, bool) {
	start := strings.Index(storage, marker.Start)
	end := strings.Index(storage, marker.End)
	if start < 0 || end < start {
		return "", false
	}
	return storage[start : end+len(marker.End)], true
}

func HasRegion(storage string, marker domain.Marker) bool {
	_, found := extractRegion(storage, marker)
	return found
}

func ReplaceRegion(storage string, region string, marker domain.Marker) (string, bool) {
	start := strings.Index(storage, marker.Start)
	end := strings.Index(storage, marker.End)
	if start < 0 || end < start {
		return storage, false
	}
	return storage[:start] + region + storage[end+len(marker.End):], true
}

func PrependBlock(storage string, body string) string {
	if strings.TrimSpace(storage) == "" {
		return body
	}
	return body + "\n" + storage
}

func HashBlock(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func BuildOperasionalParentBlock() string {
	return `<ac:structured-macro ac:name="children"/>`
}

func BuildOperasionalChildBlock(title string) string {
	builder := strings.Builder{}
	builder.WriteString(domain.OperasionalMarker.Start)
	builder.WriteString("\n")
	switch title {
	case domain.OperasionalMaintenance:
		builder.WriteString(maintenanceSkeleton)
	case domain.OperasionalMonitoring:
		builder.WriteString(monitoringSkeleton)
	case domain.OperasionalTroubleshooting:
		builder.WriteString(troubleshootingSkeleton)
	}
	builder.WriteString("\n")
	builder.WriteString(domain.OperasionalMarker.End)
	return builder.String()
}

func BuildSQLCodeMacro(sql string) string {
	return `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">sql</ac:parameter><ac:plain-text-body><![CDATA[` +
		escapeCDATA(strings.TrimSpace(sql)) +
		`]]></ac:plain-text-body></ac:structured-macro>`
}

func escapeCDATA(value string) string {
	return strings.ReplaceAll(value, "]]>", "]]]]><![CDATA[>")
}

const maintenanceSkeleton = `<ac:structured-macro ac:name="toc"/>` + "\n" +
	`<p><strong>1. Backup</strong></p>` + "\n" +
	`<p><strong>2. Restore</strong></p>` + "\n" +
	`<p><strong>3. Housekeeping</strong></p>`

const monitoringSkeleton = `<p>Langkah monitoring pada OCP:</p>` + "\n" +
	"<ol>\n" +
	`<li>Buka OCP console pada environment terkait</li>` + "\n" +
	`<li>Periksa status pod pada namespace aplikasi</li>` + "\n" +
	`<li>Periksa penggunaan CPU dan memori</li>` + "\n" +
	`<li>Periksa log aplikasi</li>` + "\n" +
	"</ol>"

const troubleshootingSkeleton = `<p></p>`

func writeDatabaseSection(builder *strings.Builder, input DeployBlockInput, warnings *[]string) {
	builder.WriteString("<p><strong>" + babIDatabaseHeading + "</strong></p>\n")

	tables := databaseTables(input.Scans)
	if len(tables) == 0 {
		builder.WriteString(buildInfoMacro(noDatabaseChangeMessage))
		builder.WriteString("\n")
		return
	}

	writeDatabaseTable(builder, tables, input.Databases, warnings)
	writeDatabaseSteps(builder, input.Scans)
	writeStatementMacros(builder, input.Scans)
}

func writeDatabaseTable(builder *strings.Builder, tables []string, databases map[string]domain.DatabaseTarget, warnings *[]string) {
	builder.WriteString(`<table class="wrapped relative-table" style="width: 100%;">`)
	builder.WriteString("\n<tbody>\n")
	builder.WriteString("<tr><th>No</th><th>IP</th><th>Database</th></tr>\n")

	for index, name := range tables {
		target, exists := databases[name]
		ip := strings.TrimSpace(target.IP)
		database := strings.TrimSpace(target.Database)
		if !exists || (ip == "" && database == "") {
			ip = databasePlaceholder
			database = databasePlaceholder
			*warnings = append(*warnings, "no database host configured for "+name)
		}
		if ip == "" {
			ip = databasePlaceholder
		}
		if database == "" {
			database = databasePlaceholder
		}
		builder.WriteString("<tr><td>" + strconv.Itoa(index+1) + "</td><td>" + html.EscapeString(ip) + "</td><td>" + html.EscapeString(database) + "</td></tr>\n")
	}

	builder.WriteString("</tbody>\n</table>\n")
}

func writeDatabaseSteps(builder *strings.Builder, scans []domain.RepositoryScan) {
	builder.WriteString("<ol>\n")
	for _, scan := range scans {
		for _, file := range scan.DatabaseFiles {
			filePath := strings.TrimSpace(file.Path)
			if filePath == "" {
				continue
			}
			builder.WriteString("<li>Jalankan script <code>" + html.EscapeString(filePath) + "</code></li>\n")
		}
	}
	builder.WriteString("</ol>\n")
}

func writeStatementMacros(builder *strings.Builder, scans []domain.RepositoryScan) {
	for _, scan := range scans {
		for _, file := range scan.DatabaseFiles {
			for _, statement := range file.Statements {
				sql := strings.TrimSpace(statement.SQL)
				if sql == "" {
					continue
				}
				builder.WriteString(BuildSQLCodeMacro(sql))
				builder.WriteString("\n")
			}
		}
	}
}

func writeApplicationSection(builder *strings.Builder, input DeployBlockInput, warnings *[]string) {
	builder.WriteString("<p><strong>" + babIIApplicationHeading + "</strong></p>\n")
	builder.WriteString("<ol>\n")

	for _, scan := range input.Scans {
		repository := strings.TrimSpace(scan.Repository)
		if repository == "" {
			continue
		}

		builder.WriteString("<li>Build <strong>" + html.EscapeString(repository) + "</strong> pada Bamboo plan ")
		plan := strings.TrimSpace(input.BambooPlans[repository])
		if plan == "" {
			builder.WriteString(html.EscapeString(bambooPlaceholder))
			*warnings = append(*warnings, "no Bamboo plan configured for "+repository)
		} else {
			builder.WriteString(`<a href="` + html.EscapeString(plan) + `">` + html.EscapeString(plan) + `</a>`)
		}
		builder.WriteString(" untuk branch <code>" + html.EscapeString(releaseBranch) + "</code>")
		builder.WriteString("<br/>Deploy ke environment DC")
		builder.WriteString("<br/>Periksa pod pada OCP")
		builder.WriteString("</li>\n")
	}

	builder.WriteString("</ol>\n")
	builder.WriteString("<p>" + html.EscapeString(manualScreenshotNote) + "</p>\n")
}

func writeRollbackApplicationSection(builder *strings.Builder) {
	builder.WriteString("<p><strong>" + babIIIRollbackHeading + "</strong></p>\n")
	builder.WriteString("<ol>\n")
	builder.WriteString("<li>Pilih Bamboo plan aplikasi yang akan di-rollback</li>\n")
	builder.WriteString("<li>Pilih <strong>promote existing</strong> untuk release sebelumnya</li>\n")
	builder.WriteString("<li>Hapus CronJob yang terkait</li>\n")
	builder.WriteString("</ol>\n")
	builder.WriteString("<p>" + html.EscapeString(manualScreenshotNote) + "</p>\n")
}

func writeRollbackDatabaseSection(builder *strings.Builder, scans []domain.RepositoryScan) {
	builder.WriteString("<p><strong>" + babIVRollbackHeading + "</strong></p>\n")

	created := createdTables(scans)
	if len(created) == 0 {
		builder.WriteString(buildInfoMacro(noCreatedTableMessage))
		builder.WriteString("\n")
		return
	}

	for _, name := range created {
		builder.WriteString(BuildSQLCodeMacro("DROP TABLE IF EXISTS " + name + ";"))
		builder.WriteString("\n")
	}
}

func databaseTables(scans []domain.RepositoryScan) []string {
	names := make([]string, 0)
	seen := make(map[string]struct{})
	for _, scan := range scans {
		for _, table := range scanTables(scan) {
			name := strings.TrimSpace(table.Name)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}

func createdTables(scans []domain.RepositoryScan) []string {
	names := make([]string, 0)
	seen := make(map[string]struct{})
	for _, scan := range scans {
		for _, table := range scanTables(scan) {
			if table.Action != domain.ActionCreate {
				continue
			}
			name := strings.TrimSpace(table.Name)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}

func scanTables(scan domain.RepositoryScan) []domain.DatabaseTable {
	if len(scan.Tables) > 0 {
		return scan.Tables
	}
	return domain.MergeDatabaseTables(scan.DatabaseFiles)
}

func buildInfoMacro(message string) string {
	return `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>` + html.EscapeString(message) + `</p></ac:rich-text-body></ac:structured-macro>`
}
