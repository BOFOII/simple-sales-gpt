package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
	"github.com/BOFOII/simple-sales-gpt/salesgpt/parser"
)

const ProspectToolName = "prospect"

type ProspectInput struct {
	Name        string
	PhoneNumber string
	Product     string
	Promotion   string
}

func ProspectTool(knowledgeBase *salesgpt.KnowledgeBase, parserSearchLimit ...int) salesgpt.Tool {
	knowledgeLanguage := prospectKnowledgeBaseLanguage(knowledgeBase)
	return salesgpt.Tool{
		Name:        ProspectToolName,
		Description: "Tool that should always be called to parse customer data into prospect data. Include this tool as run when all required params are known. Include this tool as ask when some required params are still missing, especially for lead capture or follow-up.",
		Parameters: []salesgpt.ToolParameter{
			{
				Name:        "name",
				Type:        "string",
				Description: "Customer name that will be saved as the prospect name. Use the name from conversation history, or mark it as missing if the customer has not provided it.",
				Required:    true,
			},
			{
				Name:        "phone_number",
				Type:        "string",
				Description: "Customer phone number that can be used for follow up. Use the phone number from conversation history, or mark it as missing if the customer has not provided it.",
				Required:    true,
			},
			{
				Name:        "product",
				Type:        "string",
				Description: fmt.Sprintf("Product the customer is interested in. Infer it from conversation history when the customer mentions a specific product or clear product need. Always pass this value in the knowledge-base language: %s, not necessarily the customer response language. If the customer uses another language, translate or normalize the product name to the likely catalog name before calling this tool. Mark it as missing if no product interest is available yet.", knowledgeLanguage),
				Required:    true,
			},
			{
				Name:        "promotion",
				Type:        "string",
				Description: fmt.Sprintf("Promotion or offer the customer is interested in. Infer it from conversation history when the customer mentions a specific promotion, discount, or offer. Always pass this value in the knowledge-base language: %s, not necessarily the customer response language. If the customer uses another language, translate or normalize the promotion name to the likely catalog name before calling this tool. Mark it as missing if no promotion interest is available yet.", knowledgeLanguage),
				Required:    true,
			},
		},
		Handler: ProspectParser(knowledgeBase, parserSearchLimit...),
	}
}

func prospectKnowledgeBaseLanguage(knowledgeBase *salesgpt.KnowledgeBase) string {
	if knowledgeBase == nil {
		return "the knowledge-base/catalog language"
	}

	language := strings.TrimSpace(knowledgeBase.Language())
	if language == "" {
		return "the knowledge-base/catalog language"
	}

	return language
}

func ProspectParser(knowledgeBase *salesgpt.KnowledgeBase, parserSearchLimit ...int) salesgpt.Handler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		name, _ := input["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}

		phoneNumber, _ := input["phone_number"].(string)
		if phoneNumber == "" {
			return nil, fmt.Errorf("phone_number is required")
		}

		product, _ := input["product"].(string)
		if product == "" {
			return nil, fmt.Errorf("product is required")
		}
		parsedProduct, err := parser.ParseProduct(ctx, knowledgeBase, product, parserSearchLimit...)
		if err != nil {
			log.Printf("failed to parse prospect product %q: %v", product, err)
		}

		promotion, _ := input["promotion"].(string)
		if promotion == "" {
			return nil, fmt.Errorf("promotion is required")
		}
		parsedPromotion, err := parser.ParsePromotion(ctx, knowledgeBase, promotion, parserSearchLimit...)
		if err != nil {
			log.Printf("failed to parse prospect promotion %q: %v", promotion, err)
		}

		// todo : send to your api crm here or anything

		log.Print(parsedProduct)
		log.Print(parsedPromotion)

		return nil, nil
	}
}
