package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/domain"
)

type stubUploader struct {
	calls     int
	userID    string
	pageID    string
	filename  string
	content   []byte
	mediaType string
	returnID  string
	err       error
}

func (u *stubUploader) UploadAttachment(_ context.Context, userID string, pageID string, filename string, content []byte, mediaType string) (string, error) {
	u.calls++
	u.userID = userID
	u.pageID = pageID
	u.filename = filename
	u.content = append([]byte{}, content...)
	u.mediaType = mediaType
	if u.err != nil {
		return "", u.err
	}
	return u.returnID, nil
}

type stubPageStorage struct {
	storage       string
	getErr        error
	updated       string
	updateVersion int
	updateURL     string
	updateErr     error
}

func (s *stubPageStorage) GetPageStorage(_ context.Context, _ string, _ string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return s.storage, nil
}

func (s *stubPageStorage) UpdatePageStorage(_ context.Context, _ string, _ string, storage string) (int, string, error) {
	s.updated = storage
	if s.updateErr != nil {
		return 0, "", s.updateErr
	}
	return s.updateVersion, s.updateURL, nil
}

func samplePNG(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("cannot encode the sample PNG: %v", err)
	}
	return buffer.Bytes()
}

func validApplyRequest(t *testing.T) domain.ApplyRequest {
	t.Helper()

	return domain.ApplyRequest{
		Repositories: []domain.ApplyRepository{
			{
				Repository: "micro-payment",
				Screenshot: domain.ApplyScreenshot{Filename: "ut-micro-payment.png", ContentBase64: base64.StdEncoding.EncodeToString(samplePNG(t))},
				Rows: []domain.ApplyRow{
					{Function: "Find payment", Scenario: "Mencari data", ExpectedResult: "Data ditemukan", InputData: "ID", Result: domain.ResultPassed},
				},
			},
		},
	}
}

