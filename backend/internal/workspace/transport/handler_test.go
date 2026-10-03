package transport

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedDatabase "lunar/backend/internal/shared/database"
	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
	"lunar/backend/internal/workspace/infrastructure"
)

const (
	handlerTestJWTSecret   = "workspace-handler-jwt-secret"
	handlerTestUserA       = "handler-user-a"
	handlerTestUserB       = "handler-user-b"
	handlerTestEncryptKey  = "0123456789abcdef0123456789abcdef"
	handlerTestHelperToken = "handler-helper-token-value"
)

func setupWorkspaceHandler(t *testing.T, gitReader workspaceDomain.GitReader) (http.Handler, *sql.DB) {
	t.Helper()

	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "handler.db"))
	if err != nil {
		t.Fatalf("cannot open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate test database: %v", err)
	}

	now := time.Now().UTC()
	for _, userID := range []string{handlerTestUserA, handlerTestUserB} {
		if _, err := db.Exec(
			"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			userID,
			userID+"@example.com",
			"password-hash",
			"salt-value",
			userID,
			now,
			now,
		); err != nil {
			t.Fatalf("cannot seed user %q: %v", userID, err)
		}
	}

	workspaceRepo := infrastructure.NewSQLiteWorkspaceRepository(db)
	selectionRepo := infrastructure.NewSQLiteSelectionRepository(db)
	service := application.NewWorkspaceService(workspaceRepo, selectionRepo, gitReader, nil, handlerTestEncryptKey)
	handler := NewWorkspaceHandler(service)

	authMiddleware := sharedAuth.RequireAuth(handlerTestJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("GET /api/workspace", authMiddleware(http.HandlerFunc(handler.GetWorkspace)))
	mux.Handle("PUT /api/workspace", authMiddleware(http.HandlerFunc(handler.SaveWorkspace)))
	mux.Handle("GET /api/workspace/repositories", authMiddleware(http.HandlerFunc(handler.ListRepositories)))
	mux.Handle("GET /api/workspace/services", authMiddleware(http.HandlerFunc(handler.ListServices)))
	mux.Handle("PUT /api/workspace/selections", authMiddleware(http.HandlerFunc(handler.ReplaceSelections)))

	return mux, db
}

func authorizedRequest(t *testing.T, method string, path string, body string, userID string) *http.Request {
	t.Helper()

	var requestBody io.Reader
	if body != "" {
		requestBody = bytes.NewBufferString(body)
	}

	request := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	token, err := sharedAuth.GenerateToken(userID, userID+"@example.com", handlerTestJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("cannot generate token: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)

	return request
}

func performRequest(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeWorkspaceResponse(t *testing.T, recorder *httptest.ResponseRecorder) workspaceResponse {
	t.Helper()

	var response workspaceResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode workspace response: %v", err)
	}
	return response
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) sharedHttp.ErrorResponse {
	t.Helper()

	var errorResponse sharedHttp.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("cannot decode error response: %v", err)
	}
	return errorResponse
}

func decodeSelectionsResponse(t *testing.T, recorder *httptest.ResponseRecorder) selectionsResponse {
	t.Helper()

	var response selectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode selections response: %v", err)
	}
	return response
}

func storedCiphertext(t *testing.T, db *sql.DB, userID string) string {
	t.Helper()

	var helperTokenEnc string
	if err := db.QueryRow("SELECT helper_token_enc FROM workspaces WHERE user_id = ?", userID).Scan(&helperTokenEnc); err != nil {
		t.Fatalf("cannot read stored ciphertext: %v", err)
	}
	return helperTokenEnc
}

