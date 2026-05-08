package salesgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

const (
	contextBuilderNodeName          = "context_builder"
	reasoningNodeName               = "reasoning"
	reasoningParserNodeName         = "reasoning_parser"
	stageSelectorNodeName           = "stage_selector"
	languageSelectorNodeName        = "language_selector"
	toolExecutionNodeName           = "tool_execution"
	missingToolCollectorNodeName    = "missing_tool_collector"
	missingToolQuestionPlanNodeName = "missing_tool_question_plan"
	responseNodeName                = "final_response"
)

const (
	HandoffPriorityNormal = "normal"
	HandoffPriorityUrgent = "urgent"
)

type salesGPTNodeState struct {
	Invoke                        bool
	Input                         string
	Context                       string
	ReasoningPrompt               string
	ReasoningOutput               string
	Reasoning                     reasoningResult
	ToolResults                   []toolExecutionResult
	missingToolParameters         []missingToolParameter
	MissingToolQuestionPlanPrompt string
	MissingToolQuestionPlanOutput string
	FinalResponsePrompt           string
	FinalResponseOutput           string
}

type StepResult struct {
	Invoked                       bool
	ResponseOutput                string
	Bubbles                       []ResponseBubble
	ReasoningOutput               string
	Language                      string
	Stage                         string
	Interest                      string
	Handoff                       StepHandoff
	Score                         StepScore
	PlanActions                   []StepPlanAction
	Tools                         []StepReasoningTool
	ToolResults                   []StepToolResult
	MissingToolParameters         []StepMissingToolParameter
	MissingToolQuestionPlanOutput string
}

type ResponseBubble struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Alt      string `json:"alt,omitempty"`
}

type StepScore struct {
	Opening    StepScoreDetail
	Engagement StepScoreDetail
	Closing    StepScoreDetail
}

type StepScoreDetail struct {
	Score       int
	Reason      string
	Improvement string
}

type StepPlanAction struct {
	Action    string
	Rationale string
}

type StepHandoff struct {
	Required bool
	Reason   string
	Priority string
	Summary  string
}

type StepReasoningTool struct {
	ToolName string
	Reason   string
	Action   string
	Params   map[string]any
	Missing  []StepMissingToolParameter
}

type StepToolResult struct {
	ToolName string
	Params   map[string]any
	Output   any
	Error    string
}

type StepMissingToolParameter struct {
	ToolName  string `json:"tool_name,omitempty"`
	Reason    string `json:"reason,omitempty"`
	ParamName string `json:"param_name"`
	Required  bool   `json:"required"`
}

type toolExecutionResult struct {
	ToolName string
	Params   map[string]any
	Output   any
	Error    string
}

type finalResponseBubbleMessage struct {
	Bubbles []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL string `json:"image_url"`
		Alt      string `json:"alt"`
	} `json:"bubbles"`
}

type missingToolParameter struct {
	ToolName  string `json:"tool_name"`
	Reason    string `json:"reason"`
	ParamName string `json:"param_name"`
	Required  bool   `json:"required"`
}

type reasoningResult struct {
	Conversation reasoningConversation `json:"conversation"`
	Language     string                `json:"language"`
	Handoff      reasoningHandoff      `json:"handoff"`
	Plan         reasoningPlan         `json:"plan"`
	Tools        []reasoningTool       `json:"tools"`
}

type reasoningConversation struct {
	Purpose  string            `json:"purpose"`
	Stage    string            `json:"stage"`
	Interest reasoningInterest `json:"interest"`
	Score    reasoningScore    `json:"score"`
}

type reasoningInterest struct {
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type reasoningScore struct {
	Opening    reasoningScoreDetail `json:"opening"`
	Engagement reasoningScoreDetail `json:"engagement"`
	Closing    reasoningScoreDetail `json:"closing"`
}

type reasoningScoreDetail struct {
	Score       int    `json:"score"`
	Reason      string `json:"reason"`
	Improvement string `json:"improvement"`
}

type reasoningHandoff struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
	Priority string `json:"priority"`
	Summary  string `json:"summary"`
}

