package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"

	exampletools "github.com/BOFOII/simple-sales-gpt/examples/tools"
	"github.com/BOFOII/simple-sales-gpt/salesgpt"
	salesgpttools "github.com/BOFOII/simple-sales-gpt/salesgpt/tools"
)

type finalResponse struct {
	Bubbles []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL string `json:"image_url"`
		Alt      string `json:"alt"`
	} `json:"bubbles"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	ctx := context.Background()
	debug := fallbackEnv("SALES_GPT_CHATBOT_DEBUG", "true") != "false"
	agent, err := newChatbotAgent()
	if err != nil {
		log.Fatalf("failed to initialize chatbot: %v", err)
	}

	if err := agent.ConversationHistory().AddAIMessage(ctx, "Hi, this is Ted from Sleep Haven. I am here to help you find a better sleep solution or the right mattress."); err != nil {
		log.Fatalf("failed to seed initial ai message: %v", err)
	}

	fmt.Println("SalesGPT chatbot is ready. Type your message, or type /exit to quit.")
	fmt.Println("Ted: Hi, this is Ted from Sleep Haven. I am here to help you find a better sleep solution or the right mattress.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "/exit" || input == "/quit" {
			fmt.Println("Ted: Thanks for chatting. Have a restful day.")
			return
		}

		if err := agent.ConversationHistory().AddUserMessage(ctx, input); err != nil {
			log.Printf("failed to add user message: %v", err)
			continue
		}

		state, err := agent.StepWithOptions(ctx, salesgpt.StepOptions{
			Invoke: true,
			Debug:  debug,
		})
		if err != nil {
			log.Printf("failed to run step: %v", err)
			continue
		}

		if debug {
			printTurnDebug(agent, state)
		}
		printBubbles(state.ResponseOutput)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read terminal input: %v", err)
	}
}

func newChatbotAgent() (salesgpt.SalesGPT, error) {
	agentAPIKey := fallbackEnv("SALES_GPT_AGENT_OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY"))
	embeddingAPIKey := fallbackEnv("SALES_GPT_EMBEDDING_OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY"))

	embedder, err := newOpenAIEmbedder("text-embedding-3-small", embeddingAPIKey)
	if err != nil {
		return salesgpt.SalesGPT{}, err
	}

	vectorStore, err := newChromaVectorStore(
		"sleep_haven_chatbot",
		fallbackEnv("CHROMA_URL", "http://localhost:8000"),
		embedder,
	)
	if err != nil {
		return salesgpt.SalesGPT{}, err
	}

	model, err := newOpenAIJSONModel("gpt-4o-mini", agentAPIKey)
	if err != nil {
		return salesgpt.SalesGPT{}, err
	}

	ctx := context.Background()
	knowledgeBase, err := salesgpt.NewKnowledgeBase("sleep_haven_chatbot", salesgpt.KnowledgeBaseParams{
		Vector: vectorStore,
		Vectorizer: salesgpt.KnowledgeBaseVectorizerParams{
			Language:           "English",
			SemanticEnrichment: true,
			Model:              model,
		},
	})
	if err != nil {
		return salesgpt.SalesGPT{}, err
	}

	price := 1299.00
	if err := knowledgeBase.Products().RegisterWithContext(ctx, salesgpt.Product{
		ID:          "mattress-premier-hybrid",
		Name:        "Premier Hybrid Mattress",
		Description: "A premium hybrid mattress with cooling fabric, strong back support, and zoned lumbar support.",
		Price:       &price,
		ImageURL:    "https://example.com/images/premier-hybrid.jpg",
	}); err != nil {
		return salesgpt.SalesGPT{}, err
	}
	if err := knowledgeBase.Promotions().RegisterWithContext(ctx, salesgpt.Promotion{
		ID:          "spring-sleep-sale",
		Name:        "Spring Sleep Sale",
		Description: "Save 20% on premium mattresses for a limited time.",
		ImageURL:    "https://example.com/images/spring-sale.jpg",
	}); err != nil {
		return salesgpt.SalesGPT{}, err
	}

	return salesgpt.NewSalesGPT(fmt.Sprintf("terminal-%d", time.Now().Unix()), salesgpt.AgentParams{
		KnowledgeBase: knowledgeBase,
		Model:         model,
		Profile: salesgpt.AgentProfileParams{
			SalespersonName:     "Ted Lasso",
			SalespersonRole:     "Business Development Representative",
			CompanyName:         "Sleep Haven",
			CompanyBusiness:     "Sleep Haven is a premium mattress company that provides comfortable and supportive mattresses, pillows, and bedding accessories.",
			CompanyValues:       "Sleep Haven helps customers achieve better sleep through quality products and helpful customer service.",
			ConversationPurpose: "find out whether the customer is looking to achieve better sleep by buying a premium mattress.",
			Language:            "English",
		},
		Registry: salesgpt.AgentRegistryParams{
			Tools: []salesgpt.Tool{
				salesgpttools.ProductSearchTool(knowledgeBase, 5),
				salesgpttools.PromotionSearchTool(knowledgeBase, 5),
				exampletools.ProspectTool(knowledgeBase, 1),
			},
			Stages:    defaultStages(),
			Interests: defaultInterests(),
		},
		Conversation: salesgpt.AgentConversationParams{
			ConversationStage: "1",
		},
	})
}

func newOpenAIJSONModel(modelName, apiKey string) (llms.Model, error) {
	options := []openai.Option{
		openai.WithModel(modelName),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	}
	if apiKey != "" {
		options = append(options, openai.WithToken(apiKey))
	}

	return openai.New(options...)
}

func newOpenAIEmbedder(modelName, apiKey string) (embeddings.Embedder, error) {
	options := []openai.Option{
		openai.WithEmbeddingModel(modelName),
	}
	if apiKey != "" {
		options = append(options, openai.WithToken(apiKey))
	}

	llm, err := openai.New(options...)
	if err != nil {
		return nil, err
	}

	return embeddings.NewEmbedder(llm)
}

func newChromaVectorStore(namespace, chromaURL string, embedder embeddings.Embedder) (vectorstores.VectorStore, error) {
	return chroma.New(
		chroma.WithChromaURL(chromaURL),
		chroma.WithNameSpace(namespace),
		chroma.WithEmbedder(embedder),
	)
}

func defaultStages() []salesgpt.Stage {
	return []salesgpt.Stage{
		{ID: "1", Description: "Introduction: Introduce yourself and the company, and clarify why you are calling."},
		{ID: "2", Description: "Qualification: Confirm whether the prospect is the right person and can make buying decisions."},
		{ID: "3", Description: "Value proposition: Explain how the product or service can benefit the prospect."},
		{ID: "4", Description: "Needs analysis: Ask open-ended questions to uncover needs and pain points."},
		{ID: "5", Description: "Solution presentation: Present the product or service as a solution to the prospect's needs."},
		{ID: "6", Description: "Objection handling: Address objections and provide supporting evidence."},
		{ID: "7", Description: "Close: Propose the next step and summarize the benefits."},
		{ID: "8", Description: "End conversation: End the call when there is nothing else to be said."},
	}
}

func defaultInterests() []salesgpt.Interest {
	return []salesgpt.Interest{
		{ID: "cold", Description: "The prospect is only asking general questions and is not showing buying intent yet."},
		{ID: "medium", Description: "The prospect asks deeper questions about price, product details, comparisons, or suitability."},
		{ID: "hot", Description: "The prospect is ready to buy or asks for the next purchasing step."},
	}
}

func printBubbles(output string) {
	var response finalResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil || len(response.Bubbles) == 0 {
		fmt.Println("Ted:", output)
		return
	}

	for _, bubble := range response.Bubbles {
		switch bubble.Type {
		case "image":
			if strings.TrimSpace(bubble.ImageURL) == "" {
				continue
			}
			if strings.TrimSpace(bubble.Alt) == "" {
				fmt.Println("Ted [image]:", bubble.ImageURL)
			} else {
				fmt.Printf("Ted [image: %s]: %s\n", bubble.Alt, bubble.ImageURL)
			}
		default:
			if strings.TrimSpace(bubble.Text) == "" {
				continue
			}
			fmt.Println("Ted:", bubble.Text)
		}
	}
}

func printTurnDebug(agent salesgpt.SalesGPT, state salesgpt.StepResult) {
	fmt.Println("\n--- debug ---")
	fmt.Println("language:", state.Language)
	fmt.Println("agent language:", agent.Language())
	fmt.Println("stage:", state.Stage)
	fmt.Println("interest:", state.Interest)
	fmt.Printf("score: opening=%d engagement=%d closing=%d\n", state.Score.Opening.Score, state.Score.Engagement.Score, state.Score.Closing.Score)

	if len(state.PlanActions) > 0 {
		fmt.Println("plan actions:")
		for index, action := range state.PlanActions {
			fmt.Printf("  %d. %s\n", index+1, action.Action)
		}
	}

	if len(state.Tools) == 0 {
		fmt.Println("reasoning tools: none")
	} else {
		fmt.Println("reasoning tools:")
		for _, tool := range state.Tools {
			params, _ := json.Marshal(tool.Params)
			fmt.Printf("  - %s action=%s params=%s missing=%+v\n", tool.ToolName, tool.Action, params, tool.Missing)
		}
	}

	if len(state.ToolResults) == 0 {
		fmt.Println("tool results: none")
	} else {
		fmt.Println("tool results:")
		for _, result := range state.ToolResults {
			fmt.Printf("  - %s", result.ToolName)
			if result.Error != "" {
				fmt.Printf(" error=%s\n", result.Error)
				continue
			}
			fmt.Printf(" output=%s\n", compactJSON(result.Output))
		}
	}

	if len(state.MissingToolParameters) > 0 {
		fmt.Println("missing:", compactJSON(state.MissingToolParameters))
	}
	if strings.TrimSpace(state.MissingToolQuestionPlanOutput) != "" {
		fmt.Println("missing plan:", state.MissingToolQuestionPlanOutput)
	}
	fmt.Println("--- end debug ---")
}

func compactJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%+v", value)
	}

	return string(encoded)
}

func fallbackEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