func storedSelectionNames(t *testing.T, db *sql.DB, userID string) []string {
	t.Helper()

	rows, err := db.Query("SELECT repo_name FROM repo_selections WHERE user_id = ? ORDER BY repo_name", userID)
	if err != nil {
		t.Fatalf("cannot read selections: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	repoNames := make([]string, 0)
	for rows.Next() {
		var repoName string
		if err := rows.Scan(&repoName); err != nil {
			t.Fatalf("cannot scan selection: %v", err)
		}
		repoNames = append(repoNames, repoName)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot iterate selections: %v", err)
	}
	return repoNames
}

func TestWorkspaceHandler_UnauthenticatedRequestsReturn401(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)

	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/workspace", ""},
		{http.MethodPut, "/api/workspace", `{"source":"helper"}`},
		{http.MethodGet, "/api/workspace/repositories", ""},
		{http.MethodGet, "/api/workspace/services", ""},
		{http.MethodPut, "/api/workspace/selections", `{"repos":[]}`},
	}

	for _, route := range routes {
		var requestBody io.Reader
		if route.body != "" {
			requestBody = bytes.NewBufferString(route.body)
		}
		request := httptest.NewRequest(route.method, route.path, requestBody)

		recorder := performRequest(handler, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("expected %s %s to return 401, got %d", route.method, route.path, recorder.Code)
		}
	}
}

func TestWorkspaceHandler_GetWorkspaceDefaultsForFreshUser(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)

	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace", "", handlerTestUserA))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	responseBody := recorder.Body.String()
	response := decodeWorkspaceResponse(t, recorder)
	if response.Source != string(workspaceDomain.SourceHelper) {
		t.Errorf("expected default source helper, got %q", response.Source)
	}
	if response.RootPath != "" || response.HelperURL != "" {
		t.Errorf("expected empty workspace fields, got %+v", response)
	}
	if response.HasHelperToken {
		t.Error("expected has_helper_token to be false")
	}
	if response.Selected == nil {
		t.Fatal("expected selected to be an empty array, not null")
	}
	if len(response.Selected) != 0 {
		t.Errorf("expected no selections, got %v", response.Selected)
	}
	if !strings.Contains(responseBody, `"selected":[]`) {
		t.Errorf("expected selected to serialize as an empty array, got %s", responseBody)
	}
}

func TestWorkspaceHandler_WorkspaceResponsesIncludeSelectedRepositories(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)

	putSelections := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest","aurora"]}`, handlerTestUserA))
	if putSelections.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", putSelections.Code, putSelections.Body.String())
	}

	getRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace", "", handlerTestUserA))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}
	selectedOnGet := decodeWorkspaceResponse(t, getRecorder).Selected
	if len(selectedOnGet) != 2 || selectedOnGet[0] != "aurora" || selectedOnGet[1] != "everest" {
		t.Fatalf("expected the stored selections on GET, got %v", selectedOnGet)
	}

	saveBody := `{"source":"helper","root_path":"/Users/erendt/BRI"}`
	saveRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", saveBody, handlerTestUserA))
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", saveRecorder.Code, saveRecorder.Body.String())
	}
	selectedOnSave := decodeWorkspaceResponse(t, saveRecorder).Selected
	if len(selectedOnSave) != 2 || selectedOnSave[0] != "aurora" || selectedOnSave[1] != "everest" {
		t.Fatalf("expected the stored selections on PUT, got %v", selectedOnSave)
	}

	otherUserRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace", "", handlerTestUserB))
	if otherUserRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for user B, got %d", otherUserRecorder.Code)
	}
	if len(decodeWorkspaceResponse(t, otherUserRecorder).Selected) != 0 {
		t.Fatal("expected user B to see no selections from user A")
	}
}

