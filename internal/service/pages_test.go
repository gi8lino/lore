package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pageSaveRepositoryStub struct {
	pageRepository
	slug string
}

func (r *pageSaveRepositoryStub) SavePage(
	_ context.Context,
	_, slug, title, _, _, _, _ string,
	_, _ []string,
	_ []int64,
	_ domain.PageMetadata,
	_ map[string]string,
	_ domain.User,
) (domain.Page, error) {
	r.slug = slug

	return domain.Page{Slug: slug, Title: title}, nil
}

func TestSaveValidatesPageBeforePersistence(t *testing.T) {
	t.Parallel()

	pages := NewPages(nil, slog.Default())
	_, err := pages.Save(context.Background(), PageSaveInput{
		Icon:               "not-an-icon",
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
		{Field: "icon", Message: "Choose an icon from the available icon catalog."},
		{Field: "language", Message: "Choose a supported content language."},
		{Field: "status", Message: "Choose valid page workflow settings."},
	}, validation.Fields)
}

func TestMoveValidatesDestinationBeforePersistence(t *testing.T) {
	t.Parallel()

	pages := NewPages(nil, slog.Default())
	err := pages.Move(context.Background(), "guide", "", domain.MovePageOptions{}, domain.User{})

	validation, ok := errors.AsType[*ValidationError](err)

	require.True(t, ok)
	assert.Equal(t, "slug", validation.Fields[0].Field)
}

func TestSaveSlugResolution(t *testing.T) {
	t.Parallel()

	t.Run("derives path from title for a new page", func(t *testing.T) {
		t.Parallel()

		repository := &pageSaveRepositoryStub{}
		_, err := NewPages(repository, slog.Default()).save(context.Background(), PageSaveInput{
			Title:  "Generated Page Path",
			Status: "verified",
		})

		require.NoError(t, err)
		assert.Equal(t, "generated-page-path", repository.slug)
	})

	t.Run("keeps an explicit path for a new page", func(t *testing.T) {
		t.Parallel()

		repository := &pageSaveRepositoryStub{}
		_, err := NewPages(repository, slog.Default()).save(context.Background(), PageSaveInput{
			Slug:   "custom/path",
			Title:  "Generated Page Path",
			Status: "verified",
		})

		require.NoError(t, err)
		assert.Equal(t, "custom/path", repository.slug)
	})

	t.Run("rejects a path made only of slashes", func(t *testing.T) {
		t.Parallel()

		_, err := NewPages(nil, slog.Default()).Save(context.Background(), PageSaveInput{
			Slug:   "/////",
			Title:  "Invalid path",
			Status: "verified",
		})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		require.Len(t, validation.Fields, 1)
		assert.Equal(t, "slug", validation.Fields[0].Field)
		assert.Equal(t, "Use a page path without leading, trailing, or repeated slashes.", validation.Fields[0].Message)
	})

	t.Run("rejects repeated slashes inside a path", func(t *testing.T) {
		t.Parallel()

		_, err := NewPages(nil, slog.Default()).Save(context.Background(), PageSaveInput{
			Slug:   "platform//database",
			Title:  "Invalid path",
			Status: "verified",
		})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		require.Len(t, validation.Fields, 1)
		assert.Equal(t, "slug", validation.Fields[0].Field)
	})

	t.Run("requires an explicit path when editing an existing page", func(t *testing.T) {
		t.Parallel()

		_, err := NewPages(nil, slog.Default()).Save(context.Background(), PageSaveInput{
			PreviousSlug: "existing-page",
			Title:        "Renamed title",
			Status:       "verified",
		})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		require.Len(t, validation.Fields, 1)
		assert.Equal(t, "slug", validation.Fields[0].Field)
	})
}

func TestSaveRequiresExplicitStatus(t *testing.T) {
	t.Parallel()

	_, err := NewPages(nil, slog.Default()).Save(context.Background(), PageSaveInput{Slug: "explicit-path", Title: "Explicit title"})
	validation, ok := errors.AsType[*ValidationError](err)

	require.True(t, ok)
	require.Len(t, validation.Fields, 1)
	assert.Equal(t, "status", validation.Fields[0].Field)
}

func TestBulkValidatesInputsBeforePersistence(t *testing.T) {
	t.Parallel()

	t.Run("pages", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "pages", validation.Fields[0].Field)
	})

	t.Run("group", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{Action: "group", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "group_id", validation.Fields[0].Field)
	})

	t.Run("status", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{Action: "status", Slugs: []string{"guide"}, Status: "invalid"})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "status", validation.Fields[0].Field)
	})

	t.Run("tag", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{Action: "tag", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "tag", validation.Fields[0].Field)
	})

	t.Run("move target", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{Action: "move", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "target", validation.Fields[0].Field)
	})

	t.Run("action", func(t *testing.T) {
		t.Parallel()

		err := NewPages(nil, slog.Default()).Bulk(context.Background(), BulkPageInput{Action: "invalid", Slugs: []string{"guide"}})
		validation, ok := errors.AsType[*ValidationError](err)

		require.True(t, ok)
		assert.Equal(t, "action", validation.Fields[0].Field)
	})
}
