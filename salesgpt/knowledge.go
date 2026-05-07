package salesgpt

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores"
)

type KnowledgeBase struct {
	id          string
	language    string
	products    *ProductRegistry
	promotions  *PromotionRegistry
	vectorizer  *Vectorizer
	vectorStore *VectorStoreRegistry
}

func (knowledgeBase *KnowledgeBase) ID() string {
	return knowledgeBase.id
}

func (knowledgeBase *KnowledgeBase) Language() string {
	if knowledgeBase == nil {
		return ""
	}

	return knowledgeBase.language
}

func (knowledgeBase *KnowledgeBase) Products() *ProductRegistry {
	return knowledgeBase.products
}

func (knowledgeBase *KnowledgeBase) Promotions() *PromotionRegistry {
	return knowledgeBase.promotions
}

func (knowledgeBase *KnowledgeBase) VectorStore() *VectorStoreRegistry {
	return knowledgeBase.vectorStore
}

type KnowledgeBaseParams struct {
	Catalog    KnowledgeBaseCatalogParams
	Vector     vectorstores.VectorStore
	Vectorizer KnowledgeBaseVectorizerParams
}

type KnowledgeBaseCatalogParams struct {
	Products   []Product
	Promotions []Promotion
}

type KnowledgeBaseVectorizerParams struct {
	Language           string
	SemanticEnrichment bool
	Model              llms.Model
}

func NewKnowledgeBase(id string, params ...KnowledgeBaseParams) (*KnowledgeBase, error) {
	param := KnowledgeBaseParams{}
	if len(params) > 0 {
		param = params[0]
	}
	if id == "" {
		return nil, fmt.Errorf("knowledge base id is required")
	}

	language := fallbackString(param.Vectorizer.Language, defaultLanguage)
	vectorStore := NewVectorStoreRegistry(param.Vector)
	var enrichmentModel llms.Model
	if param.Vectorizer.SemanticEnrichment {
		if param.Vectorizer.Model == nil {
			return nil, fmt.Errorf("semantic enrichment model is required")
		}
		enrichmentModel = param.Vectorizer.Model
	}
	vectorizer := NewVectorizer(
		param.Vectorizer.SemanticEnrichment,
		language,
		enrichmentModel,
	)
	products, err := NewProductRegistry(vectorStore, vectorizer, param.Catalog.Products...)
	if err != nil {
		return nil, fmt.Errorf("failed to create product registry: %w", err)
	}

	promotions, err := NewPromotionRegistry(vectorStore, vectorizer, param.Catalog.Promotions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create promotion registry: %w", err)
	}

	return &KnowledgeBase{
		id:          id,
		language:    language,
		products:    products,
		promotions:  promotions,
		vectorizer:  vectorizer,
		vectorStore: vectorStore,
	}, nil
}
