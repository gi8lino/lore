package middleware

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authenticatorStub is a deterministic authenticator used to verify middleware fallback behavior.
type authenticatorStub struct {
	// user is returned when err is nil.
	user store.User
	// err is the authentication result returned to the middleware.
	err error
	// calls counts Authenticate invocations.
	calls *int
}

// Authenticate returns the configured stub result.
func (a authenticatorStub) Authenticate(*http.Request) (store.User, error) {
	if a.calls != nil {
		*a.calls++
	}
	return a.user, a.err
}

// TestAuthenticateAPIStopsOnInvalidCredentials verifies that explicit bad credentials never fall through to browser
// auth.
func TestAuthenticateAPIStopsOnInvalidCredentials(t *testing.T) {
	t.Parallel()

	firstCalls := 0
	fallbackCalls := 0
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := AuthenticateAPI(
		logger,
		authenticatorStub{err: auth.ErrInvalidCredentials, calls: &firstCalls},
		authenticatorStub{user: store.User{ID: 1, Role: "admin"}, calls: &fallbackCalls},
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/search", nil))

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Equal(t, 1, firstCalls)
	assert.Zero(t, fallbackCalls)
	assert.JSONEq(
		t,
		`{"error":"Unauthorized.","problems":{"authorization":"Authorization header must contain a valid Bearer token."}}`,
		response.Body.String(),
	)
}

// TestAuthenticateAPIContinuesWhenCredentialsAreAbsent verifies that missing credentials may fall through to browser
// auth.
func TestAuthenticateAPIContinuesWhenCredentialsAreAbsent(t *testing.T) {
	t.Parallel()

	fallbackCalls := 0
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := AuthenticateAPI(
		logger,
		authenticatorStub{err: auth.ErrUnauthenticated},
		authenticatorStub{user: store.User{ID: 1, Role: "admin"}, calls: &fallbackCalls},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		require.True(t, ok)
		assert.Equal(t, int64(1), user.ID)
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/search", nil))

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, 1, fallbackCalls)
}

// TestAuthenticateAPIReturnsServerErrorForUnexpectedAuthFailure verifies unexpected authenticator failures remain
// internal errors.
func TestAuthenticateAPIReturnsServerErrorForUnexpectedAuthFailure(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := AuthenticateAPI(
		logger,
		authenticatorStub{err: errors.New("boom")},
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not run")
		}),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/search", nil))
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.JSONEq(
		t,
		`{"error":"The request could not be processed.","problems":null}`,
		response.Body.String(),
	)
}

// TestAuthenticateAPILogsDeniedRequests verifies authentication failures are visible without exposing credentials.
func TestAuthenticateAPILogsDeniedRequests(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := AuthenticateAPI(
		logger,
		authenticatorStub{err: auth.ErrInvalidCredentials},
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not run")
		}),
	)

	request := httptest.NewRequest("GET", "/api/search?q=postgres", nil)
	request.Header.Set("Authorization", "Bearer super-secret-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Contains(t, logs.String(), "authentication_denied")
	assert.Contains(t, logs.String(), "invalid_credentials")
	assert.Contains(t, logs.String(), "/api/search")
	assert.NotContains(t, logs.String(), "super-secret-token")
}

// TestAuthenticateAPILogsMissingCredentials verifies the final unauthenticated outcome is logged by unauthorized.
func TestAuthenticateAPILogsMissingCredentials(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := AuthenticateAPI(
		logger,
		authenticatorStub{err: auth.ErrUnauthenticated},
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not run")
		}),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/search", nil))

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Contains(t, logs.String(), "authentication_denied")
	assert.Contains(t, logs.String(), "credentials_required")
	assert.JSONEq(
		t,
		`{"error":"Unauthorized.","problems":{"authorization":"Authentication credentials are required."}}`,
		response.Body.String(),
	)
}

func TestBrowserUnauthorizedReturnsToReadablePage(t *testing.T) {
	for _, tc := range []struct{ method, path, next string }{
		{"POST", "/admin/oidc/pending/42/reopen", "/admin/users"},
		{"POST", "/admin/users/7", "/admin/users"},
		{"POST", "/admin/settings", "/"},
		{"GET", "/admin/users?filter=all", "/admin/users?filter=all"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			browserUnauthorized(response, httptest.NewRequest(tc.method, tc.path, nil), unauthorizedCredentialsRequired)
			require.Equal(t, http.StatusFound, response.Code)
			location, err := url.Parse(response.Header().Get("Location"))
			require.NoError(t, err)
			assert.Equal(t, tc.next, location.Query().Get("next"))
		})
	}
}