func TestApplyServiceUploadsAndReplacesTheTable(t *testing.T) {
	uploader := &stubUploader{returnID: "att-1"}
	pages := &stubPageStorage{
		storage:       normalTableMarker + "\n<table class=\"old\"><tbody><tr><td>old</td></tr></tbody></table>",
		updateVersion: 8,
		updateURL:     "https://confluence.example.com/display/DEV/UT",
	}
	service := NewApplyService(uploader, pages)

	result, err := service.Apply(context.Background(), "user-1", "123456", validApplyRequest(t))
	if err != nil {
		t.Fatalf("expected the apply to succeed, got %v", err)
	}
	if result.PageID != "123456" || result.Version != 8 || result.URL != "https://confluence.example.com/display/DEV/UT" {
		t.Fatalf("unexpected apply result %+v", result)
	}
	if uploader.calls != 1 || uploader.pageID != "123456" || uploader.filename != "ut-micro-payment.png" || uploader.mediaType != "image/png" {
		t.Fatalf("unexpected upload call %+v", uploader)
	}
	if len(uploader.content) == 0 {
		t.Fatal("expected the decoded PNG content")
	}
	if strings.Contains(pages.updated, "old") {
		t.Fatalf("expected the old table to be replaced, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, "Normal Test Scenario") || !strings.Contains(pages.updated, "Find payment") {
		t.Fatalf("expected the new scenario table, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, `<ac:image ac:width="400"><ri:attachment ri:filename="ut-micro-payment.png"/></ac:image>`) {
		t.Fatalf("expected the attachment markup, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, domain.ResultPassed) {
		t.Fatalf("expected the system supplied result, got %q", pages.updated)
	}
}

func TestApplyServiceGroupsMultipleRepositories(t *testing.T) {
	uploader := &stubUploader{returnID: "att-1"}
	pages := &stubPageStorage{updateVersion: 2}
	service := NewApplyService(uploader, pages)

	request := validApplyRequest(t)
	request.Repositories = append(request.Repositories, domain.ApplyRepository{
		Repository: "micro-transfer",
		Screenshot: domain.ApplyScreenshot{Filename: "ut-micro-transfer.png", ContentBase64: base64.StdEncoding.EncodeToString(samplePNG(t))},
		Rows:       []domain.ApplyRow{{Function: "Transfer", Scenario: "Transfer dana", ExpectedResult: "Transfer berhasil", InputData: "Nominal", Result: domain.ResultFailed}},
	})

	if _, err := service.Apply(context.Background(), "user-1", "123456", request); err != nil {
		t.Fatalf("expected the apply to succeed, got %v", err)
	}
	if uploader.calls != 2 {
		t.Fatalf("expected two uploads, got %d", uploader.calls)
	}
	if strings.Count(pages.updated, "Layanan / Microservice") != 2 {
		t.Fatalf("expected two group rows, got %q", pages.updated)
	}
}

func TestApplyServiceRejectsInvalidScreenshots(t *testing.T) {
	service := NewApplyService(&stubUploader{}, &stubPageStorage{})

	cases := []struct {
		name     string
		request  func(t *testing.T) domain.ApplyRequest
		wantCode error
	}{
		{
			name: "empty base64",
			request: func(t *testing.T) domain.ApplyRequest {
				request := validApplyRequest(t)
				request.Repositories[0].Screenshot.ContentBase64 = " "
				return request
			},
			wantCode: sharedErrors.ErrBadRequest,
		},
		{
			name: "not base64",
			request: func(t *testing.T) domain.ApplyRequest {
				request := validApplyRequest(t)
				request.Repositories[0].Screenshot.ContentBase64 = "!!!!"
				return request
			},
			wantCode: sharedErrors.ErrBadRequest,
		},
		{
			name: "not a png",
			request: func(t *testing.T) domain.ApplyRequest {
				request := validApplyRequest(t)
				request.Repositories[0].Screenshot.ContentBase64 = base64.StdEncoding.EncodeToString([]byte("not a png"))
				return request
			},
			wantCode: sharedErrors.ErrBadRequest,
		},
		{
			name: "unsafe filename",
			request: func(t *testing.T) domain.ApplyRequest {
				request := validApplyRequest(t)
				request.Repositories[0].Screenshot.Filename = "../evil.png"
				return request
			},
			wantCode: sharedErrors.ErrBadRequest,
		},
		{
			name: "missing repository",
			request: func(t *testing.T) domain.ApplyRequest {
				request := validApplyRequest(t)
				request.Repositories[0].Repository = " "
				return request
			},
			wantCode: sharedErrors.ErrBadRequest,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := service.Apply(context.Background(), "user-1", "123456", testCase.request(t)); !errors.Is(err, testCase.wantCode) {
				t.Fatalf("expected %v, got %v", testCase.wantCode, err)
			}
		})
	}
}

func TestApplyServiceRejectsOversizedScreenshots(t *testing.T) {
	service := NewApplyService(&stubUploader{}, &stubPageStorage{})
	request := validApplyRequest(t)
	request.Repositories[0].Screenshot.ContentBase64 = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("a"), maxApplyScreenshotBytes+1))

	if _, err := service.Apply(context.Background(), "user-1", "123456", request); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for an oversized screenshot, got %v", err)
	}
}

func TestApplyServiceValidation(t *testing.T) {
	service := NewApplyService(&stubUploader{}, &stubPageStorage{})

	if _, err := service.Apply(context.Background(), "", "123456", validApplyRequest(t)); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
	if _, err := service.Apply(context.Background(), "user-1", " ", validApplyRequest(t)); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for a missing page, got %v", err)
	}
	if _, err := service.Apply(context.Background(), "user-1", "123456", domain.ApplyRequest{}); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request without repositories, got %v", err)
	}
}

func TestApplyServicePropagatesUploadAndPageFailures(t *testing.T) {
	uploadFailure := NewApplyService(&stubUploader{err: errors.New("confluence is unreachable")}, &stubPageStorage{})
	if _, err := uploadFailure.Apply(context.Background(), "user-1", "123456", validApplyRequest(t)); err == nil {
		t.Fatal("expected the upload failure to propagate")
	}

	pageFailure := NewApplyService(&stubUploader{}, &stubPageStorage{getErr: sharedErrors.ErrNotFound})
	if _, err := pageFailure.Apply(context.Background(), "user-1", "123456", validApplyRequest(t)); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected the page failure to propagate, got %v", err)
	}
}