type reasoningPlan struct {
	Actions []reasoningPlanAction `json:"actions"`
}

type reasoningPlanAction struct {
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
}

type reasoningTool struct {
	ToolName string                 `json:"tool_name"`
	Reason   string                 `json:"reason"`
	Action   string                 `json:"action"`
	Params   map[string]any         `json:"params"`
	Missing  []reasoningMissingTool `json:"missing"`
}

type reasoningMissingTool struct {
	ParamName string `json:"param_name"`
	Required  bool   `json:"required"`
}

func newStepResult(state salesGPTNodeState, debug bool) StepResult {
	result := StepResult{
		Invoked:               state.Invoke,
		ResponseOutput:        state.FinalResponseOutput,
		Bubbles:               responseBubbles(state.FinalResponseOutput),
		Language:              state.Reasoning.Language,
		Stage:                 state.Reasoning.Conversation.Stage,
		Interest:              state.Reasoning.Conversation.Interest.Value,
		Handoff:               stepHandoff(state.Reasoning.Handoff),
		Score:                 stepScore(state.Reasoning.Conversation.Score),
		PlanActions:           stepPlanActions(state.Reasoning.Plan.Actions),
		Tools:                 stepReasoningTools(state.Reasoning.Tools),
		ToolResults:           stepToolResults(state.ToolResults),
		MissingToolParameters: stepMissingToolParameters(state.missingToolParameters),
	}
	if debug {
		result.ReasoningOutput = state.ReasoningOutput
		result.MissingToolQuestionPlanOutput = state.MissingToolQuestionPlanOutput
	}

	return result
}

func stepHandoff(handoff reasoningHandoff) StepHandoff {
	return StepHandoff{
		Required: handoff.Required,
		Reason:   handoff.Reason,
		Priority: handoff.Priority,
		Summary:  handoff.Summary,
	}
}

func stepScore(score reasoningScore) StepScore {
	return StepScore{
		Opening:    stepScoreDetail(score.Opening),
		Engagement: stepScoreDetail(score.Engagement),
		Closing:    stepScoreDetail(score.Closing),
	}
}

func stepScoreDetail(detail reasoningScoreDetail) StepScoreDetail {
	return StepScoreDetail{
		Score:       detail.Score,
		Reason:      detail.Reason,
		Improvement: detail.Improvement,
	}
}

func responseBubbles(output string) []ResponseBubble {
	var message finalResponseBubbleMessage
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		return nil
	}

	bubbles := make([]ResponseBubble, 0, len(message.Bubbles))
	for _, bubble := range message.Bubbles {
		bubbles = append(bubbles, ResponseBubble{
			Type:     bubble.Type,
			Text:     bubble.Text,
			ImageURL: bubble.ImageURL,
			Alt:      bubble.Alt,
		})
	}

	return bubbles
}

func stepPlanActions(actions []reasoningPlanAction) []StepPlanAction {
	stepActions := make([]StepPlanAction, 0, len(actions))
	for _, action := range actions {
		stepActions = append(stepActions, StepPlanAction{
			Action:    action.Action,
			Rationale: action.Rationale,
		})
	}

	return stepActions
}

func stepReasoningTools(tools []reasoningTool) []StepReasoningTool {
	stepTools := make([]StepReasoningTool, 0, len(tools))
	for _, tool := range tools {
		stepTools = append(stepTools, StepReasoningTool{
			ToolName: tool.ToolName,
			Reason:   tool.Reason,
			Action:   tool.Action,
			Params:   tool.Params,
			Missing:  stepReasoningMissingTools(tool.Missing),
		})
	}

	return stepTools
}

func stepReasoningMissingTools(missingTools []reasoningMissingTool) []StepMissingToolParameter {
	missing := make([]StepMissingToolParameter, 0, len(missingTools))
	for _, item := range missingTools {
		missing = append(missing, StepMissingToolParameter{
			ParamName: item.ParamName,
			Required:  item.Required,
		})
	}

	return missing
}