func TestWorkspaceHandler_SaveWorkspaceStoresEncryptedTokenAndNeverReturnsIt(t *testing.T) {
	handler, db := setupWorkspaceHandler(t, nil)

	saveBody := `{"source":"helper","root_path":"/Users/erendt/BRI","helper_url":"http://127.0.0.1:5199","helper_token":"` + handlerTestHelperToken + `"}`
	saveRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", saveBody, handlerTestUserA))

	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", saveRecorder.Code, saveRecorder.Body.String())
	}
	if strings.Contains(saveRecorder.Body.String(), handlerTestHelperToken) {
		t.Fatal("expected the PUT response to never contain the helper token")
	}
	savedResponse := decodeWorkspaceResponse(t, saveRecorder)
	if !savedResponse.HasHelperToken {
		t.Error("expected has_helper_token true after saving a token")
	}
	if savedResponse.HelperURL != "http://127.0.0.1:5199" {
		t.Errorf("unexpected helper url: %q", savedResponse.HelperURL)
	}

	ciphertext := storedCiphertext(t, db, handlerTestUserA)
	if ciphertext == "" {
		t.Fatal("expected the helper token to be stored")
	}
	if ciphertext == handlerTestHelperToken {
		t.Fatal("expected the helper token to be encrypted at rest")
	}

	getRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace", "", handlerTestUserA))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}
	if strings.Contains(getRecorder.Body.String(), handlerTestHelperToken) {
		t.Fatal("expected the GET response to never contain the helper token")
	}
	if strings.Contains(getRecorder.Body.String(), ciphertext) {
		t.Fatal("expected the GET response to never contain the stored ciphertext")
	}
	loadedResponse := decodeWorkspaceResponse(t, getRecorder)
	if !loadedResponse.HasHelperToken {
		t.Error("expected has_helper_token true on GET")
	}
	if loadedResponse.RootPath != "/Users/erendt/BRI" {
		t.Errorf("unexpected root path: %q", loadedResponse.RootPath)
	}

	otherUserRecorder := performRequest(handler, authorizedRequest(t, http.MethodGet, "/api/workspace", "", handlerTestUserB))
	if otherUserRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for user B, got %d", otherUserRecorder.Code)
	}
	if decodeWorkspaceResponse(t, otherUserRecorder).HasHelperToken {
		t.Fatal("expected user B to see no helper token from user A")
	}
}

func TestWorkspaceHandler_SaveWorkspacePreservesTokenWhenOmittedAndClearsWhenEmpty(t *testing.T) {
	handler, db := setupWorkspaceHandler(t, nil)

	initialBody := `{"source":"helper","helper_token":"token-to-preserve"}`
	initialRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", initialBody, handlerTestUserA))
	if initialRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", initialRecorder.Code, initialRecorder.Body.String())
	}
	initialCiphertext := storedCiphertext(t, db, handlerTestUserA)

	omittedBody := `{"source":"helper","root_path":"/Users/erendt/BRI"}`
	omittedRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", omittedBody, handlerTestUserA))
	if omittedRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", omittedRecorder.Code, omittedRecorder.Body.String())
	}
	if !decodeWorkspaceResponse(t, omittedRecorder).HasHelperToken {
		t.Error("expected an omitted helper_token to preserve the stored token")
	}
	if storedCiphertext(t, db, handlerTestUserA) != initialCiphertext {
		t.Fatal("expected the stored ciphertext to stay untouched when helper_token is omitted")
	}

	clearBody := `{"source":"helper","root_path":"/Users/erendt/BRI","helper_token":""}`
	clearRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", clearBody, handlerTestUserA))
	if clearRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", clearRecorder.Code, clearRecorder.Body.String())
	}
	if decodeWorkspaceResponse(t, clearRecorder).HasHelperToken {
		t.Error("expected an explicit empty helper_token to clear the stored token")
	}
	if storedCiphertext(t, db, handlerTestUserA) != "" {
		t.Fatal("expected the stored ciphertext to be cleared")
	}
}

func TestWorkspaceHandler_SaveWorkspaceRejectsInvalidInput(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)
	missingRoot := filepath.Join(t.TempDir(), "missing-root-folder")

	cases := []struct {
		name        string
		body        string
		expectedMsg string
	}{
		{"unknown source", `{"source":"cluster"}`, "source must be"},
		{"invalid helper url", `{"source":"helper","helper_url":"ftp://127.0.0.1:5199"}`, "helper url must use http or https"},
		{"missing root for server", `{"source":"server","root_path":"` + missingRoot + `"}`, "workspace root does not exist"},
	}

	for _, testCase := range cases {
		recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", testCase.body, handlerTestUserA))

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: expected status 400, got %d: %s", testCase.name, recorder.Code, recorder.Body.String())
			continue
		}
		errorResponse := decodeErrorResponse(t, recorder)
		if !strings.Contains(errorResponse.Error, testCase.expectedMsg) {
			t.Errorf("%s: expected error to contain %q, got %q", testCase.name, testCase.expectedMsg, errorResponse.Error)
		}
	}
}

