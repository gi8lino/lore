package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/service"
	"github.com/stretchr/testify/assert"
)

type moveErrorStub struct{ err error }

func (s moveErrorStub) Move(context.Context, string, string, domain.MovePageOptions, domain.User) error {
	return s.err
}

func TestMovePageFormValidationProblem(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	request := httptest.NewRequest(http.MethodPost, "/pages/source/move", strings.NewReader("slug="))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("slug", "source")
	response := httptest.NewRecorder()
	err := fmt.Errorf("move: %w", &service.ValidationError{Fields: []service.FieldError{{
		Field: "slug", Message: "A destination path is required.",
	}}})

	MovePageForm(moveErrorStub{err: err}, logger)(response, request)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{"error":"Page validation failed.","problems":{"slug":"A destination path is required."}}`, response.Body.String())
	assert.Empty(t, logs.String())
}

type graphErrorStub struct{ err error }

func (s graphErrorStub) KnowledgeGraph(context.Context, int) (domain.KnowledgeGraph, error) {
	return domain.KnowledgeGraph{}, s.err
}

func TestKnowledgeGraphFailureIsUnexpected(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	request := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	response := httptest.NewRecorder()
	err := fmt.Errorf("graph dependency: %w", domain.ErrNotFound)

	KnowledgeGraphAPI(graphErrorStub{err: err}, logger)(response, request)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(t, `{"error":"The request could not be processed.","problems":{}}`, response.Body.String())
	assert.Contains(t, logs.String(), err.Error())
}

type savedSearchErrorStub struct {
	err error
}

func (s savedSearchErrorStub) SaveSavedSearch(context.Context, int64, int64, string, string, bool) error {
	return s.err
}
func (s savedSearchErrorStub) DeleteSavedSearch(context.Context, int64, int64) error { return s.err }

type membershipErrorStub struct {
	groupWriter
	err error
}

func (s membershipErrorStub) AddGroupMember(context.Context, int64, int64) error    { return s.err }
func (s membershipErrorStub) RemoveGroupMember(context.Context, int64, int64) error { return s.err }

func TestKnownServiceErrorsReachHTTPTranslators(t *testing.T) {
	t.Parallel()
	missing := fmt.Errorf("repository: %w", domain.ErrNotFound)
	conflict := fmt.Errorf("repository: %w", domain.ErrAlreadyExists)
	tests := []struct {
		name        string
		handler     func(*slog.Logger) http.HandlerFunc
		body        string
		contentType string
		status      int
		message     string
	}{
		{"create search conflict", func(l *slog.Logger) http.HandlerFunc {
			return CreateSavedSearch(savedSearchErrorStub{err: conflict}, l)
		}, "name=test&query=test", "application/x-www-form-urlencoded", http.StatusConflict, "Saved search already exists."},
		{"delete missing search", func(l *slog.Logger) http.HandlerFunc { return DeleteSavedSearch(savedSearchErrorStub{err: missing}, l) }, "", "", http.StatusNotFound, "Saved search not found."},
		{"add missing membership", func(l *slog.Logger) http.HandlerFunc {
			return AddAdminGroupMember(membershipErrorStub{err: missing}, nil, l)
		}, `{"user_id":1}`, "application/json", http.StatusNotFound, "Group or user not found."},
		{"remove missing membership", func(l *slog.Logger) http.HandlerFunc {
			return RemoveAdminGroupMember(membershipErrorStub{err: missing}, l)
		}, "", "", http.StatusNotFound, "Group membership not found."},
		{"move path conflict", func(l *slog.Logger) http.HandlerFunc { return MovePageForm(moveErrorStub{err: conflict}, l) }, "slug=target", "application/x-www-form-urlencoded", http.StatusConflict, "Page path already exists."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			request.SetPathValue("id", "1")
			request.SetPathValue("userID", "1")
			request.SetPathValue("slug", "source")
			request = auth.WithUser(request, domain.User{ID: 1, Role: "admin"})
			response := httptest.NewRecorder()
			test.handler(logger)(response, request)
			assert.Equal(t, test.status, response.Code)
			assert.Contains(t, response.Body.String(), test.message)
			assert.Empty(t, logs.String())
		})
	}
}

type aliasFailureStub struct {
	pageViewCatalogService
	err error
}

func (s aliasFailureStub) GetPage(context.Context, string) (domain.Page, error) {
	return domain.Page{}, domain.ErrNotFound
}
func (s aliasFailureStub) ResolvePageAlias(context.Context, string) (string, error) { return "", s.err }

func TestAliasFailureIsNotDiscarded(t *testing.T) {
	t.Parallel()
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprint(api), func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			repository := aliasFailureStub{err: fmt.Errorf("alias database offline")}
			request := httptest.NewRequest(http.MethodGet, "/pages/missing", nil)
			request.SetPathValue("slug", "missing")
			response := httptest.NewRecorder()
			if api {
				GetPage(repository, logger)(response, request)
			} else {
				ViewPage(nil, repository, nil, nil, nil, &Views{logger: logger})(response, request)
			}
			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assert.Contains(t, logs.String(), "alias database offline")
			assert.NotContains(t, response.Body.String(), "alias database offline")
		})
	}
}