func stepToolResults(results []toolExecutionResult) []StepToolResult {
	stepResults := make([]StepToolResult, 0, len(results))
	for _, result := range results {
		stepResults = append(stepResults, StepToolResult{
			ToolName: result.ToolName,
			Params:   result.Params,
			Output:   result.Output,
			Error:    result.Error,
		})
	}

	return stepResults
}

func stepMissingToolParameters(parameters []missingToolParameter) []StepMissingToolParameter {
	stepParameters := make([]StepMissingToolParameter, 0, len(parameters))
	for _, parameter := range parameters {
		stepParameters = append(stepParameters, StepMissingToolParameter{
			ToolName:  parameter.ToolName,
			Reason:    parameter.Reason,
			ParamName: parameter.ParamName,
			Required:  parameter.Required,
		})
	}

	return stepParameters
}

func (salesGPT *SalesGPT) contextBuilderNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	messages, err := salesGPT.conversationHistory.Messages(ctx)
	if err != nil {
		return state, fmt.Errorf("failed to load conversation history: %w", err)
	}

	state.Context = salesGPT.buildContext(messages)
	return state, nil
}

func (salesGPT *SalesGPT) buildContext(messages []llms.ChatMessage) string {
	var builder strings.Builder
	messages = lastMessages(messages, salesGPT.contextWindowSize)

	builder.WriteString("SALES AGENT PROFILE\n")
	builder.WriteString(fmt.Sprintf("Salesperson name: %s\n", salesGPT.salespersonName))
	builder.WriteString(fmt.Sprintf("Salesperson role: %s\n", salesGPT.salespersonRole))
	builder.WriteString(fmt.Sprintf("Company name: %s\n", salesGPT.companyName))
	builder.WriteString(fmt.Sprintf("Company business: %s\n", salesGPT.companyBusiness))
	builder.WriteString(fmt.Sprintf("Company values: %s\n", salesGPT.companyValues))
	builder.WriteString(fmt.Sprintf("Conversation purpose: %s\n", salesGPT.conversationPurpose))
	builder.WriteString(fmt.Sprintf("Language: %s\n", salesGPT.language))

	builder.WriteString("\nCURRENT CONVERSATION STAGE\n")
	if salesGPT.conversationStage == nil {
		builder.WriteString("No current stage is set.\n")
	} else {
		writeStage(&builder, *salesGPT.conversationStage)
	}

	builder.WriteString("\nCONVERSATION HISTORY\n")
	if len(messages) == 0 {
		builder.WriteString("No conversation history yet.\n")
	} else {
		for _, message := range messages {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", message.GetType(), message.GetContent()))
		}
	}

	builder.WriteString("\nAVAILABLE TOOLS\n")
	tools := salesGPT.tools.List()
	if len(tools) == 0 {
		builder.WriteString("No tools are registered.\n")
	} else {
		for _, tool := range tools {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
			if len(tool.Parameters) == 0 {
				builder.WriteString("  Parameters: none\n")
				continue
			}
			builder.WriteString("  Parameters:\n")
			for _, parameter := range tool.Parameters {
				required := "optional"
				if parameter.Required {
					required = "required"
				}
				builder.WriteString(fmt.Sprintf("  - %s (%s, %s): %s\n", parameter.Name, parameter.Type, required, parameter.Description))
			}
		}
	}

	builder.WriteString("\nREGISTERED STAGES\n")
	stages := salesGPT.stages.List()
	if len(stages) == 0 {
		builder.WriteString("No stages are registered.\n")
	} else {
		for _, stage := range stages {
			writeStage(&builder, stage)
		}
	}

	builder.WriteString("\nINTEREST LEVELS\n")
	interests := salesGPT.interests.List()
	if len(interests) == 0 {
		builder.WriteString("No interest levels are registered.\n")
	} else {
		for _, interest := range interests {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", interest.ID, interest.Description))
		}
	}

	return strings.TrimSpace(builder.String())
}

