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
	responseContextNodeName         = "response_context"
	responseNodeName                = "response"
)

const responseContextConversationHistoryLimit = 20

const (
	HandoffPriorityNormal = "normal"
	HandoffPriorityUrgent = "urgent"
)

type salesGPTNodeState struct {
	Invoke                        bool
	Input                         string
	Context                       string
	ConversationHistory           string
	ReasoningPrompt               string
	ReasoningOutput               string
	Reasoning                     reasoningResult
	ToolResults                   []toolExecutionResult
	missingToolParameters         []missingToolParameter
	MissingToolQuestionPlanPrompt string
	MissingToolQuestionPlanOutput string
	ResponseContext               string
	ResponsePrompt                string
	ResponseOutput                string
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

type responseBubbleMessage struct {
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
		ResponseOutput:        state.ResponseOutput,
		Bubbles:               responseBubbles(state.ResponseOutput),
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
	var message responseBubbleMessage
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
	state.ConversationHistory = salesGPT.buildConversationHistory(messages)
	return state, nil
}

func (salesGPT *SalesGPT) buildContext(messages []llms.ChatMessage) string {
	var builder strings.Builder
	messages = lastMessages(messages, salesGPT.contextWindowSize)

	salesGPT.writeSalesAgentProfile(&builder)

	builder.WriteString("\nCURRENT CONVERSATION STAGE\n")
	if salesGPT.conversationStage == nil {
		builder.WriteString("No current stage is set.\n")
	} else {
		writeStage(&builder, *salesGPT.conversationStage)
	}

	builder.WriteString("\nCONVERSATION HISTORY\n")
	builder.WriteString(salesGPT.buildConversationHistory(messages))
	builder.WriteString("\n")

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

func (salesGPT *SalesGPT) writeSalesAgentProfile(builder *strings.Builder) {
	builder.WriteString("SALES AGENT PROFILE\n")
	builder.WriteString(fmt.Sprintf("Salesperson name: %s\n", salesGPT.salespersonName))
	builder.WriteString(fmt.Sprintf("Salesperson role: %s\n", salesGPT.salespersonRole))
	builder.WriteString(fmt.Sprintf("Company name: %s\n", salesGPT.companyName))
	builder.WriteString(fmt.Sprintf("Company business: %s\n", salesGPT.companyBusiness))
	builder.WriteString(fmt.Sprintf("Company values: %s\n", salesGPT.companyValues))
	builder.WriteString(fmt.Sprintf("Conversation purpose: %s\n", salesGPT.conversationPurpose))
	builder.WriteString(fmt.Sprintf("Language: %s\n", salesGPT.language))
}

func (salesGPT *SalesGPT) buildConversationHistory(messages []llms.ChatMessage) string {
	return salesGPT.buildConversationHistoryWithLimit(messages, salesGPT.contextWindowSize)
}

func (salesGPT *SalesGPT) buildConversationHistoryWithLimit(messages []llms.ChatMessage, limit int) string {
	var builder strings.Builder
	messages = lastMessages(messages, limit)

	if len(messages) == 0 {
		builder.WriteString("No conversation history yet.\n")
	} else {
		for _, message := range messages {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", message.GetType(), message.GetContent()))
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
		return state, fmt.Errorf("llm model is required for response node")
	}
	if strings.TrimSpace(state.ResponseContext) == "" {
		return state, fmt.Errorf("response context is required before response")
	}

	state.ResponsePrompt = newResponsePrompt(state.ResponseContext)
	response, err := llms.GenerateFromSinglePrompt(ctx, salesGPT.model, state.ResponsePrompt)
	if err != nil {
		return state, fmt.Errorf("failed to generate response output: %w", err)
	}

	state.ResponseOutput = strings.TrimSpace(response)
	if state.Invoke {
		if err := salesGPT.conversationHistory.AddAIMessage(ctx, responseHistoryContent(state.ResponseOutput)); err != nil {
			return state, fmt.Errorf("failed to save response to conversation history: %w", err)
		}
	}

	return state, nil
}

func (salesGPT *SalesGPT) responseContextNode(ctx context.Context, state salesGPTNodeState) (salesGPTNodeState, error) {
	messages, err := salesGPT.conversationHistory.Messages(ctx)
	if err != nil {
		return state, fmt.Errorf("failed to load conversation history for response context: %w", err)
	}

	state.ConversationHistory = salesGPT.buildConversationHistoryWithLimit(messages, responseContextConversationHistoryLimit)
	state.ResponseContext = salesGPT.buildResponseContext(state)

	return state, nil
}

func (salesGPT *SalesGPT) buildResponseContext(state salesGPTNodeState) string {
	var builder strings.Builder

	salesGPT.writeSalesAgentProfile(&builder)
	builder.WriteString("\n\nCONVERSATION HISTORY\n")
	if strings.TrimSpace(state.ConversationHistory) == "" {
		builder.WriteString("No conversation history yet.\n")
	} else {
		builder.WriteString(state.ConversationHistory)
		builder.WriteString("\n")
	}

	builder.WriteString("\n\nREASONING SUMMARY\n")
	builder.WriteString(fmt.Sprintf("- Response language: %s\n", state.Reasoning.Language))
	builder.WriteString(fmt.Sprintf("- Customer purpose: %s\n", state.Reasoning.Conversation.Purpose))
	builder.WriteString(fmt.Sprintf("- Conversation stage: %s\n", state.Reasoning.Conversation.Stage))
	builder.WriteString(fmt.Sprintf("- Interest level: %s\n", state.Reasoning.Conversation.Interest.Value))
	builder.WriteString(fmt.Sprintf("- Interest evidence: %s\n", state.Reasoning.Conversation.Interest.Reason))
	builder.WriteString(fmt.Sprintf("- Opening score: %d. %s Improvement: %s\n",
		state.Reasoning.Conversation.Score.Opening.Score,
		state.Reasoning.Conversation.Score.Opening.Reason,
		state.Reasoning.Conversation.Score.Opening.Improvement,
	))
	builder.WriteString(fmt.Sprintf("- Engagement score: %d. %s Improvement: %s\n",
		state.Reasoning.Conversation.Score.Engagement.Score,
		state.Reasoning.Conversation.Score.Engagement.Reason,
		state.Reasoning.Conversation.Score.Engagement.Improvement,
	))
	builder.WriteString(fmt.Sprintf("- Closing score: %d. %s Improvement: %s\n",
		state.Reasoning.Conversation.Score.Closing.Score,
		state.Reasoning.Conversation.Score.Closing.Reason,
		state.Reasoning.Conversation.Score.Closing.Improvement,
	))

	builder.WriteString("\nHANDOFF\n")
	if state.Reasoning.Handoff.Required {
		builder.WriteString(fmt.Sprintf("- Required: yes\n- Priority: %s\n- Reason: %s\n- Summary for human agent: %s\n",
			state.Reasoning.Handoff.Priority,
			state.Reasoning.Handoff.Reason,
			state.Reasoning.Handoff.Summary,
		))
	} else {
		builder.WriteString("- Required: no\n")
	}

	builder.WriteString("\nNEXT RESPONSE PLAN\n")
	if len(state.Reasoning.Plan.Actions) == 0 {
		builder.WriteString("- No planned actions.\n")
	} else {
		for index, action := range state.Reasoning.Plan.Actions {
			builder.WriteString(fmt.Sprintf("- Step %d: %s Reason: %s\n",
				index+1,
				action.Action,
				action.Rationale,
			))
		}
	}

	builder.WriteString("\nREASONED TOOL REQUESTS\n")
	if len(state.Reasoning.Tools) == 0 {
		builder.WriteString("- No tools requested by reasoning.\n")
	} else {
		for _, tool := range state.Reasoning.Tools {
			builder.WriteString(fmt.Sprintf("- Tool: %s\n", tool.ToolName))
			builder.WriteString(fmt.Sprintf("  Action: %s\n", tool.Action))
			builder.WriteString(fmt.Sprintf("  Reason: %s\n", tool.Reason))
			builder.WriteString(fmt.Sprintf("  Known parameters: %v\n", tool.Params))
			if len(tool.Missing) == 0 {
				builder.WriteString("  Missing parameters: none\n")
			} else {
				builder.WriteString("  Missing parameters:\n")
				for _, missing := range tool.Missing {
					required := "optional"
					if missing.Required {
						required = "required"
					}
					builder.WriteString(fmt.Sprintf("    - %s (%s)\n", missing.ParamName, required))
				}
			}
		}
	}

	builder.WriteString("\nEXECUTED TOOL RESULTS\n")
	if len(state.ToolResults) == 0 {
		builder.WriteString("- No tools were executed.\n")
	} else {
		for _, result := range state.ToolResults {
			builder.WriteString(fmt.Sprintf("- Tool: %s\n", result.ToolName))
			builder.WriteString(fmt.Sprintf("  Parameters: %v\n", result.Params))
			if strings.TrimSpace(result.Error) != "" {
				builder.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
				continue
			}
			builder.WriteString(fmt.Sprintf("  Result: %v\n", result.Output))
		}
	}

	builder.WriteString("\nMISSING-PARAMETER QUESTION PLAN\n")
	if strings.TrimSpace(state.MissingToolQuestionPlanOutput) == "" {
		builder.WriteString("No missing-parameter question plan.\n")
	} else {
		builder.WriteString(state.MissingToolQuestionPlanOutput)
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func responseHistoryContent(output string) string {
	var message responseBubbleMessage
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
