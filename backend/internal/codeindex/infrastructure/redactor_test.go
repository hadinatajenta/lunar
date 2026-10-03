package infrastructure

import (
	"strings"
	"testing"
)

func TestIsExcludedFileName(t *testing.T) {
	excluded := []string{
		".env",
		".env.local",
		".env.production",
		"server.pem",
		"app.key",
		"client.p12",
		"app.keystore",
		"signing.jks",
		"id_rsa",
		"id_rsa.pub",
		"id_ed25519",
		"credentials.go",
		"client_secret.go",
		"secret.go",
		".npmrc",
		".netrc",
		".htpasswd",
		"API_KEY.PEM",
		"config/.env.staging",
	}
	for _, name := range excluded {
		if !IsExcludedFileName(name) {
			t.Errorf("expected %q to be excluded", name)
		}
	}

	allowed := []string{
		"main.go",
		"env.go",
		"environment.go",
		"keyboard.go",
		"password_reset.go",
		"settings.go",
	}
	for _, name := range allowed {
		if IsExcludedFileName(name) {
			t.Errorf("expected %q to be indexed", name)
		}
	}
}

func TestRedactSecretsReplacesSecretValues(t *testing.T) {
	testCases := []struct {
		input         string
		expected      string
		expectedCount int
	}{
		{`apiKey = "sk-abcdefghijkl"`, `apiKey = "***"`, 1},
		{`password = "hunter2hunter2"`, `password = "***"`, 1},
		{`token: 'abcdefghijkl'`, `token: '***'`, 1},
		{`private_key = "abcdefghijklmnop"`, `private_key = "***"`, 1},
		{`secret = "abcdefghijkl" and passwd = "zyxwvutsrqpo"`, `secret = "***" and passwd = "***"`, 2},
		{`token = "short"`, `token = "short"`, 0},
		{`tokenValue = "abcdefghijkl"`, `tokenValue = "abcdefghijkl"`, 0},
	}

	for _, testCase := range testCases {
		redacted, redactionCount := RedactSecrets(testCase.input)
		if redacted != testCase.expected {
			t.Errorf("RedactSecrets(%q) = %q, want %q", testCase.input, redacted, testCase.expected)
		}
		if redactionCount != testCase.expectedCount {
			t.Errorf("RedactSecrets(%q) replaced %d values, want %d", testCase.input, redactionCount, testCase.expectedCount)
		}
	}
}

func TestRedactSecretsPerLineKeepsLineStructure(t *testing.T) {
	source := "package config\n\nvar password = \"hunter2hunter2\"\n\nvar apiKey = \"sk-abcdefghijkl\"\n"

	redacted, redactionCount := RedactSecretsPerLine(source)
	if redactionCount != 2 {
		t.Fatalf("expected 2 redactions, got %d", redactionCount)
	}
	if strings.Count(redacted, "\n") != strings.Count(source, "\n") {
		t.Errorf("expected line count to be preserved, got %q", redacted)
	}
	if strings.Contains(redacted, "hunter2hunter2") || strings.Contains(redacted, "sk-abcdefghijkl") {
		t.Errorf("expected secret values to be removed, got %q", redacted)
	}
	if !strings.Contains(redacted, `var password = "***"`) {
		t.Errorf("expected key name to be preserved, got %q", redacted)
	}
}

func TestRedactSecretsPerLineWithoutNewlines(t *testing.T) {
	redacted, redactionCount := RedactSecretsPerLine(`password = "hunter2hunter2"`)
	if redactionCount != 1 {
		t.Fatalf("expected 1 redaction, got %d", redactionCount)
	}
	if redacted != `password = "***"` {
		t.Errorf("unexpected redacted value %q", redacted)
	}
}
