package application

import (
	"context"
	"fmt"

	"lunar/backend/internal/postman/domain"
)

type Emitter func(event domain.StageEvent) error

type RunRequest struct {
	Service     string
	RepoPath    string
	Credentials domain.ProviderCredentials
}

type Result struct {
	Service      string             `json:"service"`
	RequestCount int                `json:"request_count"`
	FolderCount  int                `json:"folder_count"`
	Collection   *domain.Collection `json:"collection"`
}

type Preparation struct {
	Service        string
	Selection      domain.Selection
	Classification domain.Classification
	SystemPrompt   string
	UserPrompt     string
}

type Pipeline struct {
	selector   domain.FileSelector
	classifier domain.ServiceClassifier
	generator  domain.CollectionGenerator
}

func NewPipeline(selector domain.FileSelector, classifier domain.ServiceClassifier, generator domain.CollectionGenerator) *Pipeline {
	return &Pipeline{selector: selector, classifier: classifier, generator: generator}
}

func (p *Pipeline) Run(ctx context.Context, request RunRequest, emit Emitter) (Result, error) {
	preparation, err := p.Prepare(request.Service, request.RepoPath)
	if err != nil {
		return Result{}, err
	}
	return p.Generate(ctx, preparation, request.Credentials, emit)
}

func (p *Pipeline) Prepare(service string, repoPath string) (Preparation, error) {
	selection, err := p.selector.Select(repoPath, service)
	if err != nil {
		return Preparation{}, fmt.Errorf("prepare export: %w", err)
	}

	classification := p.classifier.Classify(selection)
	if !classification.IsExportable {
		return Preparation{}, fmt.Errorf("%w: %s", domain.ErrServiceNotExportable, classification.Reason)
	}
	if selection.ExceededFileLimit {
		return Preparation{}, fmt.Errorf("%w: the repository has more than %d candidate files", domain.ErrServiceNotExportable, domain.MaxSelectedFiles)
	}
	if selection.ExceededByteLimit {
		return Preparation{}, fmt.Errorf("%w: the selected files exceed %d bytes", domain.ErrServiceNotExportable, domain.MaxSelectedBytes)
	}

	return Preparation{
		Service:        service,
		Selection:      selection,
		Classification: classification,
		SystemPrompt:   SystemPrompt(classification),
		UserPrompt:     UserPrompt(service, selection),
	}, nil
}

func (p *Pipeline) Generate(ctx context.Context, preparation Preparation, credentials domain.ProviderCredentials, emit Emitter) (Result, error) {
	if err := emit(domain.StageEvent{
		Stage:  domain.StageSelecting,
		Detail: fmt.Sprintf("selected %d files from %s", len(preparation.Selection.Files), preparation.Service),
	}); err != nil {
		return Result{}, fmt.Errorf("emit selecting stage: %w", err)
	}
	if err := emit(domain.StageEvent{
		Stage:  domain.StagePrompt,
		Detail: fmt.Sprintf("built a prompt of %d bytes with the %s appendix", len(preparation.SystemPrompt)+len(preparation.UserPrompt), frameworkLabel(preparation.Classification)),
	}); err != nil {
		return Result{}, fmt.Errorf("emit prompt stage: %w", err)
	}
	if err := emit(domain.StageEvent{
		Stage:  domain.StageModel,
		Detail: fmt.Sprintf("awaiting %s from the %s provider", credentials.Model, credentials.Provider),
	}); err != nil {
		return Result{}, fmt.Errorf("emit model stage: %w", err)
	}

	generated, err := p.generator.Generate(ctx, domain.GenerateRequest{
		Provider: credentials.Provider,
		APIKey:   credentials.APIKey,
		BaseURL:  credentials.BaseURL,
		Model:    credentials.Model,
		System:   preparation.SystemPrompt,
		User:     preparation.UserPrompt,
	})
	if err != nil {
		return Result{}, fmt.Errorf("generate collection: %w", err)
	}

	if err := emit(domain.StageEvent{
		Stage:  domain.StageValidating,
		Detail: "validating the collection against the Postman v2.1 contract",
	}); err != nil {
		return Result{}, fmt.Errorf("emit validating stage: %w", err)
	}

	collection, err := ParseCollection(generated, preparation.Service)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Service:      preparation.Service,
		RequestCount: CountRequests(collection),
		FolderCount:  CountFolders(collection),
		Collection:   collection,
	}, nil
}

func frameworkLabel(classification domain.Classification) string {
	if classification.Framework == domain.FrameworkUnknown {
		return "Wire"
	}
	return string(classification.Framework)
}
