package domain

import "context"

type IndexStore interface {
	SaveChunks(ctx context.Context, repoName string, chunks []CodeChunk) error
	SearchChunks(ctx context.Context, query string, repoFilter string, limit int) ([]CodeChunk, error)
	SearchChunksIn(ctx context.Context, query string, repoFilter []string, limit int) ([]CodeChunk, error)
	GetServiceDependencies(ctx context.Context, serviceName string) ([]ServiceDependency, error)
	SaveDependencies(ctx context.Context, repoName string, deps []ServiceDependency) error
	GetIndexStatus(ctx context.Context) ([]IndexStatus, error)
	GetAllServices(ctx context.Context) ([]ServiceNode, error)
	ClearRepo(ctx context.Context, repoName string) error
	SearchKnowledge(ctx context.Context, query string, limit int) ([]DomainKnowledgeItem, error)
	SaveKnowledge(ctx context.Context, items []DomainKnowledgeItem) error
	ClearKnowledge(ctx context.Context) error
	Close() error
}