func TestWorkspaceHandler_SaveWorkspaceDoesNotLeakAbsolutePath(t *testing.T) {
	handler, _ := setupWorkspaceHandler(t, nil)
	missingRoot := filepath.Join(t.TempDir(), "missing-root-folder")

	recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace", `{"source":"server","root_path":"`+missingRoot+`"}`, handlerTestUserA))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), missingRoot) {
		t.Fatalf("expected the absolute path to stay out of the response, got %s", recorder.Body.String())
	}
}

func TestWorkspaceHandler_ReplaceSelectionsStoresWholeListAndReturnsCount(t *testing.T) {
	handler, db := setupWorkspaceHandler(t, nil)

	firstRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest","aurora"]}`, handlerTestUserA))
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", firstRecorder.Code, firstRecorder.Body.String())
	}
	if count := decodeSelectionsResponse(t, firstRecorder).SelectedCount; count != 2 {
		t.Fatalf("expected selected_count 2, got %d", count)
	}

	secondRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest"]}`, handlerTestUserA))
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", secondRecorder.Code, secondRecorder.Body.String())
	}
	if count := decodeSelectionsResponse(t, secondRecorder).SelectedCount; count != 1 {
		t.Fatalf("expected selected_count 1, got %d", count)
	}

	selectionsForA := storedSelectionNames(t, db, handlerTestUserA)
	if len(selectionsForA) != 1 || selectionsForA[0] != "everest" {
		t.Fatalf("expected the removed repository to be gone, got %v", selectionsForA)
	}

	otherUserRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["way4"]}`, handlerTestUserB))
	if otherUserRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for user B, got %d", otherUserRecorder.Code)
	}
	if count := decodeSelectionsResponse(t, otherUserRecorder).SelectedCount; count != 1 {
		t.Fatalf("expected selected_count 1 for user B, got %d", count)
	}

	selectionsForA = storedSelectionNames(t, db, handlerTestUserA)
	if len(selectionsForA) != 1 || selectionsForA[0] != "everest" {
		t.Fatalf("expected user B to not touch user A selections, got %v", selectionsForA)
	}
	selectionsForB := storedSelectionNames(t, db, handlerTestUserB)
	if len(selectionsForB) != 1 || selectionsForB[0] != "way4" {
		t.Fatalf("expected user B selections, got %v", selectionsForB)
	}
}

func TestWorkspaceHandler_ReplaceSelectionsRejectsInvalidNameForWholeBatch(t *testing.T) {
	handler, db := setupWorkspaceHandler(t, nil)

	seedRecorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["everest"]}`, handlerTestUserA))
	if seedRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", seedRecorder.Code)
	}

	recorder := performRequest(handler, authorizedRequest(t, http.MethodPut, "/api/workspace/selections", `{"repos":["aurora","../etc"]}`, handlerTestUserA))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	errorResponse := decodeErrorResponse(t, recorder)
	if !strings.Contains(errorResponse.Error, "invalid repository name") {
		t.Errorf("expected an invalid repository name message, got %q", errorResponse.Error)
	}

	selectionsForA := storedSelectionNames(t, db, handlerTestUserA)
	if len(selectionsForA) != 1 || selectionsForA[0] != "everest" {
		t.Fatalf("expected the rejected batch to leave selections untouched, got %v", selectionsForA)
	}
	if strings.Contains(strings.Join(selectionsForA, ","), "aurora") {
		t.Fatal("expected no partially applied batch")
	}
}
