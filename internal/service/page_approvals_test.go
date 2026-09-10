package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageApprovalValidation(t *testing.T) {
	t.Parallel()

	t.Run("request requires page path", func(t *testing.T) {
		t.Parallel()
		_, err := NewPages(nil, slog.Default()).RequestReview(context.Background(), " ", "", domain.User{Role: "editor"})
		validation, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Equal(t, "slug", validation.Fields[0].Field)
	})

	t.Run("viewer cannot request review", func(t *testing.T) {
		t.Parallel()
		_, err := NewPages(nil, slog.Default()).RequestReview(context.Background(), "guide", "", domain.User{Role: "viewer"})
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("decision validates value before persistence", func(t *testing.T) {
		t.Parallel()
		err := NewPages(nil, slog.Default()).DecideReview(context.Background(), 1, "guide", "maybe", "", domain.User{Role: "admin"})
		validation, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Equal(t, "decision", validation.Fields[0].Field)
	})

	t.Run("administrator can review without owner lookup", func(t *testing.T) {
		t.Parallel()
		allowed, err := NewPages(nil, slog.Default()).CanReview(context.Background(), "guide", domain.User{Role: "admin"})
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}
