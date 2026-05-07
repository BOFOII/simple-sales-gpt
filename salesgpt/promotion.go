package salesgpt

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Promotion struct {
	ID          string
	Name        string
	Description string
	ImageURL    string
}

func (promotion Promotion) VectorID() string {
	return promotion.ID
}

func (promotion Promotion) VectorCategory() string {
	return VectorCategoryPromotion
}

func (promotion Promotion) VectorContent() string {
	return strings.Join([]string{
		promotion.Name,
		promotion.Description,
	}, "\n")
}

func (promotion Promotion) VectorMetadata() map[string]any {
	metadata := map[string]any{
		"promotion_id": promotion.ID,
		"name":         promotion.Name,
	}
	if promotion.ImageURL != "" {
		metadata["image_url"] = promotion.ImageURL
	}

	return metadata
}

type PromotionRegistry struct {
	mu          sync.RWMutex
	promotions  map[string]Promotion
	vectorStore *VectorStoreRegistry
	vectorizer  *Vectorizer
}

func NewPromotionRegistry(vectorStore *VectorStoreRegistry, vectorizer *Vectorizer, promotions ...Promotion) (*PromotionRegistry, error) {
	registry := &PromotionRegistry{
		promotions:  make(map[string]Promotion),
		vectorStore: vectorStore,
		vectorizer:  vectorizer,
	}

	if len(promotions) > 0 {
		if err := registry.RegisterMany(promotions...); err != nil {
			return registry, err
		}
	}

	return registry, nil
}

func (registry *PromotionRegistry) Register(promotion Promotion) error {
	return registry.RegisterWithContext(context.Background(), promotion)
}

func (registry *PromotionRegistry) RegisterWithContext(ctx context.Context, promotion Promotion) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if err := registry.validateLocked(promotion); err != nil {
		return err
	}
	if err := registry.vectorize(ctx, promotion); err != nil {
		return err
	}

	registry.promotions[promotion.ID] = promotion
	return nil
}

func (registry *PromotionRegistry) RegisterMany(promotions ...Promotion) error {
	return registry.RegisterManyWithContext(context.Background(), promotions...)
}

func (registry *PromotionRegistry) RegisterManyWithContext(ctx context.Context, promotions ...Promotion) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	seen := make(map[string]struct{}, len(promotions))
	for _, promotion := range promotions {
		if err := registry.validateLocked(promotion); err != nil {
			return err
		}
		if _, exists := seen[promotion.ID]; exists {
			return fmt.Errorf("promotion %q is already registered in this batch", promotion.ID)
		}
		seen[promotion.ID] = struct{}{}
	}
	if err := registry.vectorizeMany(ctx, promotions...); err != nil {
		return err
	}

	for _, promotion := range promotions {
		registry.promotions[promotion.ID] = promotion
	}
	return nil
}

func (registry *PromotionRegistry) Get(id string) (Promotion, bool) {
	if registry == nil {
		return Promotion{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	promotion, exists := registry.promotions[id]
	return promotion, exists
}

func (registry *PromotionRegistry) Validate(promotion Promotion) error {
	if registry == nil {
		return fmt.Errorf("promotion registry is not configured")
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.validateLocked(promotion)
}

func (registry *PromotionRegistry) validateLocked(promotion Promotion) error {
	if promotion.ID == "" {
		return fmt.Errorf("promotion id is required")
	}
	if promotion.Name == "" {
		return fmt.Errorf("promotion %q name is required", promotion.ID)
	}
	if _, exists := registry.promotions[promotion.ID]; exists {
		return fmt.Errorf("promotion %q is already registered", promotion.ID)
	}

	return nil
}

func (registry *PromotionRegistry) vectorize(ctx context.Context, promotion Promotion) error {
	if registry.vectorStore == nil {
		return nil
	}
	if registry.vectorizer == nil {
		return fmt.Errorf("vectorizer is not configured")
	}

	_, err := registry.vectorizer.Store(ctx, registry.vectorStore, promotion)
	return err
}

func (registry *PromotionRegistry) vectorizeMany(ctx context.Context, promotions ...Promotion) error {
	if registry.vectorStore == nil {
		return nil
	}
	if registry.vectorizer == nil {
		return fmt.Errorf("vectorizer is not configured")
	}

	items := make([]Vectorizable, 0, len(promotions))
	for _, promotion := range promotions {
		items = append(items, promotion)
	}

	_, err := registry.vectorizer.StoreMany(ctx, registry.vectorStore, VectorCategoryPromotion, items...)
	return err
}
