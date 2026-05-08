package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	ctx := context.Background()
	sessionID := "session-123"
	handoffActive := false

	model, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
	)
	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	knowledgeBase, err := salesgpt.NewKnowledgeBase("handoff-empty-knowledge")
	if err != nil {
		log.Fatalf("failed to create knowledge base: %v", err)
	}

	agent, err := salesgpt.NewSalesGPT(sessionID, salesgpt.AgentParams{
		Model:         model,
		KnowledgeBase: knowledgeBase,
		Profile: salesgpt.AgentProfileParams{
			SalespersonName:     "Sales Assistant",
			SalespersonRole:     "AI Sales Assistant",
			CompanyName:         "Example Company",
			CompanyBusiness:     "Example Company helps customers with product and service questions.",
			CompanyValues:       "Helpful, respectful, and customer-focused service.",
			ConversationPurpose: "help the customer and hand off to a human agent when needed.",
			Language:            "English",
		},
	})
	if err != nil {
		log.Fatalf("failed to create sales gpt agent: %v", err)
	}

	// 1. The user sends a message.
	userMessage := "I want to speak with a human agent."
	fmt.Println("user:", userMessage)

	// 2. Save the user message to ConversationHistory.
	if err := agent.ConversationHistory().AddUserMessage(ctx, userMessage); err != nil {
		log.Fatalf("failed to save user message: %v", err)
	}

	// 3. Run StepWithOptions while the bot is still active.
	state, err := agent.StepWithOptions(ctx, salesgpt.StepOptions{Invoke: true})
	if err != nil {
		log.Fatalf("failed to run bot step: %v", err)
	}
	fmt.Println("bot:", firstBubbleText(state))

	// 4. If handoff is required, save the status in external memory.
	if state.Handoff.Required {
		handoffActive = true
		fmt.Printf("handoff active: %t\n", handoffActive)
	}

	// 5. After handoff_active == true, do not call Step again.
	// Save the next user message and forward it to the human agent.
	nextUserMessage := "Please tell the human agent I need help today."
	if handoffActive {
		if err := agent.ConversationHistory().AddUserMessage(ctx, nextUserMessage); err != nil {
			log.Fatalf("failed to save next user message: %v", err)
		}
		fmt.Println("route to human:", nextUserMessage)
	}

	// 6. Insert the human agent message into the same history.
	humanMessage := "Hello, I am from the sales team. I can continue helping you."
	if err := agent.ConversationHistory().AddMessage(ctx, llms.GenericChatMessage{
		Role:    "human_agent",
		Name:    "Sales Team",
		Content: humanMessage,
	}); err != nil {
		log.Fatalf("failed to save human agent message: %v", err)
	}
	fmt.Println("human agent:", humanMessage)

	// 7. When the human agent returns control to the bot, disable handoff in
	// external memory. The next normal customer message can be processed again
	// with agent.StepWithOptions(ctx, salesgpt.StepOptions{Invoke: true}).
	handoffActive = false
	fmt.Printf("handoff active after return to bot: %t\n", handoffActive)

	resumeUserMessage := "I understand now. Can you recommend the next step?"
	if !handoffActive {
		if err := agent.ConversationHistory().AddUserMessage(ctx, resumeUserMessage); err != nil {
			log.Fatalf("failed to save resumed user message: %v", err)
		}

		state, err := agent.StepWithOptions(ctx, salesgpt.StepOptions{Invoke: true})
		if err != nil {
			log.Fatalf("failed to resume bot step: %v", err)
		}
		fmt.Println("bot resumed:", firstBubbleText(state))
	}
}

func firstBubbleText(state salesgpt.StepResult) string {
	if len(state.Bubbles) == 0 {
		return state.ResponseOutput
	}

	return state.Bubbles[0].Text
}
