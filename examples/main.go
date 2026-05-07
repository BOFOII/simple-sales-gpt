package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	ctx := context.Background()
	agentAPIKey := fallbackEnv("SALES_GPT_AGENT_OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY"))
	embeddingAPIKey := fallbackEnv("SALES_GPT_EMBEDDING_OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY"))

	embedder, err := newOpenAIEmbedder("text-embedding-3-small", embeddingAPIKey)
	if err != nil {
		log.Fatalf("failed to create embedder: %v", err)
	}

	vectorStore, err := newChromaVectorStore(
		"sleep_haven",
		fallbackEnv("CHROMA_URL", "http://localhost:8000"),
		embedder,
	)
	if err != nil {
		log.Fatalf("failed to create vector store: %v", err)
	}

	model, err := newOpenAIJSONModel("gpt-4o-mini", agentAPIKey)
	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// 1 . Create a knowledge base
	knowledgeBase, err := salesgpt.NewKnowledgeBase("sleep_haven", salesgpt.KnowledgeBaseParams{
		Vector: vectorStore,
		Vectorizer: salesgpt.KnowledgeBaseVectorizerParams{
			Language:           "English",
			SemanticEnrichment: true,
			Model:              model,
		},
	})
	if err != nil {
		log.Fatalf("failed to create knowledge base: %v", err)
	}

	// 2. Register products and promotions in the knowledge base
	price := 1299.00
	if err := knowledgeBase.Products().RegisterWithContext(ctx, salesgpt.Product{
		ID:          "mattress-premier-hybrid",
		Name:        "Premier Hybrid Mattress",
		Description: "A premium hybrid mattress with cooling fabric, strong back support, and zoned lumbar support.",
		Price:       &price,
		ImageURL:    "https://example.com/images/premier-hybrid.jpg",
	}); err != nil {
		log.Fatalf("failed to register product: %v", err)
	}
	fmt.Println("registered product: mattress-premier-hybrid")

	if err := knowledgeBase.Promotions().RegisterWithContext(ctx, salesgpt.Promotion{
		ID:          "spring-sleep-sale",
		Name:        "Spring Sleep Sale",
		Description: "Save 20% on premium mattresses for a limited time.",
		ImageURL:    "https://example.com/images/spring-sale.jpg",
	}); err != nil {
		log.Fatalf("failed to register promotion: %v", err)
	}
	fmt.Println("registered promotion: spring-sleep-sale")

	// 3. Create stage sales agents
	stages := []salesgpt.Stage{
		{
			ID:          "1",
			Description: "Introduction: Start the conversation by introducing yourself and your company. Be polite and respectful while keeping the tone of the conversation professional. Your greeting should be welcoming. Always clarify in your greeting the reason why you are calling."},
		{
			ID:          "2",
			Description: "Qualification: Qualify the prospect by confirming if they are the right person to talk to regarding your product/service. Ensure that they have the authority to make purchasing decisions."},
		{
			ID:          "3",
			Description: "Value proposition: Briefly explain how your product/service can benefit the prospect. Focus on the unique selling points and value proposition of your product/service that sets it apart from competitors."},
		{
			ID:          "4",
			Description: "Needs analysis: Ask open-ended questions to uncover the prospect's needs and pain points. Listen carefully to their responses and take notes."},
		{
			ID:          "5",
			Description: "Solution presentation: Based on the prospect's needs, present your product/service as the solution that can address their pain points."},
		{
			ID:          "6",
			Description: "Objection handling: Address any objections that the prospect may have regarding your product/service. Be prepared to provide evidence or testimonials to support your claims."},
		{
			ID:          "7",
			Description: "Close: Ask for the sale by proposing a next step. This could be a demo, a trial or a meeting with decision-makers. Ensure to summarize what has been discussed and reiterate the benefits."},
		{
			ID:          "8",
			Description: "End conversation: It's time to end the call as there is nothing else to be said.",
		},
	}

	// 4. Create tools for the agents to use during the conversation. In this case, we will create a product search tool and a promotion search tool that the agents can use to retrieve relevant information from the knowledge base when talking to prospects.
	agentTools := []salesgpt.Tool{
		salesgpttools.ProductSearchTool(knowledgeBase, 5),
		salesgpttools.PromotionSearchTool(knowledgeBase, 5),
		exampletools.ProspectTool(knowledgeBase, 1),
	}

	// 5. Register interests levels for the agents to classify the prospect's buying intent during the conversation. This will help the agent to adapt its responses and sales approach accordingly.
	interests := []salesgpt.Interest{
		{
			ID:          "cold",
			Description: "The prospect is only asking general questions and is not showing buying intent yet.",
		},
		{
			ID:          "medium",
			Description: "The prospect asks deeper questions about price, product details, comparisons, or suitability.",
		},
		{
			ID:          "hot",
			Description: "The prospect is ready to buy or asks for the next purchasing step.",
		},
	}

	agentParams := salesgpt.AgentParams{
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
			Tools:     agentTools,
			Stages:    stages,
			Interests: interests,
		},
		Conversation: salesgpt.AgentConversationParams{
			ConversationStage: "1",
		},
	}

	// runScenario(ctx, "Scenario 1: product and promotion search", "conversation-product-promotion", agentParams, seedProductPromotionHistory)
	// runScenario(ctx, "Scenario 2: prospect tool fulfilled", "conversation-prospect", agentParams, seedProspectHistory)
	runScenario(ctx, "Scenario 3: prospect missing product", "conversation-missing-product", agentParams, seedMissingProductHistory)
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

func fallbackEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func debugVectorSearch(ctx context.Context, knowledgeBase *salesgpt.KnowledgeBase, category, query string) {
	docs, err := knowledgeBase.VectorStore().SearchDocuments(ctx, category, query, 5)
	if err != nil {
		log.Printf("debug vector search failed category=%s query=%q: %v", category, query, err)
		return
	}

	fmt.Printf("\n=== Debug Vector Search: %s ===\n", category)
	fmt.Println("query:", query)
	fmt.Println("documents:", len(docs))
	for index, doc := range docs {
		fmt.Printf("[%d] score=%.4f metadata=%+v\n", index+1, doc.Score, doc.Metadata)
		fmt.Println(doc.PageContent)
	}
}

func runScenario(
	ctx context.Context,
	title string,
	agentID string,
	params salesgpt.AgentParams,
	seedHistory func(context.Context, salesgpt.SalesGPT),
) {
	fmt.Printf("\n\n================ %s ================\n", title)

	agent, err := salesgpt.NewSalesGPT(agentID, params)
	if err != nil {
		log.Fatalf("failed to create agent %q: %v", agentID, err)
	}
	seedHistory(ctx, agent)

	state, err := agent.StepWithOptions(ctx, salesgpt.StepOptions{
		Invoke: true,
		Debug:  true,
	})
	if err != nil {
		log.Fatalf("failed to run agent step %q: %v", agentID, err)
	}

	fmt.Println("knowledge base:", agent.KnowledgeBase().ID())
	fmt.Println("agent:", agent.AgentID(), "uses", agent.KnowledgeBase().ID())
	if stage, ok := agent.ConversationStage(); ok {
		fmt.Println("agent stage:", stage.ID)
	}
	fmt.Println("agent tools:", len(agent.Tools().List()))
	fmt.Println("agent interests:", len(agent.Interests().List()))

	fmt.Println("\n=== Reasoning Output ===")
	fmt.Println(state.ReasoningOutput)

	fmt.Println("\n=== Tool Results ===")
	if len(state.ToolResults) == 0 {
		fmt.Println("no tools were executed")
	} else {
		for _, result := range state.ToolResults {
			fmt.Println("tool:", result.ToolName)
			fmt.Printf("params: %+v\n", result.Params)
			if result.Error != "" {
				fmt.Println("error:", result.Error)
				continue
			}
			printJSON("output", result.Output)
		}
	}

	fmt.Println("\n=== Missing Tool Parameters ===")
	if len(state.MissingToolParameters) == 0 {
		fmt.Println("no missing tool parameters")
	} else {
		printJSON("missing", state.MissingToolParameters)
	}

	fmt.Println("\n=== Missing Tool Question Plan ===")
	if state.MissingToolQuestionPlanOutput == "" {
		fmt.Println("no missing tool question plan")
	} else {
		fmt.Println(state.MissingToolQuestionPlanOutput)
		debugPlanMessages(ctx, agent)
	}

	fmt.Println("\n=== Final Response Output ===")
	if state.ResponseOutput == "" {
		fmt.Println("no final response output")
	} else {
		fmt.Println(state.ResponseOutput)
	}
}

