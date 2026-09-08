package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveValidatesPageBeforePersistence(t *testing.T) {
	t.Parallel()

	pages := NewPages(nil)
	_, err := pages.Save(context.Background(), PageSaveInput{
		Icon:               "not-a-lucide-icon",
		Language:           "klingon",
		Status:             "unknown",
		OwnerGroupID:       -1,
		ReviewIntervalDays: 3651,
	})

	validation, ok := errors.AsType[*ValidationError](err)

	require.True(t, ok)
	assert.Equal(t, []FieldError{
		{Field: "slug", Message: "A page path is required."},
		{Field: "title", Message: "Title is required."},
		{Field: "icon", Message: "Choose an icon from the available Lucide icons."},
		{Field: "language", Message: "Choose a supported content language."},
		{Field: "status", Message: "Choose valid page workflow settings."},
	}, validation.Fields)
}

func TestMoveValidatesDestinationBeforePersistence(t *testing.T) {
	t.Parallel()

	pages := NewPages(nil)
	err := pages.Move(context.Background(), "guide", "", domain.MovePageOptions{}, domain.User{})

	validation, ok := errors.AsType[*ValidationError](err)

	require.True(t, ok)
	assert.Equal(t, "slug", validation.Fields[0].Field)
}

func TestSaveRequiresExplicitSlugAndStatus(t *testing.T) {
	t.Parallel()

	t.Run("slug is not derived from title", func(t *testing.T) {
		t.Parallel()

		_, err := NewPages(nil).Save(context.Background(), PageSaveInput{Title: "Explicit title", Status: "verified"})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		require.Len(t, validation.Fields, 1)
		assert.Equal(t, "slug", validation.Fields[0].Field)
	})

	t.Run("status has no default", func(t *testing.T) {
		t.Parallel()

		_, err := NewPages(nil).Save(context.Background(), PageSaveInput{Slug: "explicit-path", Title: "Explicit title"})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		require.Len(t, validation.Fields, 1)
		assert.Equal(t, "status", validation.Fields[0].Field)
	})
}

func TestBulkValidatesInputsBeforePersistence(t *testing.T) {
	t.Parallel()

	t.Run("pages", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "pages", validation.Fields[0].Field)
	})

	t.Run("group", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{Action: "group", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "group_id", validation.Fields[0].Field)
	})

	t.Run("status", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{Action: "status", Slugs: []string{"guide"}, Status: "invalid"})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "status", validation.Fields[0].Field)
	})

	t.Run("tag", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{Action: "tag", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "tag", validation.Fields[0].Field)
	})

	t.Run("move target", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{Action: "move", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "target", validation.Fields[0].Field)
	})

	t.Run("action", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil).Bulk(context.Background(), BulkPageInput{Action: "invalid", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "action", validation.Fields[0].Field)
	})
}
