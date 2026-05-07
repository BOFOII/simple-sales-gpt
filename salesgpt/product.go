package salesgpt

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Product struct {
	ID          string
	Name        string
	Description string
	Price       *float64
	ImageURL    string
}

func (product Product) VectorID() string {
	return product.ID
}

func (product Product) VectorCategory() string {
	return VectorCategoryProduct
}

func (product Product) VectorContent() string {
	parts := []string{
		product.Name,
		product.Description,
	}
	if product.Price != nil {
		parts = append(parts, fmt.Sprintf("Price: %.2f", *product.Price))
	}

	return strings.Join(parts, "\n")
}

func (product Product) VectorMetadata() map[string]any {
	metadata := map[string]any{
		"product_id": product.ID,
		"name":       product.Name,
	}
	if product.Price != nil {
		metadata["price"] = *product.Price
	}
	if product.ImageURL != "" {
		metadata["image_url"] = product.ImageURL
	}

	return metadata
}

type ProductRegistry struct {
	mu          sync.RWMutex
	products    map[string]Product
	vectorStore *VectorStoreRegistry
	vectorizer  *Vectorizer
}

func NewProductRegistry(vectorStore *VectorStoreRegistry, vectorizer *Vectorizer, products ...Product) (*ProductRegistry, error) {
	registry := &ProductRegistry{
		products:    make(map[string]Product),
		vectorStore: vectorStore,
		vectorizer:  vectorizer,
	}

	if len(products) > 0 {
		if err := registry.RegisterMany(products...); err != nil {
			return registry, err
		}
	}

	return registry, nil
}

func (registry *ProductRegistry) Register(product Product) error {
	return registry.RegisterWithContext(context.Background(), product)
}

func (registry *ProductRegistry) RegisterWithContext(ctx context.Context, product Product) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if err := registry.validateLocked(product); err != nil {
		return err
	}
	if err := registry.vectorize(ctx, product); err != nil {
		return err
	}

	registry.products[product.ID] = product
	return nil
}

func (registry *ProductRegistry) RegisterMany(products ...Product) error {
	return registry.RegisterManyWithContext(context.Background(), products...)
}

func (registry *ProductRegistry) RegisterManyWithContext(ctx context.Context, products ...Product) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	seen := make(map[string]struct{}, len(products))
	for _, product := range products {
		if err := registry.validateLocked(product); err != nil {
			return err
		}
		if _, exists := seen[product.ID]; exists {
			return fmt.Errorf("product %q is already registered in this batch", product.ID)
		}
		seen[product.ID] = struct{}{}
	}
	if err := registry.vectorizeMany(ctx, products...); err != nil {
		return err
	}

	for _, product := range products {
		registry.products[product.ID] = product
	}
	return nil
}

func (registry *ProductRegistry) Get(id string) (Product, bool) {
	if registry == nil {
		return Product{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	product, exists := registry.products[id]
	return product, exists
}

func (registry *ProductRegistry) Validate(product Product) error {
	if registry == nil {
		return fmt.Errorf("product registry is not configured")
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.validateLocked(product)
}

func (registry *ProductRegistry) validateLocked(product Product) error {
	if product.ID == "" {
		return fmt.Errorf("product id is required")
	}
	if product.Name == "" {
		return fmt.Errorf("product %q name is required", product.ID)
	}
	if _, exists := registry.products[product.ID]; exists {
		return fmt.Errorf("product %q is already registered", product.ID)
	}

	return nil
}

func (registry *ProductRegistry) vectorize(ctx context.Context, product Product) error {
	if registry.vectorStore == nil {
		return nil
	}
	if registry.vectorizer == nil {
		return fmt.Errorf("vectorizer is not configured")
	}

	_, err := registry.vectorizer.Store(ctx, registry.vectorStore, product)
	return err
}

func (registry *ProductRegistry) vectorizeMany(ctx context.Context, products ...Product) error {
	if registry.vectorStore == nil {
		return nil
	}
	if registry.vectorizer == nil {
		return fmt.Errorf("vectorizer is not configured")
	}

	items := make([]Vectorizable, 0, len(products))
	for _, product := range products {
		items = append(items, product)
	}

	_, err := registry.vectorizer.StoreMany(ctx, registry.vectorStore, VectorCategoryProduct, items...)
	return err
}
