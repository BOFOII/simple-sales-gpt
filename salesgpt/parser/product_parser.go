package parser

import (
	"context"
	"fmt"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
)

func ParseProduct(ctx context.Context, knowledgeBase *salesgpt.KnowledgeBase, productName string, searchLimit ...int) (*salesgpt.Product, error) {
	if knowledgeBase == nil {
		return nil, fmt.Errorf("knowledge base is required")
	}
	if productName == "" {
		return nil, fmt.Errorf("product name is required")
	}

	documents, err := knowledgeBase.VectorStore().SearchDocuments(
		ctx,
		salesgpt.VectorCategoryProduct,
		productName,
		firstPositive(searchLimit, 1),
	)
	if err != nil {
		return nil, err
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("product %q was not found", productName)
	}

	productID, _ := documents[0].Metadata["product_id"].(string)
	if productID == "" {
		return nil, fmt.Errorf("product id was not found in vector metadata")
	}

	product, exists := knowledgeBase.Products().Get(productID)
	if !exists {
		return nil, fmt.Errorf("product %q was not found in knowledge base", productID)
	}

	return &product, nil
}

func ProductParser(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Handler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		productName, _ := input["product"].(string)
		return ParseProduct(ctx, knowledgeBase, productName, searchLimit...)
	}
}

func firstPositive(values []int, fallback int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return fallback
}
