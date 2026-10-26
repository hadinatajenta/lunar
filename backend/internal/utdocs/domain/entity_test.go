package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDocumentRequestAcceptsTheHelperBlockedFileShape(t *testing.T) {
	body := `{"model":"DeepSeek-V4 Flash","repositories":[{"repository":"mms-api","branch":"feature/x","base_ref":"development","test_files":[{"path":"a_test.go","package":"./a","language":"go","classification":"safe","reason":"","test_functions":["TestA"]}],"blocked_files":[{"path":"routes/x.test.js","package":"","language":"","classification":"blocked","reason":"the node test runner is not available yet"}],"run":{"status":"blocked","executed_packages":[],"total":0,"passed":0,"failed":0,"duration_ms":0,"output_excerpt":""},"tests":[]}]}`

	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()

	var request DocumentRequest
	if err := decoder.Decode(&request); err != nil {
		t.Fatalf("expected the helper blocked file shape to decode, got %v", err)
	}
	if len(request.Repositories) != 1 || len(request.Repositories[0].BlockedFiles) != 1 {
		t.Fatalf("expected one repository with one blocked file, got %+v", request.Repositories)
	}
	if request.Repositories[0].BlockedFiles[0].Path != "routes/x.test.js" {
		t.Fatalf("expected the blocked file path, got %q", request.Repositories[0].BlockedFiles[0].Path)
	}
}
