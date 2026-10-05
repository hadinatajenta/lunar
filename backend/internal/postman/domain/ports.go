package domain

import "context"

type ProviderCredentials struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

type GenerateRequest struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
	System   string
	User     string
}

type FileSelector interface {
	Select(repoPath string, service string) (Selection, error)
}

type ServiceClassifier interface {
	Classify(selection Selection) Classification
}

type CollectionGenerator interface {
	Generate(ctx context.Context, request GenerateRequest) (string, error)
}
