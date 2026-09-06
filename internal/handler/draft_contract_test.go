package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/service"
)

type draftContractService struct {
	editorDraftService
	draft domain.PageDraft
}

func (s draftContractService) Draft(context.Context, int64, string) (domain.PageDraft, error) {
	return s.draft, nil
}
func (s draftContractService) Save(context.Context, service.PageDraftSaveInput) (domain.PageDraft, error) {
	return s.draft, nil
}

func TestDraftResponseContract(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../test/contracts/draft.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected any
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	drafts := draftContractService{draft: domain.PageDraft{ID: 1, Key: "new", CreatedAt: at, UpdatedAt: at}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, method := range []string{"GET", "PUT"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			request := auth.WithUser(httptest.NewRequest(method, "/api/drafts/new", strings.NewReader(`{"values":{}}`)), domain.User{ID: 7})
			request.SetPathValue("key", "new")
			handler := GetPageDraft(drafts, logger)
			if method == "PUT" {
				handler = SavePageDraft(drafts, logger)
			}
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != 200 {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
			var got any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("response = %#v, want %#v", got, expected)
			}
		})
	}
}

func TestDraftResponseNormalizesEmptySelectionsWithoutMutatingInput(t *testing.T) {
	t.Parallel()
	draft := domain.PageDraft{Values: map[string][]string{"group_id": nil, "title": {"Example"}}}
	got := draftResponse(draft)
	if got.Values["group_id"] == nil {
		t.Fatal("empty selection must be an array")
	}
	if draft.Values["group_id"] != nil {
		t.Fatal("response mapping mutated input")
	}
	if got.Values["title"][0] != "Example" {
		t.Fatal("response mapping lost a field")
	}
}
