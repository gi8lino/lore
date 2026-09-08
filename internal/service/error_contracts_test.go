package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownInputFailuresAreValidationErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name  string
		run   func() error
		field string
	}{
		{"saved search name", func() error { return NewKnowledge(nil).SaveSavedSearch(ctx, 1, 0, " ", "query", false) }, "name"},
		{"saved search query", func() error { return NewKnowledge(nil).SaveSavedSearch(ctx, 1, 0, "name", " ", false) }, "query"},
		{"snippet kind", func() error {
			_, err := NewKnowledge(nil).SaveKnowledgeSnippet(ctx, 0, 1, "invalid", "name", "", "")
			return err
		}, "kind"},
		{"snippet name", func() error {
			_, err := NewKnowledge(nil).SaveKnowledgeSnippet(ctx, 0, 1, "snippet", " ", "", "")
			return err
		}, "name"},
		{"group name", func() error { _, err := NewGroups(nil).CreateGroup(ctx, " "); return err }, "name"},
		{"create template name", func() error { _, err := NewTemplates(nil).CreatePageTemplate(ctx, " ", "", ""); return err }, "name"},
		{"update template name", func() error { return NewTemplates(nil).UpdatePageTemplate(ctx, 1, " ", "", "") }, "name"},
		{"same move path", func() error {
			return NewPages(nil).Move(ctx, "/guide/", "guide", domain.MovePageOptions{}, domain.User{})
		}, "slug"},
		{"move tree into itself", func() error {
			return NewPages(nil).Move(ctx, "guide", "guide/child", domain.MovePageOptions{MoveChildren: true}, domain.User{})
		}, "slug"},
		{"bulk move same path", func() error {
			return NewPages(nil).Bulk(ctx, BulkPageInput{Action: "move", Slugs: []string{"guide/child"}, Target: "guide"})
		}, "target"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// Nil repositories ensure invalid input is rejected before persistence.
			validation, ok := errors.AsType[*ValidationError](test.run())
			require.True(t, ok)
			require.Len(t, validation.Fields, 1)
			assert.Equal(t, test.field, validation.Fields[0].Field)
		})
	}
}

func TestAdditionalServiceValidationBeforePersistence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name  string
		run   func() error
		field string
	}{
		{"comment body", func() error { return NewPages(nil).AddComment(ctx, "page", "", " ", domain.User{}) }, "body"},
		{"user role", func() error { return NewUsers(nil).UpdateUser(ctx, 1, "invalid", true, nil, nil) }, "role"},
		{"token name", func() error { _, err := NewTokens(nil).CreateToken(ctx, " ", 1, 1, nil); return err }, "name"},
		{"sidebar width", func() error { return NewPreferences(nil).SetSidebarWidth(ctx, 1, -1) }, "sidebar_width"},
		{"preference density", func() error {
			return NewPreferences(nil).SavePreferences(ctx, 1, domain.UserPreferences{NavigationDensity: "invalid"})
		}, "navigation_density"},
		{"navigation icon", func() error { return NewNavigation(nil).SetNavigationIcon(ctx, "page", "not-an-icon") }, "icon"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			validation, ok := errors.AsType[*domain.ValidationError](test.run())
			require.True(t, ok)
			require.Len(t, validation.Fields, 1)
			assert.Equal(t, test.field, validation.Fields[0].Field)
		})
	}
}