func TestApplyServiceAppendsMockTestCodeMacros(t *testing.T) {
	uploader := &stubUploader{returnID: "att-1"}
	pages := &stubPageStorage{
		storage:       normalTableMarker + "\n<table class=\"old\"><tbody><tr><td>old</td></tr></tbody></table>",
		updateVersion: 3,
	}
	service := NewApplyService(uploader, pages)

	request := validApplyRequest(t)
	request.Repositories[0].MockTests = []domain.ApplyMockTest{
		{
			Function:       "Save",
			Scenario:       "Menyimpan data",
			ExpectedResult: "Data tersimpan",
			InputData:      "Payload",
			Code:           "func TestSave(t *testing.T) {\n\tmockRepo := new(MockRepository)\n\tmockRepo.AssertExpectations(t)\n}",
		},
	}

	if _, err := service.Apply(context.Background(), "user-1", "123456", request); err != nil {
		t.Fatalf("expected the apply to succeed, got %v", err)
	}
	if strings.Contains(pages.updated, "old") {
		t.Fatalf("expected the old table to be replaced, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, `<ac:structured-macro ac:name="code">`) {
		t.Fatalf("expected a code macro, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, `<ac:parameter ac:name="language">go</ac:parameter>`) {
		t.Fatalf("expected the go language parameter, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, "<![CDATA[") || !strings.Contains(pages.updated, "func TestSave") {
		t.Fatalf("expected the code body inside CDATA, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, "Normal Test Scenario") {
		t.Fatalf("expected the table marker, got %q", pages.updated)
	}
}

func TestApplyServiceReplacesPreviouslyWrittenCodeMacros(t *testing.T) {
	uploader := &stubUploader{returnID: "att-1"}
	previous := BuildMockTestCodeMacro("func TestOld(t *testing.T) {}")
	pages := &stubPageStorage{
		storage:       normalTableMarker + "\n<table><tbody><tr><td>old</td></tr></tbody></table>\n" + previous,
		updateVersion: 4,
	}
	service := NewApplyService(uploader, pages)

	request := validApplyRequest(t)
	request.Repositories[0].MockTests = []domain.ApplyMockTest{{Function: "Save", Code: "func TestSave(t *testing.T) {}"}}

	if _, err := service.Apply(context.Background(), "user-1", "123456", request); err != nil {
		t.Fatalf("expected the apply to succeed, got %v", err)
	}
	if strings.Contains(pages.updated, "TestOld") {
		t.Fatalf("expected the previous code macro to be replaced, got %q", pages.updated)
	}
	if !strings.Contains(pages.updated, "func TestSave") {
		t.Fatalf("expected the new code macro, got %q", pages.updated)
	}
	if strings.Count(pages.updated, `<ac:structured-macro ac:name="code"`) != 1 {
		t.Fatalf("expected a single code macro, got %q", pages.updated)
	}
}

func TestBuildMockTestCodeMacroSplitsTheCDATATerminator(t *testing.T) {
	macro := BuildMockTestCodeMacro("a ]]> b")
	if strings.Contains(macro, "a ]]> b") {
		t.Fatalf("expected the raw CDATA terminator to be split, got %q", macro)
	}
	if !strings.Contains(macro, "]]]]><![CDATA[>") {
		t.Fatalf("expected the CDATA split sequence, got %q", macro)
	}
	if !strings.HasSuffix(macro, "</ac:structured-macro>") {
		t.Fatalf("expected a closed macro, got %q", macro)
	}
}

func TestBuildMockTestCodeMacroBoundsLongCode(t *testing.T) {
	macro := BuildMockTestCodeMacro(strings.Repeat("a", maxMockTestCodeLength+50))
	body := strings.SplitN(macro, "<![CDATA[", 2)
	if len(body) != 2 {
		t.Fatalf("expected a CDATA body, got %q", macro)
	}
	code := strings.SplitN(body[1], "]]>", 2)[0]
	if len([]rune(code)) != maxMockTestCodeLength {
		t.Fatalf("expected the code to be bounded to %d runes, got %d", maxMockTestCodeLength, len([]rune(code)))
	}
}
