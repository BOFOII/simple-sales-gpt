package salesgpt

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type Vectorizable interface {
	VectorID() string
	VectorCategory() string
	VectorContent() string
	VectorMetadata() map[string]any
}

type Vectorizer struct {
	semanticEnrichment bool
	language           string
	model              llms.Model
}

func NewVectorizer(semanticEnrichment bool, language string, model llms.Model) *Vectorizer {
	return &Vectorizer{
		semanticEnrichment: semanticEnrichment,
		language:           language,
		model:              model,
	}
}

func (vectorizer *Vectorizer) vectorize(ctx context.Context, item Vectorizable) (schema.Document, error) {
	if !vectorizer.semanticEnrichment {
		return vectorDocument(item, item.VectorContent()), nil
	}
	if vectorizer.model == nil {
		return schema.Document{}, fmt.Errorf("semantic enrichment model is not configured")
	}

	content, err := llms.GenerateFromSinglePrompt(ctx, vectorizer.model, semanticEnrichmentPrompt(item, vectorizer.language))
	if err != nil {
		return schema.Document{}, fmt.Errorf("failed to generate semantic enrichment: %w", err)
	}

	return vectorDocument(item, strings.TrimSpace(content)), nil
}

func (vectorizer *Vectorizer) Store(ctx context.Context, vectorStore *VectorStoreRegistry, item Vectorizable) ([]string, error) {
	return vectorizer.StoreMany(ctx, vectorStore, item.VectorCategory(), item)
}

func (vectorizer *Vectorizer) StoreMany(ctx context.Context, vectorStore *VectorStoreRegistry, category string, items ...Vectorizable) ([]string, error) {
	if vectorStore == nil {
		return nil, fmt.Errorf("vector store is required")
	}
	if category == "" {
		return nil, fmt.Errorf("vector category is required")
	}

	documents := make([]schema.Document, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.VectorCategory() != category {
			return nil, fmt.Errorf("vector item %q category %q does not match %q", item.VectorID(), item.VectorCategory(), category)
		}

		doc, err := vectorizer.vectorize(ctx, item)
		if err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}
	if len(documents) == 0 {
		return nil, nil
	}

	return vectorStore.AddDocuments(ctx, category, documents)
}

func vectorDocument(item Vectorizable, content string) schema.Document {
	metadata := item.VectorMetadata()
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["id"] = item.VectorID()
	metadata["category"] = item.VectorCategory()

	return schema.Document{
		PageContent: content,
		Metadata:    metadata,
	}
}

func semanticEnrichmentPrompt(item Vectorizable, language string) string {
	var builder strings.Builder
	builder.WriteString("Enrich this source data into a semantic search document. ")
	builder.WriteString("Use natural customer search language, benefits, use cases, relevant synonyms, and buying intent phrases. ")
	builder.WriteString("Do not invent facts that are not implied by the source data. ")
	builder.WriteString("Return plain text only. Do not return JSON, markdown, bullet labels, or explanations. ")
	if language != "" {
		builder.WriteString("Write in ")
		builder.WriteString(language)
		builder.WriteString(". ")
	}
	builder.WriteString("\nSource data:\n")
	builder.WriteString(item.VectorContent())

	return builder.String()
}
