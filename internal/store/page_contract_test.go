package store

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A narrow row fake checks the projection order without a running database.
type pageContractRow struct{ status string }

func (r pageContractRow) Scan(destinations ...any) error {
	if len(destinations) != 13 {
		return fmt.Errorf("page projection has %d fields, want 13", len(destinations))
	}
	status, ok := destinations[12].(*string)
	if !ok {
		return fmt.Errorf("page status destination is %T, want *string", destinations[12])
	}
	*status = r.status
	return nil
}

func TestScanPagePreservesLifecycleStatus(t *testing.T) {
	t.Parallel()
	require.Contains(t, pageSelect, ",p.status", "common page SELECT must include status")

	t.Run("draft", func(t *testing.T) {
		t.Parallel()

		page, err := scanPage(pageContractRow{status: "draft"})
		require.NoError(t, err)
		assert.Equal(t, "draft", page.Status)
	})

	t.Run("verified", func(t *testing.T) {
		t.Parallel()

		page, err := scanPage(pageContractRow{status: "verified"})
		require.NoError(t, err)
		assert.Equal(t, "verified", page.Status)
	})

	t.Run("deprecated", func(t *testing.T) {
		t.Parallel()

		page, err := scanPage(pageContractRow{status: "deprecated"})
		require.NoError(t, err)
		assert.Equal(t, "deprecated", page.Status)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()

		page, err := scanPage(pageContractRow{status: "archived"})
		require.NoError(t, err)
		assert.Equal(t, "archived", page.Status)
	})
}

type propertyContractTx struct {
	pgx.Tx
	inserted [][]any
}

func (tx *propertyContractTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "INSERT INTO page_properties") {
		tx.inserted = append(tx.inserted, slices.Clone(args))
	}
	return pgconn.CommandTag{}, nil
}

func TestPagePropertyKeyNormalizationPreservesValue(t *testing.T) {
	t.Parallel()
	tx := &propertyContractTx{}
	properties := map[string]string{" Owner ": " Platform ", "environment": " prod ", "blank": " ", " ": "ignored"}

	require.NoError(t, replacePageProperties(context.Background(), tx, 7, properties))
	assert.Equal(t, [][]any{{int64(7), "Owner", "Platform"}, {int64(7), "environment", "prod"}}, tx.inserted)
	assert.Equal(t, " Platform ", properties[" Owner "], "normalization must not mutate the caller's properties")
}

// This test exercises the actual SQL projections in an isolated PostgreSQL schema.
func TestPageLifecycleQueryContracts(t *testing.T) {
	dsn := integrationDatabase(t)
	ctx := context.Background()
	database, err := Open(ctx, dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	defer database.Close()
	actor, err := database.EnsureAdministrator(ctx, "contract-admin", "", "Contract Admin")
	require.NoError(t, err)
	statuses := []string{"draft", "verified", "deprecated", "archived"}
	for _, status := range statuses {
		slug := "contract/" + status
		_, err := database.SavePage(ctx, "", slug, "Contract "+status, "", "", "Lifecycle contract", "Created", nil, nil, nil,
			domain.PageMetadata{Status: status}, map[string]string{" Owner ": " Platform "}, actor)
		require.NoError(t, err)
		require.NoError(t, database.SetFavorite(ctx, slug, actor.ID, true))
		require.NoError(t, database.RecordView(ctx, slug, actor.ID))
	}
	t.Run("list", func(t *testing.T) {
		pages, err := database.ListPages(ctx, 100)
		require.NoError(t, err)
		require.Len(t, pages, len(statuses))
		for _, page := range pages {
			assert.Equal(t, strings.TrimPrefix(page.Slug, "contract/"), page.Status, "page %s", page.Slug)
		}
	})

	t.Run("search", func(t *testing.T) {
		pages, err := database.Search(ctx, "Contract", 100)
		require.NoError(t, err)
		require.Len(t, pages, len(statuses))
		for _, page := range pages {
			assert.Equal(t, strings.TrimPrefix(page.Slug, "contract/"), page.Status, "page %s", page.Slug)
		}
	})

	t.Run("favorites", func(t *testing.T) {
		pages, err := database.Favorites(ctx, actor.ID)
		require.NoError(t, err)
		require.Len(t, pages, len(statuses))
		for _, page := range pages {
			assert.Equal(t, strings.TrimPrefix(page.Slug, "contract/"), page.Status, "page %s", page.Slug)
		}
	})

	t.Run("recently viewed", func(t *testing.T) {
		pages, err := database.RecentViewed(ctx, actor.ID, 100)
		require.NoError(t, err)
		require.Len(t, pages, len(statuses))
		for _, page := range pages {
			assert.Equal(t, strings.TrimPrefix(page.Slug, "contract/"), page.Status, "page %s", page.Slug)
		}
	})

	t.Run("popular", func(t *testing.T) {
		pages, err := database.Popular(ctx, 100)
		require.NoError(t, err)
		require.Len(t, pages, len(statuses))
		for _, page := range pages {
			assert.Equal(t, strings.TrimPrefix(page.Slug, "contract/"), page.Status, "page %s", page.Slug)
		}
	})

	t.Run("page properties", func(t *testing.T) {
		page, err := database.GetPage(ctx, "contract/verified")
		require.NoError(t, err)
		assert.Equal(t, []domain.PageProperty{{Key: "Owner", Value: "Platform"}}, page.Properties)
	})
}
