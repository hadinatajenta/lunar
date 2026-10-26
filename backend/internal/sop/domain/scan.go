package domain

import (
	"path"
	"regexp"
	"strings"
)

const (
	maxStatementLength = 4000
	maxExtractedTables = 500
)

var (
	sqlLineCommentPattern  = regexp.MustCompile(`--[^\n]*`)
	sqlBlockCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)

	createTablePattern = regexp.MustCompile(`(?is)^CREATE\s+(?:OR\s+REPLACE\s+)?(?:TEMPORARY\s+|TEMP\s+|UNLOGGED\s+|GLOBAL\s+|LOCAL\s+)*TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z0-9_."$]+)`)
	alterTablePattern  = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?([A-Za-z0-9_."$]+)`)
	dropTablePattern   = regexp.MustCompile(`(?is)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([A-Za-z0-9_."$]+)`)
	insertIntoPattern  = regexp.MustCompile(`(?is)^INSERT\s+INTO\s+([A-Za-z0-9_."$]+)`)
	updatePattern      = regexp.MustCompile(`(?is)^UPDATE\s+([A-Za-z0-9_."$]+)`)
	deleteFromPattern  = regexp.MustCompile(`(?is)^DELETE\s+FROM\s+([A-Za-z0-9_."$]+)`)
)

var databaseActionPatterns = []struct {
	action  string
	pattern *regexp.Regexp
}{
	{ActionCreate, createTablePattern},
	{ActionAlter, alterTablePattern},
	{ActionDrop, dropTablePattern},
	{ActionInsert, insertIntoPattern},
	{ActionUpdate, updatePattern},
	{ActionDelete, deleteFromPattern},
}

var databaseDirectorySegments = []string{"migrations", "migration", "database/migrations"}

func ExtractDatabaseStatements(content string) []DatabaseStatement {
	cleaned := sqlLineCommentPattern.ReplaceAllString(content, "")
	cleaned = sqlBlockCommentPattern.ReplaceAllString(cleaned, "")

	statements := make([]DatabaseStatement, 0)
	for _, rawStatement := range strings.Split(cleaned, ";") {
		statement := strings.TrimSpace(rawStatement)
		if statement == "" {
			continue
		}
		action, table, ok := classifyStatement(statement)
		if !ok {
			continue
		}
		statements = append(statements, DatabaseStatement{
			Action: action,
			Table:  table,
			SQL:    boundStatement(statement),
		})
	}
	return statements
}

func classifyStatement(statement string) (string, string, bool) {
	for _, candidate := range databaseActionPatterns {
		match := candidate.pattern.FindStringSubmatch(statement)
		if len(match) < 2 {
			continue
		}
		table := normalizeTableName(match[1])
		if table == "" {
			continue
		}
		return candidate.action, table, true
	}
	return "", "", false
}

func normalizeTableName(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}

func boundStatement(statement string) string {
	runes := []rune(statement)
	if len(runes) <= maxStatementLength {
		return statement
	}
	return string(runes[:maxStatementLength])
}

func MergeDatabaseTables(files []DatabaseFile) []DatabaseTable {
	tables := make([]DatabaseTable, 0)
	seen := make(map[string]struct{})
	for _, file := range files {
		for _, statement := range file.Statements {
			key := statement.Action + "\x1f" + statement.Table
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			tables = append(tables, DatabaseTable{Name: statement.Table, Action: statement.Action})
			if len(tables) >= maxExtractedTables {
				return tables
			}
		}
	}
	return tables
}

func IsDatabaseFilePath(value string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if normalized == "" {
		return false
	}
	if strings.HasSuffix(strings.ToLower(path.Base(normalized)), ".sql") {
		return true
	}

	lower := strings.ToLower(normalized)
	for _, segment := range databaseDirectorySegments {
		if strings.HasPrefix(lower, segment+"/") ||
			strings.Contains(lower, "/"+segment+"/") ||
			strings.HasSuffix(lower, "/"+segment) {
			return true
		}
	}
	return false
}
