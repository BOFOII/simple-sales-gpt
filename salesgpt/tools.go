package salesgpt

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Handler func(ctx context.Context, input map[string]any) (any, error)

type Tool struct {
	Name        string
	Description string
	Parameters  []ToolParameter
	Handler     Handler
}

type ToolParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewToolRegistry(tools ...Tool) (*Registry, error) {
	registry := &Registry{
		tools: make(map[string]Tool),
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return registry, err
		}
	}

	return registry, nil
}

func (registry *Registry) Register(tool Tool) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool %q handler is required", tool.Name)
	}
	for _, parameter := range tool.Parameters {
		if parameter.Name == "" {
			return fmt.Errorf("tool %q parameter name is required", tool.Name)
		}
		if parameter.Description == "" {
			return fmt.Errorf("tool %q parameter %q description is required", tool.Name, parameter.Name)
		}
	}
	if _, exists := registry.tools[tool.Name]; exists {
		return fmt.Errorf("tool %q is already registered", tool.Name)
	}

	registry.tools[tool.Name] = cloneTool(tool)
	return nil
}

func (registry *Registry) List() []Tool {
	if registry == nil {
		return nil
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	tools := make([]Tool, 0, len(registry.tools))
	for _, tool := range registry.tools {
		tools = append(tools, cloneTool(tool))
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

func (registry *Registry) Get(name string) (Tool, bool) {
	if registry == nil {
		return Tool{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	tool, exists := registry.tools[name]
	return cloneTool(tool), exists
}

func cloneTool(tool Tool) Tool {
	tool.Parameters = append([]ToolParameter(nil), tool.Parameters...)
	return tool
}
