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

	tests := []struct {
		name  string
		input PageSaveInput
		field string
	}{
		{
			name:  "slug is not derived from title",
			input: PageSaveInput{Title: "Explicit title", Status: "verified"},
			field: "slug",
		},
		{
			name:  "status has no default",
			input: PageSaveInput{Slug: "explicit-path", Title: "Explicit title"},
			field: "status",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewPages(nil).Save(context.Background(), test.input)
			validation, ok := errors.AsType[*ValidationError](err)

			require.True(t, ok)
			require.Len(t, validation.Fields, 1)
			assert.Equal(t, test.field, validation.Fields[0].Field)
		})
	}
}

func TestBulkValidatesInputsBeforePersistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input BulkPageInput
		field string
	}{
		{name: "pages", input: BulkPageInput{}, field: "pages"},
		{
			name:  "group",
			input: BulkPageInput{Action: "group", Slugs: []string{"guide"}},
			field: "group_id",
		},
		{
			name:  "status",
			input: BulkPageInput{Action: "status", Slugs: []string{"guide"}, Status: "invalid"},
			field: "status",
		},
		{
			name:  "tag",
			input: BulkPageInput{Action: "tag", Slugs: []string{"guide"}},
			field: "tag",
		},
		{
			name:  "move target",
			input: BulkPageInput{Action: "move", Slugs: []string{"guide"}},
			field: "target",
		},
		{
			name:  "action",
			input: BulkPageInput{Action: "invalid", Slugs: []string{"guide"}},
			field: "action",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := NewPages(nil).Bulk(context.Background(), test.input)
			validation, ok := errors.AsType[*ValidationError](err)

			require.True(t, ok)
			assert.Equal(t, test.field, validation.Fields[0].Field)
		})
	}
}
