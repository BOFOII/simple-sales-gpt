# Simple Sales GPT

Simple Sales GPT is a modular Go package for building sales conversation agents.
It provides a small set of primitives for creating an agent session, registering
conversation stages, attaching tools, storing conversation history, and searching
business knowledge through a vector store.

The project uses [LangChainGo](https://github.com/tmc/langchaingo) for LLM,
embedding, memory, and vector store integrations. It is inspired by
[SalesGPT](https://github.com/filip-michalsky/SalesGPT).

## Features

- One agent session per conversation.
- Shared knowledge base for products, promotions, and vector search.
- Configurable `llms.Model`; the core package does not hardcode a model provider.
- Configurable `vectorstores.VectorStore`; use Chroma, Milvus, Pinecone, or any
  LangChainGo-compatible vector store.
- Tool registry with named tools, descriptions, parameters, and handlers.
- Conversation stages and interest levels.
- Conversation history powered by LangChainGo memory.
- Optional semantic enrichment before storing products or promotions in vectors.
- Step-based execution for reasoning, tool selection, tool execution, and final
  response generation.
- MIT licensed.

## Installation

```bash
go get github.com/BOFOII/simple-sales-gpt
```

## Core Concepts

### Knowledge Base

A `KnowledgeBase` contains business data that can be shared by many agent
sessions.

```text
KnowledgeBase: sleep_haven
  - products
  - promotions
  - vector store
```

Products and promotions registered in the knowledge base can be stored in the
vector store and later retrieved by tools.

### Agent Session

A `SalesGPT` agent represents one conversation session.

```text
Agent session
  - conversation id
  - conversation history
  - current stage
  - registered tools
  - shared knowledge base
```

For multiple conversations, create multiple agents that share the same knowledge
base.

## Quick Start

```go
package main

import (
	"context"

	"github.com/BOFOII/simple-sales-gpt/salesgpt"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	model, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)
	if err != nil {
		panic(err)
	}

	// Create your vector store outside the core package.
	// See examples/main.go for a Chroma setup.
	var vectorStore = createYourVectorStore()

	knowledgeBase, err := salesgpt.NewKnowledgeBase("sleep_haven", salesgpt.KnowledgeBaseParams{
		Vector: vectorStore,
		Vectorizer: salesgpt.KnowledgeBaseVectorizerParams{
			Language:           "English",
			SemanticEnrichment: true,
			Model:              model,
		},
	})
	if err != nil {
		panic(err)
	}

	price := 1299.00
	err = knowledgeBase.Products().RegisterWithContext(ctx, salesgpt.Product{
		ID:          "mattress-premier-hybrid",
		Name:        "Premier Hybrid Mattress",
		Description: "A premium hybrid mattress with cooling fabric and zoned lumbar support.",
		Price:       &price,
	})
	if err != nil {
		panic(err)
	}

	agent, err := salesgpt.NewSalesGPT("conversation-001", salesgpt.AgentParams{
		Model:         model,
		KnowledgeBase: knowledgeBase,
		Profile: salesgpt.AgentProfileParams{
			SalespersonName:     "Ted",
			SalespersonRole:     "Sales Representative",
			CompanyName:         "Sleep Haven",
			ConversationPurpose: "help the customer find a suitable sleep solution.",
			Language:            "English",
		},
		Conversation: salesgpt.AgentConversationParams{
			ConversationStage: "1",
		},
	})
	if err != nil {
		panic(err)
	}

	_ = agent.ConversationHistory().AddUserMessage(ctx, "I need a better mattress for back support.")

	result, err := agent.StepWithOptions(ctx, salesgpt.StepOptions{
		Invoke: true,
		Debug:  true,
	})
	if err != nil {
		panic(err)
	}

	for _, bubble := range result.Bubbles {
		println(bubble.Type, bubble.Text, bubble.ImageURL)
	}
}
```

## Tools

Tools are registered per agent. A tool has a name, description, parameter
schema, and handler.

```go
tool := salesgpt.Tool{
	Name:        "prospect",
	Description: "Parse customer data into prospect data.",
	Parameters: []salesgpt.ToolParameter{
		{Name: "name", Type: "string", Description: "Customer name.", Required: true},
		{Name: "phone_number", Type: "string", Description: "Customer phone number.", Required: true},
	},
	Handler: func(ctx context.Context, input map[string]any) (any, error) {
		// Send prospect data to your CRM or API.
		return nil, nil
	},
}
```

The package also includes product and promotion search tools in
`salesgpt/tools`.

## Examples

Run the scenario example:

```bash
go run examples/main.go
```

Run the terminal chatbot:

```bash
go run examples/chatbot/main.go
```

The examples use OpenAI and Chroma through LangChainGo. Set these environment
variables as needed:

```bash
OPENAI_API_KEY=...
SALES_GPT_AGENT_OPENAI_API_KEY=...
SALES_GPT_EMBEDDING_OPENAI_API_KEY=...
CHROMA_URL=http://localhost:8000
```

If `SALES_GPT_AGENT_OPENAI_API_KEY` or `SALES_GPT_EMBEDDING_OPENAI_API_KEY` is
not set, the examples fall back to `OPENAI_API_KEY`.

## Architecture

```text
Customer message
  -> Conversation history
  -> Context builder
  -> Reasoning node
  -> Tool execution
  -> Missing parameter planning
  -> Final response bubbles
```

The final response is returned as bubbles:

```json
{
  "bubbles": [
    { "type": "text", "text": "Here is a suitable option." },
    { "type": "image", "image_url": "https://example.com/product.jpg", "alt": "Product image" }
  ]
}
```

## Test Result

```text
SalesGPT chatbot is ready. Type your message, or type /exit to quit.
Ted: Hi, this is Ted from Sleep Haven. I am here to help you find a better sleep solution or the right mattress.

You: hallo, can you help me
Ted: To help you find the right mattress, could you share what specific sleep issues you're experiencing?

You: i dont have specific sleep issue
Ted: To help you find the perfect mattress, could you share your general preferences? For example, do you prefer a firmer or softer feel?
Ted: Also, to ensure we can follow up with you effectively, could you please share your name and phone number?

You: I think a soft mattress would suit me better.
Ted: I found a great option for you! The Premier Hybrid Mattress offers ultimate comfort with advanced cooling fabric and strong back support.
Ted [image: Premier Hybrid Mattress]: https://example.com/images/premier-hybrid.jpg
Ted: To ensure we can follow up with you effectively, could you please share your name and phone number?

You: I forgot to tell you my name, my name is Riski and I want to know how much the mattress costs?
Ted: The price of the Premier Hybrid Mattress is $1299.00. It offers excellent support and comfort, perfect for enhancing your sleep quality.
Ted: To ensure we can follow up with you effectively, could you please provide your phone number?

You: I will tell you my phone number later after I buy it, let me know what the specifications are.
Ted: The Premier Hybrid Mattress offers advanced cooling fabric, strong back support, and zoned lumbar support for optimal comfort. It's designed to promote proper spinal alignment and reduce pressure points, making it ideal for anyone looking to enhance their sleep quality.
Ted: At a price of $1299, this mattress combines the support of innerspring coils with the comfort of memory foam, adapting to your sleeping position.
Ted [image: Premier Hybrid Mattress]: https://example.com/images/premier-hybrid.jpg
You: I will buy this
Ted: I'm excited to help you with your purchase of the Premier Hybrid Mattress! To ensure you receive the best service and follow-up, could you please provide your phone number?

You: my phone number is 0812345678977
Ted: Great to hear you're ready to purchase the Premier Hybrid Mattress, Riski! Before we finalize everything, could you let me know if you'd like to take advantage of any current promotions?

You: are there any promotions? it would be better if yes
Ted: Great news, Riski! We currently have a "Spring Sleep Sale" where you can enjoy a fantastic 20% discount on the Premier Hybrid Mattress.
Ted: This limited-time offer is perfect for enhancing your sleep experience. Would you like to proceed with this promotion?
Ted [image: Spring Sleep Sale promotion image]: https://example.com/images/spring-sale.jpg

You: how much is the mattress after the discount?
Ted: The Premier Hybrid Mattress is currently available for you at a fantastic price after applying the 20% discount. Let me confirm the final price for you.
Ted: Also, just to ensure everything is set for your purchase, could you please confirm your preferred payment method?
Ted [image: Spring Sleep Sale promotion image]: https://example.com/images/spring-sale.jpg

You: I don't know what I'll pay for yet, but I'd like to know the price of the mattress after the discount.
Ted: The final price of the Premier Hybrid Mattress after applying the 20% discount is $1,199. Are you ready to proceed with the purchase?
Ted: Also, could you please let me know your preferred payment method?
Ted [image: Spring Sleep Sale promotion image]: https://example.com/images/spring-sale.jpg

You: 
```

## Project Status

This project was built with help from Codex. It was developed without a
dedicated `SKILL.md`, `AGENTS.md`, or other custom AI guidance files that would
strictly shape how the AI assistant should design and review the code.

Because of that, some parts of the package may still need deeper review,
cleanup, testing, API refinement, and production hardening. Please treat this
project as an early open-source package and expect rough edges.

Contributions are very welcome. If you find inefficient code, unclear APIs,
missing tests, unsafe behavior, or better architectural patterns, please open an
issue or pull request.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

## Acknowledgements

- Built on top of [LangChainGo](https://github.com/tmc/langchaingo).
- Inspired by [SalesGPT](https://github.com/filip-michalsky/SalesGPT).
