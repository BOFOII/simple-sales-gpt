package parser

import (
	"context"
	"fmt"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
)

func ParsePromotion(ctx context.Context, knowledgeBase *salesgpt.KnowledgeBase, promotionName string, searchLimit ...int) (*salesgpt.Promotion, error) {
	if knowledgeBase == nil {
		return nil, fmt.Errorf("knowledge base is required")
	}
	if promotionName == "" {
		return nil, fmt.Errorf("promotion name is required")
	}

	documents, err := knowledgeBase.VectorStore().SearchDocuments(
		ctx,
		salesgpt.VectorCategoryPromotion,
		promotionName,
		firstPositive(searchLimit, 1),
	)
	if err != nil {
		return nil, err
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("promotion %q was not found", promotionName)
	}

	promotionID, _ := documents[0].Metadata["promotion_id"].(string)
	if promotionID == "" {
		return nil, fmt.Errorf("promotion id was not found in vector metadata")
	}

	promotion, exists := knowledgeBase.Promotions().Get(promotionID)
	if !exists {
		return nil, fmt.Errorf("promotion %q was not found in knowledge base", promotionID)
	}

	return &promotion, nil
}

func PromotionParser(knowledgeBase *salesgpt.KnowledgeBase, searchLimit ...int) salesgpt.Handler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		promotionName, _ := input["promotion"].(string)
		return ParsePromotion(ctx, knowledgeBase, promotionName, searchLimit...)
	}
}
