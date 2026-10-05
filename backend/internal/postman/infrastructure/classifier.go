package infrastructure

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"lunar/backend/internal/postman/domain"
)

const codeIgniterReason = "the repository is a CodeIgniter 3 application: it has both an application/ directory and a system/ directory"

const missingRouteReason = "no route or handler definition was found in the selected files"

var frontendPackages = []string{"vue", "react", "next", "nuxt", "angular", "svelte"}

var routeDefinitionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[a-z_][a-z0-9_]*\s*\.\s*(get|post|put|patch|delete|options|head|all|any|group|handle|handlefunc|use|register)\s*\(`),
	regexp.MustCompile(`(?i)(Route::|\$[a-z_][a-z0-9_]*\s*->\s*(get|post|put|patch|delete)|@(get|post|put|patch|delete|request)mapping\b)`),
	regexp.MustCompile(`\$_(GET|POST|REQUEST)\b`),
}

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) Classify(selection domain.Selection) domain.Classification {
	if selection.HasApplicationDirectory && selection.HasSystemDirectory {
		return domain.Classification{Reason: codeIgniterReason}
	}

	signals := detectSignals(selection)
	framework := resolveFramework(signals)
	if frontend := declaredFrontendFramework(selection); frontend != "" && framework == domain.FrameworkUnknown {
		return domain.Classification{Reason: fmt.Sprintf("the repository is a %s front-end application and declares no server framework", frontend)}
	}
	if !hasRouteDefinition(selection) {
		return domain.Classification{Reason: missingRouteReason}
	}

	return domain.Classification{
		IsExportable: true,
		Framework:    framework,
		UsesWire:     signals.wire,
	}
}

type frameworkSignals struct {
	gin       bool
	fiber     bool
	echo      bool
	goNetHTTP bool
	laravel   bool
	lumen     bool
	phpPlain  bool
	node      bool
	wire      bool
}

func detectSignals(selection domain.Selection) frameworkSignals {
	signals := frameworkSignals{}
	for _, file := range selection.Files {
		content := file.Content
		signals.wire = signals.wire || isWireFileName(file.RelativePath)
		signals.gin = signals.gin || strings.Contains(content, "gin-gonic/gin") || strings.Contains(content, "*gin.Engine")
		signals.fiber = signals.fiber || strings.Contains(content, "gofiber/fiber") || strings.Contains(content, "fiber.Ctx")
		signals.echo = signals.echo || strings.Contains(content, "labstack/echo") || strings.Contains(content, "echo.Context")
		signals.goNetHTTP = signals.goNetHTTP || (strings.Contains(content, "net/http") && strings.Contains(content, "Handle"))
		signals.laravel = signals.laravel || strings.Contains(content, "Route::") || strings.Contains(content, "Illuminate\\") || strings.Contains(file.RelativePath, "routes/api.php")
		signals.lumen = signals.lumen || strings.Contains(content, "$router->")
		signals.phpPlain = signals.phpPlain || strings.Contains(content, "$_GET") || strings.Contains(content, "$_POST")
		signals.node = signals.node || hasNodeServerSignature(content)
	}
	return signals
}

func resolveFramework(signals frameworkSignals) domain.Framework {
	switch {
	case signals.laravel:
		return domain.FrameworkLaravel
	case signals.lumen:
		return domain.FrameworkLumen
	case signals.phpPlain:
		return domain.FrameworkPHPPlain
	case signals.gin:
		return domain.FrameworkGin
	case signals.fiber:
		return domain.FrameworkFiber
	case signals.echo:
		return domain.FrameworkEcho
	case signals.node:
		return domain.FrameworkNode
	case signals.goNetHTTP:
		return domain.FrameworkGoNetHTTP
	default:
		return domain.FrameworkUnknown
	}
}

func hasNodeServerSignature(content string) bool {
	markers := []string{"require('express')", "require(\"express\")", "from 'express'", "from \"express\"", "require('koa')", "require(\"koa\")", "require('fastify')", "require(\"fastify\")", "fastify("}
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return strings.Contains(content, "router.get(") || strings.Contains(content, "router.post(")
}

func isWireFileName(relativePath string) bool {
	baseName := path.Base(relativePath)
	return baseName == "wire.go" || baseName == "wire_gen.go"
}

func hasRouteDefinition(selection domain.Selection) bool {
	for _, file := range selection.Files {
		for _, pattern := range routeDefinitionPatterns {
			if pattern.MatchString(file.Content) {
				return true
			}
		}
	}
	return false
}

type packageManifest struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

func declaredFrontendFramework(selection domain.Selection) string {
	if !selection.HasPackageJSON {
		return ""
	}

	packages := declaredPackages(selection.PackageJSONContent)
	for name := range packages {
		normalized := strings.TrimPrefix(name, "@")
		for _, candidate := range frontendPackages {
			if matchesPackage(normalized, candidate) {
				return candidate
			}
		}
	}
	return ""
}

func declaredPackages(content string) map[string]bool {
	packages := map[string]bool{}
	var manifest packageManifest
	if err := json.Unmarshal([]byte(content), &manifest); err != nil {
		return packages
	}
	groups := []map[string]string{manifest.Dependencies, manifest.DevDependencies, manifest.PeerDependencies, manifest.OptionalDependencies}
	for _, group := range groups {
		for name := range group {
			packages[strings.ToLower(name)] = true
		}
	}
	return packages
}

func matchesPackage(name string, candidate string) bool {
	return name == candidate ||
		strings.HasPrefix(name, candidate+"/") ||
		strings.HasPrefix(name, candidate+"-") ||
		strings.HasPrefix(name, candidate+".")
}
