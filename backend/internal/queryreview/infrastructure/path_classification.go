package infrastructure

import (
	"path"
	"strings"
)

const (
	skipReasonTestFile        = "test file"
	skipReasonMockFile        = "mock file"
	skipReasonGeneratedFile   = "generated file"
	skipReasonVendorFile      = "vendor file"
	skipReasonDocumentation   = "documentation file"
	skipReasonAssetFile       = "asset file"
	skipReasonNoAddedLines    = "no added lines"
	skipReasonNotDatabase     = "no database relevant path or query lines"
	skipReasonOverBudget      = "exceeds the per file analysis budget"
	skipReasonOverTotalBudget = "exceeds the total analysis budget"
)

var databasePathTokens = map[string]bool{
	"dao":          true,
	"daos":         true,
	"repository":   true,
	"repositories": true,
	"repo":         true,
	"repos":        true,
	"migration":    true,
	"migrations":   true,
	"schema":       true,
	"schemas":      true,
	"query":        true,
	"queries":      true,
	"sql":          true,
	"database":     true,
	"db":           true,
	"store":        true,
	"storage":      true,
	"persistence":  true,
	"persistent":   true,
	"entity":       true,
	"entities":     true,
	"model":        true,
	"models":       true,
	"mapper":       true,
	"mappers":      true,
	"gorm":         true,
	"orm":          true,
}

func isDatabaseRelevantPath(filePath string) bool {
	lowered := strings.ToLower(filePath)
	if strings.HasSuffix(lowered, ".sql") {
		return true
	}

	for _, segment := range strings.Split(lowered, "/") {
		for _, token := range strings.FieldsFunc(segment, isPathTokenSeparator) {
			if databasePathTokens[token] {
				return true
			}
		}
	}
	return false
}

func isPathTokenSeparator(r rune) bool {
	return r == '_' || r == '-' || r == '.'
}

var querySyntaxMarkers = []string{
	"select ",
	"insert into",
	"insert ignore",
	"replace into",
	"update ",
	"delete from",
	"where ",
	" join ",
	"left join",
	"inner join",
	"outer join",
	"group by",
	"order by",
	"having ",
	"gorm",
	".raw(",
	".exec(",
	".query(",
	".queryrow(",
	".execcontext(",
	".querycontext(",
	"db.table(",
	"db::table",
	"->table(",
	"->where(",
	"->select(",
	"->insert(",
	"->update(",
	"->delete(",
	"querybuilder",
	"createquerybuilder(",
	"knex(",
	"squirrel",
	"@query(",
	"@select(",
}

var testPathMarkers = []string{
	"_test.go",
	".test.ts",
	".test.tsx",
	".test.js",
	".spec.ts",
	".spec.tsx",
	".spec.js",
	"_test.py",
	".test.php",
}

var generatedPathMarkers = []string{
	"_generated",
	".gen.",
	".pb.go",
	".g.dart",
	".designer.cs",
	"__generated__",
	".min.js",
	".min.css",
	"package-lock.json",
	"yarn.lock",
	"go.sum",
	"poetry.lock",
}

var documentationExtensions = map[string]bool{
	".md":   true,
	".mdx":  true,
	".rst":  true,
	".adoc": true,
	".txt":  true,
}

var assetExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".svg":   true,
	".ico":   true,
	".webp":  true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".mp4":   true,
	".mp3":   true,
	".pdf":   true,
}

func classifyIgnoredPath(filePath string) string {
	lowered := strings.ToLower(strings.TrimSpace(filePath))
	if lowered == "" {
		return ""
	}

	segments := strings.Split(lowered, "/")
	fileName := segments[len(segments)-1]
	extension := strings.ToLower(path.Ext(lowered))

	for _, segment := range segments {
		switch segment {
		case "vendor", "node_modules", "third_party", "bower_components", ".venv":
			return skipReasonVendorFile
		}
	}

	for _, segment := range segments {
		if segment == "docs" || segment == "doc" || segment == "documentation" {
			return skipReasonDocumentation
		}
	}
	if strings.HasPrefix(fileName, "readme") || documentationExtensions[extension] {
		return skipReasonDocumentation
	}
	if assetExtensions[extension] {
		return skipReasonAssetFile
	}
	for _, segment := range segments {
		if segment == "assets" || segment == "static" || segment == "public" {
			return skipReasonAssetFile
		}
	}

	for _, marker := range testPathMarkers {
		if strings.Contains(lowered, marker) {
			return skipReasonTestFile
		}
	}
	for _, segment := range segments {
		switch segment {
		case "test", "tests", "__tests__", "spec", "specs":
			return skipReasonTestFile
		}
	}
	if strings.HasPrefix(fileName, "test_") {
		return skipReasonTestFile
	}

	for _, segment := range segments {
		switch segment {
		case "mock", "mocks", "__mocks__":
			return skipReasonMockFile
		}
	}
	if strings.Contains(lowered, "_mock.") || strings.Contains(lowered, "_mock_") || strings.HasPrefix(fileName, "mock_") {
		return skipReasonMockFile
	}

	for _, marker := range generatedPathMarkers {
		if strings.Contains(lowered, marker) {
			return skipReasonGeneratedFile
		}
	}
	for _, segment := range segments {
		switch segment {
		case "dist", "build", "generated", "gen", "__generated__":
			return skipReasonGeneratedFile
		}
	}

	return ""
}

var languageByExtension = map[string]string{
	".go":   "go",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".php":  "php",
	".py":   "python",
	".sql":  "sql",
	".java": "java",
	".kt":   "kotlin",
	".rb":   "ruby",
	".cs":   "csharp",
	".rs":   "rust",
	".vue":  "vue",
	".yaml": "yaml",
	".yml":  "yaml",
}

func languageForPath(filePath string) string {
	language, ok := languageByExtension[strings.ToLower(path.Ext(filePath))]
	if !ok {
		return "unknown"
	}
	return language
}
