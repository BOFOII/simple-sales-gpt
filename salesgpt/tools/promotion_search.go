package tools

import (
	"context"
	"fmt"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
)

const PromotionSearchToolName = "promotion_search"

type PromotionSearchInput struct {
	Query string
}

func PromotionSearchTool(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Tool {
	knowledgeLanguage := knowledgeBase.Language()
	return salesgpt.Tool{
		Name:        PromotionSearchToolName,
		Description: "Search and suggest relevant promotions from the knowledge base. Use this when the customer asks about offers, discounts, promos, promotion availability, deals, affordability, price concerns, buying incentives, says they want the best deal, asks for product price plus promo, or another tool is missing a promotion parameter. This tool can be run even when the customer does not provide a promotion name.",
		Parameters: []salesgpt.ToolParameter{
			{
				Name:        "query",
				Type:        "string",
				Description: fmt.Sprintf("Generative retrieval query. Always write this query in the knowledge-base language: %s. This is independent from the customer response language. If the customer uses another language, translate promotion names, deal intent, product interest, and keywords into %s before calling this tool. If the customer has not named a promotion, create a concise query from their product interest, goal, needs, budget concern, discount request, timing, or conversation context so the tool can suggest suitable promotions. Never ask the customer for a query.", knowledgeLanguage, knowledgeLanguage),
				Required:    true,
			},
		},
		Handler: PromotionSearch(knowledgeBase, searchLimit...),
	}
}

func PromotionSearch(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Handler {
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
			salesgpt.VectorCategoryPromotion,
			query,
			limit,
		)
	}
}
