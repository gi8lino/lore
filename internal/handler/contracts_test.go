package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
)

// Unused capabilities remain embedded; an unexpected call fails the test.
type emptyContractServices struct {
	navigationService
	knowledgeSnippetReader
	groupReader
	userDirectoryService
	imageService
	attachmentService
}

func (emptyContractServices) ListPages(context.Context, int) ([]domain.Page, error) { return nil, nil }
func (emptyContractServices) Search(context.Context, string, int) ([]domain.Page, error) {
	return nil, nil
}
func (emptyContractServices) Tags(context.Context) ([]string, error) { return nil, nil }
func (emptyContractServices) AssignableGroups(context.Context, domain.User) ([]domain.Group, error) {
	return nil, nil
}
func (emptyContractServices) GroupMembers(context.Context, int64) ([]domain.User, error) {
	return nil, nil
}
func (emptyContractServices) SearchUsers(context.Context, string, int) ([]domain.User, error) {
	return nil, nil
}
func (emptyContractServices) NavigationPages(context.Context) ([]domain.Page, error) { return nil, nil }
func (emptyContractServices) KnowledgeSnippets(context.Context) ([]domain.KnowledgeSnippet, error) {
	return nil, nil
}
func (emptyContractServices) PageAliases(context.Context) (map[string]string, error) { return nil, nil }
func (emptyContractServices) KnowledgeGraph(context.Context, int) (domain.KnowledgeGraph, error) {
	return domain.KnowledgeGraph{}, nil
}
func (emptyContractServices) Notifications(context.Context, int64, int) ([]domain.Notification, int, error) {
	return nil, 0, nil
}
func (emptyContractServices) MarkNotificationRead(context.Context, int64, int64) error { return nil }
func (emptyContractServices) Images(context.Context) ([]domain.Image, error)           { return nil, nil }
func (emptyContractServices) Attachments(context.Context) ([]domain.Attachment, error) {
	return nil, nil
}

// Exercise real HTTP serialization, not just zero-value domain marshaling.
func TestEmptyAPICollectionContracts(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../test/contracts/http.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	services := emptyContractServices{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range []struct {
		name, fixture string
		handler       http.HandlerFunc
	}{
		{"pages", "empty_collection", ListPages(services, logger)},
		{"search", "empty_collection", SearchAPI(services, logger)},
		{"recent", "empty_collection", Recent(services, logger)},
		{"tags", "empty_collection", Tags(services, logger)},
		{"groups", "empty_collection", GroupsAPI(services, logger)},
		{"members", "empty_collection", AdminGroupMembers(services, logger)},
		{"users", "empty_collection", SearchAdminUsers(services, logger)},
		{"images", "empty_collection", ListImages(services, logger)},
		{"attachments", "empty_collection", ListAttachments(services, logger)},
		{"graph", "empty_graph", KnowledgeGraphAPI(services, logger)},
		{"catalog", "empty_catalog", EditorCatalog(services, services, services, logger)},
		{"notifications", "empty_notifications", NotificationsAPI(services, logger)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request := auth.WithUser(httptest.NewRequest(http.MethodGet, "/?q=nobody", nil), domain.User{ID: 1, Role: "admin", Enabled: true})
			request.SetPathValue("id", "1")
			response := httptest.NewRecorder()
			tt.handler(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
			var got, want any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(fixtures[tt.fixture], &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("response = %#v, want %#v", got, want)
			}
		})
	}
}
