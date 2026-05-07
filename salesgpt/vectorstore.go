package salesgpt

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

const (
	VectorCategoryProduct   = "product"
	VectorCategoryPromotion = "promotion"
)

type VectorStoreRegistry struct {
	vectorStore vectorstores.VectorStore
}

func NewVectorStoreRegistry(vectorStore vectorstores.VectorStore) *VectorStoreRegistry {
	return &VectorStoreRegistry{
		vectorStore: vectorStore,
	}
}

func (registry *VectorStoreRegistry) AddDocuments(ctx context.Context, category string, docs []schema.Document) ([]string, error) {
	if registry == nil {
		return nil, fmt.Errorf("vector store registry is not configured")
	}
	if category == "" {
		return nil, fmt.Errorf("vector store category is required")
	}
	if err := registry.ensureStore(ctx); err != nil {
		return nil, err
	}

	return registry.vectorStore.AddDocuments(ctx, docs, vectorstores.WithNameSpace(category))
}

func (registry *VectorStoreRegistry) SearchDocuments(ctx context.Context, category, query string, limit int) ([]schema.Document, error) {
	if registry == nil {
		return nil, fmt.Errorf("vector store registry is not configured")
	}
	if category == "" {
		return nil, fmt.Errorf("vector store category is required")
	}
	if query == "" {
		return nil, fmt.Errorf("vector store query is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("vector store search limit must be greater than zero")
	}
	if err := registry.ensureStore(ctx); err != nil {
		return nil, err
	}

	return registry.vectorStore.SimilaritySearch(
		ctx,
		query,
		limit,
		vectorstores.WithNameSpace(category),
	)
}

func (registry *VectorStoreRegistry) ensureStore(_ context.Context) error {
	if registry.vectorStore != nil {
		return nil
	}

	return fmt.Errorf("vector store is required")
}