func lastMessages(messages []llms.ChatMessage, limit int) []llms.ChatMessage {
	if limit <= 0 || len(messages) <= limit {
		return messages
	}

	return messages[len(messages)-limit:]
}

func writeStage(builder *strings.Builder, stage Stage) {
	builder.WriteString(fmt.Sprintf("- %s: %s\n", stage.ID, stage.Description))
	if stage.Tone != "" {
		builder.WriteString(fmt.Sprintf("  Tone: %s\n", stage.Tone))
	}
}

func (salesGPT *SalesGPT) reasoningNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	if strings.TrimSpace(state.Context) == "" {
		return state, fmt.Errorf("context is required before reasoning")
	}
	if salesGPT.model == nil {
		return state, fmt.Errorf("llm model is required for reasoning node")
	}

	state.ReasoningPrompt = newReasoningNodePrompt(state.Context)
	response, err := llms.GenerateFromSinglePrompt(ctx, salesGPT.model, state.ReasoningPrompt)
	if err != nil {
		return state, fmt.Errorf("failed to generate reasoning output: %w", err)
	}

	state.ReasoningOutput = strings.TrimSpace(response)
	return state, nil
}

func (salesGPT *SalesGPT) reasoningParserNode(_ context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	if strings.TrimSpace(state.ReasoningOutput) == "" {
		return state, fmt.Errorf("reasoning output is required before parsing")
	}

	var reasoning reasoningResult
	if err := json.Unmarshal([]byte(state.ReasoningOutput), &reasoning); err != nil {
		return state, fmt.Errorf("failed to parse reasoning output: %w", err)
	}

	state.Reasoning = reasoning
	return state, nil
}

func (salesGPT *SalesGPT) stageSelectorNode(_ context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	stageID := strings.TrimSpace(state.Reasoning.Conversation.Stage)
	if stageID == "" {
		return state, nil
	}

	stage, exists := salesGPT.stages.Get(stageID)
	if !exists {
		return state, nil
	}

	salesGPT.conversationStage = &stage
	return state, nil
}

func (salesGPT *SalesGPT) languageSelectorNode(_ context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	language := strings.TrimSpace(state.Reasoning.Language)
	if language == "" {
		return state, nil
	}

	salesGPT.language = language
	return state, nil
}

func (salesGPT *SalesGPT) toolExecutionNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	var results []toolExecutionResult

	for _, reasoningTool := range state.Reasoning.Tools {
		if reasoningTool.Action != "run" {
			continue
		}

		result := toolExecutionResult{
			ToolName: reasoningTool.ToolName,
			Params:   reasoningTool.Params,
		}

		tool, exists := salesGPT.tools.Get(reasoningTool.ToolName)
		if !exists {
			result.Error = fmt.Sprintf("tool %q is not registered", reasoningTool.ToolName)
			results = append(results, result)
			continue
		}

		output, err := tool.Handler(ctx, reasoningTool.Params)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Output = output
		}

		results = append(results, result)
	}

	state.ToolResults = results
	return state, nil
}

func (salesGPT *SalesGPT) missingToolCollectorNode(_ context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	var missingParameters []missingToolParameter

	for _, reasoningTool := range state.Reasoning.Tools {
		if reasoningTool.Action != "ask" {
			continue
		}

		for _, missing := range reasoningTool.Missing {
			missingParameters = append(missingParameters, missingToolParameter{
				ToolName:  reasoningTool.ToolName,
				Reason:    reasoningTool.Reason,
				ParamName: missing.ParamName,
				Required:  missing.Required,
			})
		}
	}

	state.missingToolParameters = missingParameters
	return state, nil
}

