package salesgpt

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

const (
	defaultSalespersonName     = "Sales Agent"
	defaultSalespersonRole     = "Sales Representative"
	defaultCompanyName         = "Your Company"
	defaultCompanyBusiness     = "Your Company provides products and services to help customers solve their needs."
	defaultCompanyValues       = "Your Company is committed to helpful, respectful, and customer-focused service."
	defaultConversationPurpose = "understand the customer's needs and help them find a suitable solution."
	defaultLanguage            = "English"
	defaultContextWindowSize   = 20
)

type SalesGPT struct {
	agentID             string
	model               llms.Model
	salespersonName     string
	salespersonRole     string
	companyName         string
	companyBusiness     string
	companyValues       string
	conversationPurpose string
	language            string
	conversationStage   *Stage
	conversationHistory schema.ChatMessageHistory
	contextWindowSize   int
	tools               *Registry
	stages              *StageRegistry
	interests           *InterestRegistry
	knowledgeBase       *KnowledgeBase
}

type AgentParams struct {
	Model         llms.Model
	Profile       AgentProfileParams
	Conversation  AgentConversationParams
	Registry      AgentRegistryParams
	KnowledgeBase *KnowledgeBase
}

type AgentProfileParams struct {
	SalespersonName     string
	SalespersonRole     string
	CompanyName         string
	CompanyBusiness     string
	CompanyValues       string
	ConversationPurpose string
	Language            string
}

type AgentConversationParams struct {
	ConversationStage   string
	ConversationHistory []llms.ChatMessage
	ContextWindowSize   int
}

type AgentRegistryParams struct {
	Tools     []Tool
	Stages    []Stage
	Interests []Interest
}

func NewSalesGPT(agentID string, params ...AgentParams) (SalesGPT, error) {
	param := AgentParams{}
	if len(params) > 0 {
		param = params[0]
	}
	if agentID == "" {
		return SalesGPT{}, fmt.Errorf("agent id is required")
	}

	knowledgeBase := param.KnowledgeBase
	if knowledgeBase == nil {
		return SalesGPT{}, fmt.Errorf("knowledge base is required")
	}

	if param.Model == nil {
		return SalesGPT{}, fmt.Errorf("llm model is required")
	}

	registry, err := NewToolRegistry(param.Registry.Tools...)
	if err != nil {
		return SalesGPT{}, fmt.Errorf("failed to create tool registry: %w", err)
	}

	stages, err := NewStageRegistry(param.Registry.Stages...)
	if err != nil {
		return SalesGPT{}, fmt.Errorf("failed to create stage registry: %w", err)
	}

	interests, err := NewInterestRegistry(param.Registry.Interests...)
	if err != nil {
		return SalesGPT{}, fmt.Errorf("failed to create interest registry: %w", err)
	}

	history := memory.NewChatMessageHistory(
		memory.WithPreviousMessages(param.Conversation.ConversationHistory),
	)
	conversationStage := resolveConversationStage(stages, param.Conversation.ConversationStage)

	salesGPT := SalesGPT{
		agentID:             agentID,
		model:               param.Model,
		salespersonName:     fallbackString(param.Profile.SalespersonName, defaultSalespersonName),
		salespersonRole:     fallbackString(param.Profile.SalespersonRole, defaultSalespersonRole),
		companyName:         fallbackString(param.Profile.CompanyName, defaultCompanyName),
		companyBusiness:     fallbackString(param.Profile.CompanyBusiness, defaultCompanyBusiness),
		companyValues:       fallbackString(param.Profile.CompanyValues, defaultCompanyValues),
		conversationPurpose: fallbackString(param.Profile.ConversationPurpose, defaultConversationPurpose),
		language:            fallbackString(param.Profile.Language, defaultLanguage),
		conversationStage:   conversationStage,
		conversationHistory: history,
		contextWindowSize:   fallbackPositiveInt(param.Conversation.ContextWindowSize, defaultContextWindowSize),
		tools:               registry,
		stages:              stages,
		interests:           interests,
		knowledgeBase:       knowledgeBase,
	}

	return salesGPT, nil
}

func (salesGPT SalesGPT) AgentID() string {
	return salesGPT.agentID
}

func (salesGPT SalesGPT) Model() llms.Model {
	return salesGPT.model
}

func (salesGPT SalesGPT) SalespersonName() string {
	return salesGPT.salespersonName
}

func (salesGPT SalesGPT) SalespersonRole() string {
	return salesGPT.salespersonRole
}

func (salesGPT SalesGPT) CompanyName() string {
	return salesGPT.companyName
}

func (salesGPT SalesGPT) CompanyBusiness() string {
	return salesGPT.companyBusiness
}

func (salesGPT SalesGPT) CompanyValues() string {
	return salesGPT.companyValues
}

func (salesGPT SalesGPT) ConversationPurpose() string {
	return salesGPT.conversationPurpose
}

func (salesGPT SalesGPT) Language() string {
	return salesGPT.language
}

func (salesGPT SalesGPT) ConversationStage() (Stage, bool) {
	if salesGPT.conversationStage == nil {
		return Stage{}, false
	}

	return *salesGPT.conversationStage, true
}

func (salesGPT SalesGPT) ConversationHistory() schema.ChatMessageHistory {
	return salesGPT.conversationHistory
}

func (salesGPT SalesGPT) ContextWindowSize() int {
	return salesGPT.contextWindowSize
}

func (salesGPT SalesGPT) Tools() *Registry {
	return salesGPT.tools
}

func (salesGPT SalesGPT) Stages() *StageRegistry {
	return salesGPT.stages
}

func (salesGPT SalesGPT) Interests() *InterestRegistry {
	return salesGPT.interests
}

func (salesGPT SalesGPT) KnowledgeBase() *KnowledgeBase {
	return salesGPT.knowledgeBase
}

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func fallbackPositiveInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}

	return value
}

func resolveConversationStage(registry *StageRegistry, stageID string) *Stage {
	if stageID == "" {
		return nil
	}

	stage, exists := registry.Get(stageID)
	if !exists {
		return nil
	}

	return &stage
}
