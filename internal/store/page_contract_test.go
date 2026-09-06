package store

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	if !strings.Contains(pageSelect, ",p.status") {
		t.Fatal("common page SELECT omits status")
	}
	for _, status := range []string{"draft", "verified", "deprecated", "archived"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			page, err := scanPage(pageContractRow{status: status})
			if err != nil {
				t.Fatal(err)
			}
			if page.Status != status {
				t.Fatalf("status = %q, want %q", page.Status, status)
			}
		})
	}
}

type propertyContractTx struct {
	pgx.Tx
	inserted [][]any
}

func (tx *propertyContractTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "INSERT INTO page_properties") {
		tx.inserted = append(tx.inserted, append([]any(nil), args...))
	}
	return pgconn.CommandTag{}, nil
}

func TestPagePropertyKeyNormalizationPreservesValue(t *testing.T) {
	t.Parallel()
	tx := &propertyContractTx{}
	properties := map[string]string{" Owner ": " Platform ", "environment": " prod ", "blank": " ", " ": "ignored"}
	if err := replacePageProperties(context.Background(), tx, 7, properties); err != nil {
		t.Fatal(err)
	}
	want := [][]any{{int64(7), "Owner", "Platform"}, {int64(7), "environment", "prod"}}
	if !reflect.DeepEqual(tx.inserted, want) {
		t.Fatalf("properties = %#v, want %#v", tx.inserted, want)
	}
	if properties[" Owner "] != " Platform " {
		t.Fatal("normalization mutated the caller's properties")
	}
}

// This test exercises the actual SQL projections in an isolated PostgreSQL schema.
func TestPageLifecycleQueryContracts(t *testing.T) {
	dsn := integrationDatabase(t)
	ctx := context.Background()
	database, err := Open(ctx, dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	actor, err := database.EnsureAdministrator(ctx, "contract-admin", "", "Contract Admin")
	if err != nil {
		t.Fatal(err)
	}
	statuses := []string{"draft", "verified", "deprecated", "archived"}
	for _, status := range statuses {
		slug := "contract/" + status
		_, err := database.SavePage(ctx, "", slug, "Contract "+status, "", "", "Lifecycle contract", "Created", nil, nil, nil,
			domain.PageMetadata{Status: status}, map[string]string{" Owner ": " Platform "}, actor)
		if err != nil {
			t.Fatal(err)
		}
		if err := database.SetFavorite(ctx, slug, actor.ID, true); err != nil {
			t.Fatal(err)
		}
		if err := database.RecordView(ctx, slug, actor.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		name string
		load func() ([]domain.Page, error)
	}{
		{"list", func() ([]domain.Page, error) { return database.ListPages(ctx, 100) }},
		{"search", func() ([]domain.Page, error) { return database.Search(ctx, "Contract", 100) }},
		{"favorites", func() ([]domain.Page, error) { return database.Favorites(ctx, actor.ID) }},
		{"recently viewed", func() ([]domain.Page, error) { return database.RecentViewed(ctx, actor.ID, 100) }},
		{"popular", func() ([]domain.Page, error) { return database.Popular(ctx, 100) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pages, err := tt.load()
			if err != nil {
				t.Fatal(err)
			}
			if len(pages) != len(statuses) {
				t.Fatalf("got %d pages, want %d", len(pages), len(statuses))
			}
			for _, page := range pages {
				if page.Status != strings.TrimPrefix(page.Slug, "contract/") {
					t.Fatalf("%s status = %q", page.Slug, page.Status)
				}
			}
		})
	}
	page, err := database.GetPage(ctx, "contract/verified")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.PageProperty{{Key: "Owner", Value: "Platform"}}
	if !reflect.DeepEqual(page.Properties, want) {
		t.Fatalf("properties = %#v, want %#v", page.Properties, want)
	}
}
