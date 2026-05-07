package salesgpt

import (
	"fmt"
	"sort"
	"sync"
)

type Interest struct {
	ID          string
	Description string
}

type InterestRegistry struct {
	mu        sync.RWMutex
	interests map[string]Interest
}

func NewInterestRegistry(interests ...Interest) (*InterestRegistry, error) {
	registry := &InterestRegistry{
		interests: make(map[string]Interest),
	}

	for _, interest := range interests {
		if err := registry.Register(interest); err != nil {
			return registry, err
		}
	}

	return registry, nil
}

func (registry *InterestRegistry) Register(interest Interest) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if interest.ID == "" {
		return fmt.Errorf("interest id is required")
	}
	if interest.Description == "" {
		return fmt.Errorf("interest %q description is required", interest.ID)
	}
	if _, exists := registry.interests[interest.ID]; exists {
		return fmt.Errorf("interest %q is already registered", interest.ID)
	}

	registry.interests[interest.ID] = interest
	return nil
}

func (registry *InterestRegistry) Get(id string) (Interest, bool) {
	if registry == nil {
		return Interest{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	interest, exists := registry.interests[id]
	return interest, exists
}

func (registry *InterestRegistry) List() []Interest {
	if registry == nil {
		return nil
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	interests := make([]Interest, 0, len(registry.interests))
	for _, interest := range registry.interests {
		interests = append(interests, interest)
	}
	sort.Slice(interests, func(i, j int) bool {
		return interests[i].ID < interests[j].ID
	})

	return interests
}
