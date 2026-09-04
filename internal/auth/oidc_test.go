package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOIDCAuthenticateRejectsUnboundSessions(t *testing.T) {
	t.Parallel()

	authenticator := &OIDC{
		secret: []byte("0123456789abcdef0123456789abcdef"),
		issuer: "https://identity.example.com/realms/lore",
	}

	t.Run("rejects session from another issuer", func(t *testing.T) {
		t.Parallel()

		request := oidcSessionRequest(t, authenticator, session{
			Issuer:  "https://identity.example.com/realms/other",
			Subject: "user-123",
			Expires: time.Now().Add(time.Hour).Unix(),
		})

		_, err := authenticator.Authenticate(request)
		assert.ErrorIs(t, err, ErrUnauthenticated)
	})

	t.Run("rejects session without subject", func(t *testing.T) {
		t.Parallel()

		request := oidcSessionRequest(t, authenticator, session{
			Issuer:  authenticator.issuer,
			Expires: time.Now().Add(time.Hour).Unix(),
		})

		_, err := authenticator.Authenticate(request)
		assert.ErrorIs(t, err, ErrUnauthenticated)
	})

	t.Run("rejects expired session", func(t *testing.T) {
		t.Parallel()

		request := oidcSessionRequest(t, authenticator, session{
			Issuer:  authenticator.issuer,
			Subject: "user-123",
			Expires: time.Now().Add(-time.Hour).Unix(),
		})

		_, err := authenticator.Authenticate(request)
		assert.ErrorIs(t, err, ErrUnauthenticated)
	})

	t.Run("rejects legacy profile session", func(t *testing.T) {
		t.Parallel()

		request := oidcSessionRequest(t, authenticator, struct {
			Username string `json:"u"`
			Email    string `json:"e"`
			Name     string `json:"n"`
			Expires  int64  `json:"x"`
		}{
			Username: "admin",
			Email:    "admin@example.com",
			Name:     "Administrator",
			Expires:  time.Now().Add(time.Hour).Unix(),
		})

		_, err := authenticator.Authenticate(request)
		assert.ErrorIs(t, err, ErrUnauthenticated)
	})
}

// oidcSessionRequest creates a request carrying one signed Lore session cookie.
func oidcSessionRequest(t *testing.T, authenticator *OIDC, value any) *http.Request {
	t.Helper()

	response := httptest.NewRecorder()
	authenticator.setCookie(response, "lore_session", value, 3600)
	cookies := response.Result().Cookies()
	if !assert.Len(t, cookies, 1) {
		return httptest.NewRequest("GET", "/", nil)
	}

	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(cookies[0])
	return request
}

func TestOIDCGroupValues(t *testing.T) {
	t.Parallel()

	t.Run("accepts string array", func(t *testing.T) {
		t.Parallel()

		groups, err := oidcGroupValues([]byte(`["/admins"," /family ","/admins"]`))
		assert.NoError(t, err)
		assert.Equal(t, []string{"/admins", "/family"}, groups)
	})

	t.Run("accepts one string", func(t *testing.T) {
		t.Parallel()

		groups, err := oidcGroupValues([]byte(`"/admins"`))
		assert.NoError(t, err)
		assert.Equal(t, []string{"/admins"}, groups)
	})

	t.Run("accepts missing claim", func(t *testing.T) {
		t.Parallel()

		groups, err := oidcGroupValues(nil)
		assert.NoError(t, err)
		assert.Empty(t, groups)
	})

	t.Run("rejects non-string values", func(t *testing.T) {
		t.Parallel()

		_, err := oidcGroupValues([]byte(`{"name":"admins"}`))
		assert.Error(t, err)
	})
}