func debugPlanMessages(ctx context.Context, agent salesgpt.SalesGPT) {
	messages, err := agent.ConversationHistory().Messages(ctx)
	if err != nil {
		log.Printf("failed to debug plan messages: %v", err)
		return
	}

	fmt.Println("\n=== Debug Plan Messages In History ===")
	found := false
	for index, message := range messages {
		if string(message.GetType()) != "generic" {
			continue
		}
		if named, ok := message.(interface{ GetName() string }); ok && named.GetName() != "missing_tool_question_plan" {
			continue
		}

		found = true
		fmt.Printf("[%d] type=%s content=%s\n", index+1, message.GetType(), message.GetContent())
	}
	if !found {
		fmt.Println("no plan messages found in history")
	}
}

func seedProductPromotionHistory(ctx context.Context, agent salesgpt.SalesGPT) {
	mustAddAIMessage(ctx, agent, "Hi, this is Ted from Sleep Haven. I am calling to understand whether you are looking for a better sleep solution or a new mattress.")
	mustAddUserMessage(ctx, agent, "Hi Ted. I am looking around because my back hurts when I wake up, but I do not know which mattress fits me.")
	mustAddAIMessage(ctx, agent, "I understand. Are you mainly looking for more back support, cooling comfort, or something softer?")
	mustAddUserMessage(ctx, agent, "Back support is the main thing. I also want to know the price and whether there is any current promo.")
}

func seedProspectHistory(ctx context.Context, agent salesgpt.SalesGPT) {
	seedProductPromotionHistory(ctx, agent)
	mustAddAIMessage(ctx, agent, "The Premier Hybrid Mattress is designed for strong back support and zoned lumbar support. We also currently have the Spring Sleep Sale promotion. If you are interested, may I have your name and phone number for follow up?")
	mustAddUserMessage(ctx, agent, "Yes, please follow up with me. My name is Maya Johnson and my phone number is +1 415 555 0198. I am interested in the Premier Hybrid Mattress with the Spring Sleep Sale promo.")
}

func seedMissingProductHistory(ctx context.Context, agent salesgpt.SalesGPT) {
	mustAddAIMessage(ctx, agent, "Hi, this is Ted from Sleep Haven. I am calling to understand whether you are looking for a better sleep solution or a new mattress.")
	mustAddUserMessage(ctx, agent, "Hi Ted. I want better sleep and I am interested if you have a good promotion.")
	mustAddAIMessage(ctx, agent, "We currently have the Spring Sleep Sale promotion. I can help match you with the right mattress. May I have your name and phone number for follow up?")
	mustAddUserMessage(ctx, agent, "Sure. My name is Jordan Lee and my phone number is +1 415 555 0144. I like the Spring Sleep Sale promo, but I am not sure which mattress I want yet.")
}

func mustAddAIMessage(ctx context.Context, agent salesgpt.SalesGPT, message string) {
	if err := agent.ConversationHistory().AddAIMessage(ctx, message); err != nil {
		log.Fatalf("failed to add ai message: %v", err)
	}
}

func mustAddUserMessage(ctx context.Context, agent salesgpt.SalesGPT, message string) {
	if err := agent.ConversationHistory().AddUserMessage(ctx, message); err != nil {
		log.Fatalf("failed to add user message: %v", err)
	}
}

func printJSON(label string, value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Printf("%s: %+v\n", label, value)
		return
	}

	fmt.Printf("%s: %s\n", label, encoded)
}
