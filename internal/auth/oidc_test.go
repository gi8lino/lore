package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gi8lino/lore/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type oidcRepositoryStub struct{ user model.User }

func (r *oidcRepositoryStub) LoginOIDCUser(context.Context, string, string, string, string, string) (model.User, error) {
	return r.user, nil
}

func (r *oidcRepositoryStub) OIDCUser(context.Context, string, string) (model.User, error) {
	return r.user, nil
}

func (*oidcRepositoryStub) SyncOIDCGroups(context.Context, int64, []string, []model.OIDCGroupMapping, bool) error {
	return nil
}

func (*oidcRepositoryStub) SetExternalAdminStatus(context.Context, int64, string, bool) error {
	return nil
}

func TestOIDCAuthenticateExternalAdministrator(t *testing.T) {
	t.Parallel()

	const issuer = "https://identity.example.com/realms/lore"
	repository := &oidcRepositoryStub{user: model.User{
		ID:             7,
		Role:           "viewer",
		Enabled:        true,
		ExternalAdmin:  true,
		SessionVersion: 4,
	}}
	authenticator := &OIDC{
		repository: repository,
		secret:     []byte("0123456789abcdef0123456789abcdef"),
		issuer:     issuer,
		adminGroup: "/lore-admins",
	}
	request := oidcSessionRequest(t, authenticator, session{
		Issuer: issuer, Subject: "user-123", Expires: time.Now().Add(time.Hour).Unix(), Version: 4,
	})

	user, err := authenticator.Authenticate(request)

	require.NoError(t, err)
	assert.Equal(t, "admin", user.Role)

	authenticator.adminGroup = ""
	user, err = authenticator.Authenticate(request)

	require.NoError(t, err)
	assert.Equal(t, "viewer", user.Role, "a removed administrator-group setting must stop stale elevation")
}

func TestOIDCAuthenticatePreservesPreVersionedSession(t *testing.T) {
	t.Parallel()

	const issuer = "https://identity.example.com/realms/lore"
	authenticator := &OIDC{
		repository: &oidcRepositoryStub{user: model.User{Enabled: true, SessionVersion: 1}},
		secret:     []byte("0123456789abcdef0123456789abcdef"),
		issuer:     issuer,
	}
	request := oidcSessionRequest(t, authenticator, struct {
		Issuer  string `json:"iss"`
		Subject string `json:"sub"`
		Expires int64  `json:"x"`
	}{
		Issuer:  issuer,
		Subject: "user-123",
		Expires: time.Now().Add(time.Hour).Unix(),
	})

	_, err := authenticator.Authenticate(request)

	require.NoError(t, err)
}

func TestOIDCAuthenticateRejectsPreVersionedSessionAfterRevocation(t *testing.T) {
	t.Parallel()

	const issuer = "https://identity.example.com/realms/lore"
	authenticator := &OIDC{
		repository: &oidcRepositoryStub{user: model.User{Enabled: true, SessionVersion: 2}},
		secret:     []byte("0123456789abcdef0123456789abcdef"),
		issuer:     issuer,
	}
	request := oidcSessionRequest(t, authenticator, struct {
		Issuer  string `json:"iss"`
		Subject string `json:"sub"`
		Expires int64  `json:"x"`
	}{
		Issuer:  issuer,
		Subject: "user-123",
		Expires: time.Now().Add(time.Hour).Unix(),
	})

	_, err := authenticator.Authenticate(request)

	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestOIDCAuthenticateRejectsRevokedSession(t *testing.T) {
	t.Parallel()

	const issuer = "https://identity.example.com/realms/lore"
	authenticator := &OIDC{
		repository: &oidcRepositoryStub{user: model.User{Enabled: true, SessionVersion: 5}},
		secret:     []byte("0123456789abcdef0123456789abcdef"),
		issuer:     issuer,
	}
	request := oidcSessionRequest(t, authenticator, session{
		Issuer: issuer, Subject: "user-123", Expires: time.Now().Add(time.Hour).Unix(), Version: 4,
	})

	_, err := authenticator.Authenticate(request)

	assert.ErrorIs(t, err, ErrUnauthenticated)
}

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
