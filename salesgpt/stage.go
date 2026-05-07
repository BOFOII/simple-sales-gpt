package salesgpt

import (
	"fmt"
	"sort"
	"sync"
)

type Stage struct {
	ID          string
	Description string
	Tone        string
}

type StageRegistry struct {
	mu     sync.RWMutex
	stages map[string]Stage
}

func NewStageRegistry(stages ...Stage) (*StageRegistry, error) {
	registry := &StageRegistry{
		stages: make(map[string]Stage),
	}

	for _, stage := range stages {
		if err := registry.Register(stage); err != nil {
			return registry, err
		}
	}

	return registry, nil
}
func (registry *StageRegistry) Register(stage Stage) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if stage.ID == "" {
		return fmt.Errorf("stage id is required")
	}
	if stage.Description == "" {
		return fmt.Errorf("stage %q description is required", stage.ID)
	}
	if _, exists := registry.stages[stage.ID]; exists {
		return fmt.Errorf("stage %q is already registered", stage.ID)
	}

	registry.stages[stage.ID] = stage
	return nil
}

func (registry *StageRegistry) Get(id string) (Stage, bool) {
	if registry == nil {
		return Stage{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	stage, exists := registry.stages[id]
	return stage, exists
}

func (registry *StageRegistry) List() []Stage {
	if registry == nil {
		return nil
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	stages := make([]Stage, 0, len(registry.stages))
	for _, stage := range registry.stages {
		stages = append(stages, stage)
	}
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].ID < stages[j].ID
	})

	return stages
}
