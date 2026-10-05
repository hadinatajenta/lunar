package infrastructure

import (
	"sort"
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

func selectionFromFiles(files map[string]string) domain.Selection {
	selection := domain.Selection{Service: "sample"}
	paths := make([]string, 0, len(files))
	for relativePath := range files {
		paths = append(paths, relativePath)
	}
	sort.Strings(paths)
	for _, relativePath := range paths {
		selection.Files = append(selection.Files, domain.SelectedFile{RelativePath: relativePath, Content: files[relativePath]})
	}
	return selection
}

func TestClassifierRefusesACodeIgniterMonolith(t *testing.T) {
	selection := selectionFromFiles(map[string]string{"application/config/routes.php": "<?php\n"})
	selection.HasApplicationDirectory = true
	selection.HasSystemDirectory = true

	classification := NewClassifier().Classify(selection)
	if classification.IsExportable {
		t.Fatalf("a CodeIgniter 3 tree must be refused")
	}
	if !strings.Contains(classification.Reason, "CodeIgniter") {
		t.Fatalf("expected a CodeIgniter reason, received %q", classification.Reason)
	}
}

func TestClassifierRefusesAVueFrontendWithoutAServerFramework(t *testing.T) {
	selection := selectionFromFiles(map[string]string{"src/main.js": "import { createApp } from 'vue'\n"})
	selection.HasPackageJSON = true
	selection.PackageJSONContent = "{\"dependencies\":{\"vue\":\"^3.4.0\",\"vue-router\":\"^4.0.0\"}}"

	classification := NewClassifier().Classify(selection)
	if classification.IsExportable {
		t.Fatalf("a Vue front-end without a server framework must be refused")
	}
	if !strings.Contains(classification.Reason, "vue") {
		t.Fatalf("expected a vue reason, received %q", classification.Reason)
	}
}

func TestClassifierKeepsALaravelApplicationThatDeclaresVue(t *testing.T) {
	files := map[string]string{
		"routes/api.php": "<?php Route::get('/merchants', 'MerchantController@index');\n",
		"package.json":   "{\"dependencies\":{\"vue\":\"^3.4.0\"}}",
	}
	selection := selectionFromFiles(files)
	selection.HasPackageJSON = true
	selection.PackageJSONContent = files["package.json"]

	classification := NewClassifier().Classify(selection)
	if !classification.IsExportable {
		t.Fatalf("a Laravel application must stay exportable, received reason %q", classification.Reason)
	}
	if classification.Framework != domain.FrameworkLaravel {
		t.Fatalf("expected the Laravel appendix, received %q", classification.Framework)
	}
}

func TestClassifierRefusesARepositoryWithoutRouteDefinitions(t *testing.T) {
	selection := selectionFromFiles(map[string]string{"src/math/const.go": "package math\n\nconst PiValue = 3.14\n"})

	classification := NewClassifier().Classify(selection)
	if classification.IsExportable {
		t.Fatalf("a repository without a route definition must be refused")
	}
	if !strings.Contains(classification.Reason, "no route or handler definition") {
		t.Fatalf("expected a missing-route reason, received %q", classification.Reason)
	}
}

func TestClassifierDetectsTheFrameworkAppendix(t *testing.T) {
	cases := []struct {
		name     string
		files    map[string]string
		expected domain.Framework
		usesWire bool
	}{
		{
			name: "gin with wire",
			files: map[string]string{
				"src/manage_user/routes.go": "package manage_user\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc Register(r *gin.Engine) { route := r.Group(\"/v1\"); route.GET(\"/x\", nil) }\n",
				"src/wire_gen.go":           "package main\n",
			},
			expected: domain.FrameworkGin,
			usesWire: true,
		},
		{
			name:     "go net http",
			files:    map[string]string{"cmd/main.go": "package main\n\nimport \"net/http\"\n\nfunc main() { http.HandleFunc(\"/x\", nil) }\n"},
			expected: domain.FrameworkGoNetHTTP,
		},
		{
			name:     "fiber",
			files:    map[string]string{"src/app.go": "package app\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc main() { app := fiber.New(); app.Get(\"/x\", nil) }\n"},
			expected: domain.FrameworkFiber,
		},
		{
			name:     "echo",
			files:    map[string]string{"src/app.go": "package app\n\nimport \"github.com/labstack/echo/v4\"\n\nfunc main() { e := echo.New(); e.GET(\"/x\", nil) }\n"},
			expected: domain.FrameworkEcho,
		},
		{
			name:     "lumen",
			files:    map[string]string{"routes/web.php": "<?php $router->get('/x', 'XController@index');\n"},
			expected: domain.FrameworkLumen,
		},
		{
			name:     "php plain",
			files:    map[string]string{"app/index.php": "<?php $id = $_GET['id'];\n"},
			expected: domain.FrameworkPHPPlain,
		},
		{
			name:     "node express",
			files:    map[string]string{"app.js": "const express = require('express')\napp.get('/x', handler)\n"},
			expected: domain.FrameworkNode,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			classification := NewClassifier().Classify(selectionFromFiles(testCase.files))
			if !classification.IsExportable {
				t.Fatalf("expected an exportable classification, received %q", classification.Reason)
			}
			if classification.Framework != testCase.expected {
				t.Fatalf("expected framework %q, received %q", testCase.expected, classification.Framework)
			}
			if classification.UsesWire != testCase.usesWire {
				t.Fatalf("expected UsesWire %v, received %v", testCase.usesWire, classification.UsesWire)
			}
		})
	}
}
