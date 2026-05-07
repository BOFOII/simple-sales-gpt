package tools

import (
	"context"
	"fmt"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
)

const ProductSearchToolName = "product_search"

type ProductSearchInput struct {
	Query string
}

func ProductSearchTool(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Tool {
	knowledgeLanguage := knowledgeBase.Language()
	return salesgpt.Tool{
		Name:        ProductSearchToolName,
		Description: "Search and suggest relevant products from the knowledge base. Use this when the customer asks for product price, product details, product features, product suitability, a named product, a product recommendation, has a product need, has not chosen a specific product yet, says they do not know what product they want, has no preference, describes a problem, mentions an old or unsuitable current product, or another tool is missing a product parameter. This tool can be run even when the customer does not provide a product name.",
		Parameters: []salesgpt.ToolParameter{
			{
				Name:        "query",
				Type:        "string",
				Description: fmt.Sprintf("Generative retrieval query. Always write this query in the knowledge-base language: %s. This is independent from the customer response language. If the customer uses another language, translate product names, needs, and keywords into %s before calling this tool. If the customer has not named a product or has no preference, create a concise query from their goal, pain points, current product condition, intended use, budget, promotion interest, or conversation context so the tool can suggest suitable products. Never ask the customer for a query.", knowledgeLanguage, knowledgeLanguage),
				Required:    true,
			},
		},
		Handler: ProductSearch(knowledgeBase, searchLimit...),
	}
}

func ProductSearch(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Handler {
	limit := firstPositive(searchLimit, 5)

	return func(ctx context.Context, input map[string]any) (any, error) {
		if knowledgeBase == nil {
			return nil, fmt.Errorf("knowledge base is required")
		}

		query, _ := input["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query is required")
		}

		return knowledgeBase.VectorStore().SearchDocuments(
			ctx,
			salesgpt.VectorCategoryProduct,
			query,
			limit,
		)
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
