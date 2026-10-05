package application

import (
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

func TestFrameworkAppendixMatchesTheDetectedFramework(t *testing.T) {
	cases := []struct {
		framework domain.Framework
		markers   []string
		forbidden string
	}{
		{domain.FrameworkGoNetHTTP, []string{"http.HandleFunc", "mux.Handle"}, "NESTED GROUPS"},
		{domain.FrameworkGin, []string{"NESTED GROUPS", "r.Group"}, "app.Get"},
		{domain.FrameworkFiber, []string{"app.Get", "Group", "app.Use"}, "NESTED GROUPS"},
		{domain.FrameworkEcho, []string{"e.GET", "e.Group", "bind"}, "NESTED GROUPS"},
		{domain.FrameworkPHPPlain, []string{"$_GET", "$_POST", "front controller"}, "Route::"},
		{domain.FrameworkLaravel, []string{"routes/api.php", "routes/web.php", "Route::apiResource"}, "$router->get"},
		{domain.FrameworkLumen, []string{"routes/web.php", "$router->get", "$router->post"}, "routes/api.php"},
		{domain.FrameworkNode, []string{"app.get", "app.post", "router mounts"}, "NESTED GROUPS"},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.framework), func(t *testing.T) {
			appendix := FrameworkAppendix(testCase.framework, false)
			for _, marker := range testCase.markers {
				if !strings.Contains(appendix, marker) {
					t.Fatalf("expected the %s appendix to mention %q, received %q", testCase.framework, marker, appendix)
				}
			}
			if strings.Contains(appendix, testCase.forbidden) {
				t.Fatalf("the %s appendix must not carry %q, received %q", testCase.framework, testCase.forbidden, appendix)
			}
		})
	}
}

func TestFrameworkAppendixAddsWireGuidanceWithoutASecondAppendix(t *testing.T) {
	prompt := SystemPrompt(domain.Classification{Framework: domain.FrameworkGin, UsesWire: true})

	if count := strings.Count(prompt, "Framework appendix"); count != 1 {
		t.Fatalf("expected exactly one framework appendix, received %d", count)
	}
	if !strings.Contains(prompt, "wire_gen.go") {
		t.Fatalf("expected the Wire guidance, received %q", prompt)
	}
	if !strings.Contains(prompt, "NESTED GROUPS") {
		t.Fatalf("expected the Gin guidance to survive, received %q", prompt)
	}
}

func TestSystemPromptCarriesTheOutputContractAndTheVendorRule(t *testing.T) {
	prompt := SystemPrompt(domain.Classification{Framework: domain.FrameworkGin})

	for _, marker := range []string{
		"Postman Collection v2.1",
		"NEVER omit a key you identified",
		"handler not found",
		"{{baseUrl}}",
		"src/vendor are first-party service code",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("expected the system prompt to state %q", marker)
		}
	}
	if strings.Contains(prompt, "effort") {
		t.Fatalf("the system prompt must not mention an effort setting")
	}
}

func TestUserPromptCarriesTheManifestAndTheContents(t *testing.T) {
	selection := domain.Selection{
		Service: "atlas",
		Files: []domain.SelectedFile{
			{RelativePath: "src/manage_user/routes.go", Content: "route.GET(\"/level\", handler)", SizeBytes: 30},
		},
		TotalBytes: 30,
	}

	prompt := UserPrompt("atlas", selection)

	for _, marker := range []string{
		"Service: atlas",
		"Selected files: 1",
		"Total bytes: 30",
		"- src/manage_user/routes.go (30 bytes)",
		"===== src/manage_user/routes.go =====",
		"route.GET(\"/level\", handler)",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("expected the user prompt to contain %q, received %q", marker, prompt)
		}
	}
}
