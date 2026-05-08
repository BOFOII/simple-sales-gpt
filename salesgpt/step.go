package salesgpt

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
)

type StepOptions struct {
	Invoke bool
	Debug  bool
}

func DefaultStepOptions() StepOptions {
	return StepOptions{
		Invoke: true,
	}
}

func (salesGPT *SalesGPT) Step(ctx context.Context, invoke ...bool) (StepResult, error) {
	options := DefaultStepOptions()
	if len(invoke) > 0 {
		options.Invoke = invoke[0]
	}

	return salesGPT.StepWithOptions(ctx, options)
}

func (salesGPT *SalesGPT) StepWithOptions(ctx context.Context, options StepOptions) (StepResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	workflow := graph.NewStateGraph[salesGPTNodeState]()
	workflow.AddNode(contextBuilderNodeName, "Build sales conversation context", salesGPT.contextBuilderNode)
	workflow.AddNode(reasoningNodeName, "Generate sales reasoning JSON", salesGPT.reasoningNode)
	workflow.AddNode(reasoningParserNodeName, "Parse sales reasoning JSON", salesGPT.reasoningParserNode)
	workflow.AddNode(stageSelectorNodeName, "Select agent conversation stage", salesGPT.stageSelectorNode)
	workflow.AddNode(languageSelectorNodeName, "Select agent response language", salesGPT.languageSelectorNode)
	workflow.AddNode(toolExecutionNodeName, "Execute runnable sales tools", salesGPT.toolExecutionNode)
	workflow.AddNode(missingToolCollectorNodeName, "Collect missing tool parameters", salesGPT.missingToolCollectorNode)
	workflow.AddNode(missingToolQuestionPlanNodeName, "Plan questions for missing tool parameters", salesGPT.missingToolQuestionPlanNode)
	workflow.AddNode(responseNodeName, "Generate final customer response bubbles", salesGPT.responseNode)
	workflow.SetEntryPoint(contextBuilderNodeName)
	workflow.AddEdge(contextBuilderNodeName, reasoningNodeName)
	workflow.AddEdge(reasoningNodeName, reasoningParserNodeName)
	workflow.AddEdge(reasoningParserNodeName, stageSelectorNodeName)
	workflow.AddEdge(stageSelectorNodeName, languageSelectorNodeName)
	workflow.AddEdge(languageSelectorNodeName, toolExecutionNodeName)
	workflow.AddEdge(toolExecutionNodeName, missingToolCollectorNodeName)
	workflow.AddEdge(missingToolCollectorNodeName, missingToolQuestionPlanNodeName)
	workflow.AddEdge(missingToolQuestionPlanNodeName, responseNodeName)
	workflow.AddEdge(responseNodeName, graph.END)

	runnable, err := workflow.Compile()
	if err != nil {
		return StepResult{}, fmt.Errorf("failed to compile sales gpt graph: %w", err)
	}

	state, err := runnable.Invoke(ctx, salesGPTNodeState{
		Invoke: options.Invoke,
	})
	if err != nil {
		return newStepResult(state, options.Debug), fmt.Errorf("failed to run sales gpt graph: %w", err)
	}

	return newStepResult(state, options.Debug), nil
}