func (salesGPT *SalesGPT) missingToolQuestionPlanNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	if len(state.missingToolParameters) == 0 {
		return state, nil
	}
	if salesGPT.model == nil {
		return state, fmt.Errorf("llm model is required for missing tool question plan node")
	}

	missingParameters, err := json.MarshalIndent(state.missingToolParameters, "", "  ")
	if err != nil {
		return state, fmt.Errorf("failed to encode missing tool parameters: %w", err)
	}

	state.MissingToolQuestionPlanPrompt = newMissingToolQuestionPlanPrompt(string(missingParameters))
	response, err := llms.GenerateFromSinglePrompt(ctx, salesGPT.model, state.MissingToolQuestionPlanPrompt)
	if err != nil {
		return state, fmt.Errorf("failed to generate missing tool question plan output: %w", err)
	}

	state.MissingToolQuestionPlanOutput = strings.TrimSpace(response)
	if err := salesGPT.conversationHistory.AddMessage(ctx, llms.GenericChatMessage{
		Role:    "plan",
		Name:    missingToolQuestionPlanNodeName,
		Content: state.MissingToolQuestionPlanOutput,
	}); err != nil {
		return state, fmt.Errorf("failed to save missing tool question plan to conversation history: %w", err)
	}

	return state, nil
}

func (salesGPT *SalesGPT) responseNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	if salesGPT.model == nil {
		return state, fmt.Errorf("llm model is required for final response node")
	}

	toolResults, err := json.MarshalIndent(state.ToolResults, "", "  ")
	if err != nil {
		return state, fmt.Errorf("failed to encode tool results for final response: %w", err)
	}
	reasoning, err := json.MarshalIndent(state.Reasoning, "", "  ")
	if err != nil {
		return state, fmt.Errorf("failed to encode reasoning for final response: %w", err)
	}

	state.FinalResponsePrompt = newFinalResponsePrompt(
		salesGPT.finalResponseProfile(),
		string(reasoning),
		string(toolResults),
		state.MissingToolQuestionPlanOutput,
	)
	response, err := llms.GenerateFromSinglePrompt(ctx, salesGPT.model, state.FinalResponsePrompt)
	if err != nil {
		return state, fmt.Errorf("failed to generate final response output: %w", err)
	}

	state.FinalResponseOutput = strings.TrimSpace(response)
	if state.Invoke {
		if err := salesGPT.conversationHistory.AddAIMessage(ctx, responseHistoryContent(state.FinalResponseOutput)); err != nil {
			return state, fmt.Errorf("failed to save final response to conversation history: %w", err)
		}
	}

	return state, nil
}

func responseHistoryContent(output string) string {
	var message finalResponseBubbleMessage
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		return output
	}

	bubbles := make([]string, 0, len(message.Bubbles))
	for _, bubble := range message.Bubbles {
		switch bubble.Type {
		case "", "text":
			text := strings.TrimSpace(bubble.Text)
			if text != "" {
				bubbles = append(bubbles, text)
			}
		case "image":
			imageURL := strings.TrimSpace(bubble.ImageURL)
			if imageURL == "" {
				continue
			}
			alt := strings.TrimSpace(bubble.Alt)
			if alt == "" {
				alt = "image"
			}
			bubbles = append(bubbles, fmt.Sprintf("[Image sent: %s | %s]", alt, imageURL))
		}
	}
	if len(bubbles) == 0 {
		return output
	}

	return strings.Join(bubbles, "\n")
}

func (salesGPT *SalesGPT) finalResponseProfile() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Salesperson name: %s\n", salesGPT.salespersonName))
	builder.WriteString(fmt.Sprintf("Salesperson role: %s\n", salesGPT.salespersonRole))
	builder.WriteString(fmt.Sprintf("Company name: %s\n", salesGPT.companyName))
	builder.WriteString(fmt.Sprintf("Company business: %s\n", salesGPT.companyBusiness))
	builder.WriteString(fmt.Sprintf("Company values: %s\n", salesGPT.companyValues))
	builder.WriteString(fmt.Sprintf("Conversation purpose: %s\n", salesGPT.conversationPurpose))
	builder.WriteString(fmt.Sprintf("Language: %s", salesGPT.language))

	return builder.String()
}
