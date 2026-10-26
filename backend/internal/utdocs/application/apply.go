package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/domain"
)

const (
	maxApplyRepositories         = 50
	maxApplyRowsPerRepository    = 500
	maxApplyScreenshotBytes      = 4 << 20
	maxMockTestCodeLength        = 20000
	screenshotContentType        = "image/png"
	base64DataURIPrefixSeparator = ","
)

type AttachmentUploader interface {
	UploadAttachment(ctx context.Context, userID string, pageID string, filename string, content []byte, mediaType string) (string, error)
}

type PageStorage interface {
	GetPageStorage(ctx context.Context, userID string, pageID string) (string, error)
	UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error)
}

type ApplyService struct {
	uploader AttachmentUploader
	pages    PageStorage
}

type preparedRepository struct {
	name      string
	filename  string
	content   []byte
	rows      []domain.ApplyRow
	mockTests []domain.ApplyMockTest
}

func NewApplyService(uploader AttachmentUploader, pages PageStorage) *ApplyService {
	return &ApplyService{uploader: uploader, pages: pages}
}

func (s *ApplyService) Apply(ctx context.Context, userID string, documentID string, request domain.ApplyRequest) (*domain.ApplyResult, error) {
	if s.uploader == nil || s.pages == nil {
		return nil, errors.New("the unit test document apply service is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required to apply the document", sharedErrors.ErrUnauthorized)
	}

	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if len(request.Repositories) == 0 {
		return nil, fmt.Errorf("%w: at least one repository is required", sharedErrors.ErrBadRequest)
	}
	if len(request.Repositories) > maxApplyRepositories {
		return nil, fmt.Errorf("%w: at most %d repositories are allowed", sharedErrors.ErrBadRequest, maxApplyRepositories)
	}

	prepared, err := prepareRepositories(request.Repositories)
	if err != nil {
		return nil, err
	}

	for _, repository := range prepared {
		if _, uploadErr := s.uploader.UploadAttachment(ctx, userID, trimmedID, repository.filename, repository.content, screenshotContentType); uploadErr != nil {
			return nil, fmt.Errorf("upload the screenshot for %s: %w", repository.name, uploadErr)
		}
	}

	storage, err := s.pages.GetPageStorage(ctx, userID, trimmedID)
	if err != nil {
		return nil, err
	}

	rows := buildApplyRows(prepared)
	mockTestMacros := BuildMockTestCodeMacros(buildApplyMockTests(prepared))
	updatedStorage := ReplaceNormalTestScenarioBlock(storage, BuildNormalTestScenarioTable(rows), mockTestMacros)
	version, pageURL, err := s.pages.UpdatePageStorage(ctx, userID, trimmedID, updatedStorage)
	if err != nil {
		return nil, err
	}

	return &domain.ApplyResult{
		PageID:  trimmedID,
		Version: version,
		URL:     pageURL,
	}, nil
}

func prepareRepositories(repositories []domain.ApplyRepository) ([]preparedRepository, error) {
	prepared := make([]preparedRepository, 0, len(repositories))
	for _, repository := range repositories {
		name := strings.TrimSpace(repository.Repository)
		if name == "" {
			return nil, fmt.Errorf("%w: the repository name is required", sharedErrors.ErrBadRequest)
		}
		if len([]rune(name)) > maxCellRepositoryLength {
			return nil, fmt.Errorf("%w: the repository name is too long", sharedErrors.ErrBadRequest)
		}

		filename := strings.TrimSpace(repository.Screenshot.Filename)
		if !ValidAttachmentFilename(filename) {
			return nil, fmt.Errorf("%w: the screenshot filename must be a valid PNG name", sharedErrors.ErrBadRequest)
		}
		if BuildScreenCaptureCell(filename) == "" {
			return nil, fmt.Errorf("%w: the screenshot filename is not allowed", sharedErrors.ErrBadRequest)
		}

		content, err := decodeScreenshot(repository.Screenshot.ContentBase64)
		if err != nil {
			return nil, err
		}

		if len(repository.Rows) > maxApplyRowsPerRepository {
			return nil, fmt.Errorf("%w: at most %d rows per repository are allowed", sharedErrors.ErrBadRequest, maxApplyRowsPerRepository)
		}

		prepared = append(prepared, preparedRepository{
			name:      name,
			filename:  filename,
			content:   content,
			rows:      repository.Rows,
			mockTests: repository.MockTests,
		})
	}
	return prepared, nil
}

func decodeScreenshot(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: the screenshot content is required", sharedErrors.ErrBadRequest)
	}
	if index := strings.Index(trimmed, base64DataURIPrefixSeparator); index >= 0 {
		trimmed = strings.TrimSpace(trimmed[index+1:])
	}

	content, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: the screenshot is not valid base64", sharedErrors.ErrBadRequest)
	}
	if len(content) > maxApplyScreenshotBytes {
		return nil, fmt.Errorf("%w: the screenshot exceeds the 4 MiB limit", sharedErrors.ErrBadRequest)
	}
	if _, decodeErr := png.Decode(bytes.NewReader(content)); decodeErr != nil {
		return nil, fmt.Errorf("%w: the screenshot is not a valid PNG", sharedErrors.ErrBadRequest)
	}
	return content, nil
}

func buildApplyRows(prepared []preparedRepository) []domain.DocumentRow {
	rows := make([]domain.DocumentRow, 0)
	for _, repository := range prepared {
		cell := BuildScreenCaptureCell(repository.filename)
		for _, applyRow := range repository.rows {
			rows = append(rows, domain.DocumentRow{
				Repository:     repository.name,
				Function:       boundCellValue(applyRow.Function, maxCellFunctionLength),
				Scenario:       boundCellValue(applyRow.Scenario, maxCellScenarioLength),
				ExpectedResult: boundCellValue(applyRow.ExpectedResult, maxCellExpectedLength),
				InputData:      boundCellValue(applyRow.InputData, maxCellInputLength),
				Result:         boundCellValue(applyRow.Result, maxCellResultLength),
				ScreenCapture:  cell,
			})
		}
	}
	return rows
}

func buildApplyMockTests(prepared []preparedRepository) []domain.ApplyMockTest {
	mockTests := make([]domain.ApplyMockTest, 0)
	for _, repository := range prepared {
		mockTests = append(mockTests, repository.mockTests...)
	}
	return mockTests
}

func BuildMockTestCodeMacros(mockTests []domain.ApplyMockTest) string {
	macros := make([]string, 0, len(mockTests))
	for _, mockTest := range mockTests {
		if code := strings.TrimSpace(mockTest.Code); code != "" {
			macros = append(macros, BuildMockTestCodeMacro(code))
		}
	}
	return strings.Join(macros, "\n")
}

func BuildMockTestCodeMacro(code string) string {
	return `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[` +
		escapeCDATA(boundMockTestCode(code)) +
		`]]></ac:plain-text-body></ac:structured-macro>`
}

func boundMockTestCode(code string) string {
	runes := []rune(strings.TrimSpace(code))
	if len(runes) <= maxMockTestCodeLength {
		return string(runes)
	}
	return string(runes[:maxMockTestCodeLength])
}

func escapeCDATA(value string) string {
	return strings.ReplaceAll(value, "]]>", "]]]]><![CDATA[>")
}
