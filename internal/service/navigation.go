package service

import (
	"context"

	"github.com/gi8lino/lore/internal/domain"
)

// navigationRepository contains navigation tree and icon operations.
type navigationRepository interface {
	NavigationPages(context.Context) ([]domain.Page, error)
	NavigationItems(context.Context) ([]domain.NavigationItem, error)
	NavigationIcons(context.Context) (map[string]string, error)
	SetNavigationIcon(context.Context, string, string) error
}

// Navigation exposes navigation tree and icon use cases.
type Navigation struct{ repository navigationRepository }

// NewNavigation constructs the navigation service.
func NewNavigation(repository navigationRepository) *Navigation {
	return &Navigation{repository: repository}
}

// NavigationPages returns the page projection needed to build navigation.
func (s *Navigation) NavigationPages(ctx context.Context) ([]domain.Page, error) {
	return s.repository.NavigationPages(ctx)
}

// NavigationItems returns configured navigation folders and metadata.
func (s *Navigation) NavigationItems(ctx context.Context) ([]domain.NavigationItem, error) {
	return s.repository.NavigationItems(ctx)
}

// NavigationIcons returns configured icons keyed by navigation path.
func (s *Navigation) NavigationIcons(ctx context.Context) (map[string]string, error) {
	return s.repository.NavigationIcons(ctx)
}

// SetNavigationIcon sets or clears the icon for a navigation path.
func (s *Navigation) SetNavigationIcon(ctx context.Context, path, icon string) error {
	return s.repository.SetNavigationIcon(ctx, path, icon)
}
