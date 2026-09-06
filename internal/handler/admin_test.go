package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/store"
	"github.com/gi8lino/lore/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationSettingsFromForm(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"allow_user_registration": {"on"},
		"discussions_enabled":     {"on"},
	}
	request := httptest.NewRequest("POST", "/admin/settings", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, request.ParseForm())

	settings := applicationSettingsFromForm(request)

	assert.True(t, settings.AllowUserRegistration)
	assert.True(t, settings.DiscussionsEnabled)
}

func TestRenderingSettingsFromForm(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"content_language":    {"de-CH"},
		"coding_ligatures":    {"on"},
		"wiki_links":          {"on"},
		"syntax_highlighting": {"on"},
	}
	request := httptest.NewRequest("POST", "/admin/rendering", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, request.ParseForm())

	settings := renderingSettingsFromForm(request)

	assert.Equal(t, "de-CH", settings.ContentLanguage)
	assert.True(t, settings.CodingLigatures)
	assert.True(t, settings.WikiLinks)
	assert.True(t, settings.SyntaxHighlighting)
	assert.False(t, settings.Tables)
}

func TestRenderingLanguageValidation(t *testing.T) {
	t.Parallel()

	assert.True(t, isRenderingLanguage("de-CH"))
	assert.True(t, isRenderingLanguage("en"))
	assert.False(t, isRenderingLanguage("invalid"))
}

func TestAuthenticationSettingsFromForm(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"auth_mode":                    {"trusted-proxy"},
		"oidc_issuer":                  {" https://identity.example.com "},
		"oidc_client_id":               {" lore "},
		"oidc_group_claim":             {" groups "},
		"oidc_group_sync":              {"on"},
		"oidc_groups_authoritative":    {"on"},
		"oidc_admin_group":             {" /lore-admins "},
		"oidc_group_source":            {" /admins ", "/family"},
		"oidc_group_id":                {"7", "9"},
		"trusted_username_headers":     {"X-User, X-Backup-User, x-user"},
		"trusted_email_headers":        {"X-Email"},
		"trusted_display_name_headers": {"X-Name"},
		"trusted_group_headers":        {"X-Groups, X-Backup-Groups"},
		"trusted_admin_group":          {" lore-admins "},
	}
	request := httptest.NewRequest("POST", "/admin/authentication", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, request.ParseForm())

	settings := authenticationSettingsFromForm(request)

	assert.Equal(t, "trusted-proxy", settings.Mode)
	assert.Equal(t, "https://identity.example.com", settings.OIDCIssuer)
	assert.Equal(t, "lore", settings.OIDCClientID)
	assert.Equal(t, "groups", settings.OIDCGroupClaim)
	assert.True(t, settings.OIDCGroupSync)
	assert.True(t, settings.OIDCGroupsAuthoritative)
	assert.Equal(t, "/lore-admins", settings.OIDCAdminGroup)
	assert.Equal(t, []domain.OIDCGroupMapping{
		{OIDCGroup: "/admins", GroupID: 7},
		{OIDCGroup: "/family", GroupID: 9},
	}, settings.OIDCGroupMappings)
	assert.Equal(t, []string{"X-User", "X-Backup-User"}, settings.TrustedUsernameHeaders)
	assert.Equal(t, []string{"X-Email"}, settings.TrustedEmailHeaders)
	assert.Equal(t, []string{"X-Name"}, settings.TrustedDisplayNameHeaders)
	assert.Equal(t, []string{"X-Groups", "X-Backup-Groups"}, settings.TrustedGroupHeaders)
	assert.Equal(t, "lore-admins", settings.TrustedAdminGroup)
}

func TestAuthenticationSettingsProblemsRejectsInvalidGroupMappings(t *testing.T) {
	t.Parallel()

	settings := domain.AuthenticationSettings{
		Mode:                    "oidc",
		OIDCIssuer:              "https://identity.example.com",
		OIDCClientID:            "lore",
		OIDCGroupSync:           true,
		OIDCGroupsAuthoritative: true,
		OIDCGroupMappings: []domain.OIDCGroupMapping{
			{OIDCGroup: "/admins", GroupID: 1},
			{OIDCGroup: "/admins", GroupID: 2},
		},
	}
	problems := authenticationSettingsProblems(settings, RuntimeInfo{
		OIDCClientSecretConfigured: true,
		SessionSecretConfigured:    true,
	})

	require.Len(t, problems, 2)
	assert.Equal(t, "oidc_group_claim", problems[0].Field)
	assert.Equal(t, "oidc_group_mapping", problems[1].Field)
}

func TestAuthenticationSettingsProblems(t *testing.T) {
	t.Parallel()

	settings := domain.AuthenticationSettings{
		Mode:         "oidc",
		OIDCIssuer:   "https://identity.example.com",
		OIDCClientID: "lore",
	}
	problems := authenticationSettingsProblems(settings, RuntimeInfo{})

	require.Len(t, problems, 2)
	assert.Equal(t, "oidc_client_secret", problems[0].Field)
	assert.Equal(t, "session_secret", problems[1].Field)
}

func TestPendingOIDCIdentityID(t *testing.T) {
	t.Parallel()

	t.Run("accepts positive identifier", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest("POST", "/admin/oidc/pending/42/link", nil)

		request.SetPathValue("id", "42")

		id, err := pendingOIDCIdentityID(request)

		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("rejects invalid identifier", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest("POST", "/admin/oidc/pending/nope/link", nil)

		request.SetPathValue("id", "nope")

		_, err := pendingOIDCIdentityID(request)

		require.Error(t, err)
	})
}

type pendingIdentityStatusStub struct {
	oidcIdentityService
	id, actor int64
	rejected  bool
}

func (s *pendingIdentityStatusStub) SetPendingOIDCIdentityRejected(_ context.Context, id int64, rejected bool, actor int64) error {
	s.id, s.rejected, s.actor = id, rejected, actor
	return nil
}
func TestReopenPendingOIDCIdentity(t *testing.T) {
	users := &pendingIdentityStatusStub{rejected: true}
	mux := http.NewServeMux()

	mux.Handle("POST /admin/oidc/pending/{id}/reopen", ReopenPendingOIDCIdentity(users, slog.Default()))

	request := auth.WithUser(httptest.NewRequest("POST", "/admin/oidc/pending/42/reopen", nil), domain.User{ID: 7, Role: "admin"})
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)
	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, "/admin/users#pending-identities", response.Header().Get("Location"))
	assert.Equal(t, int64(42), users.id)
	assert.Equal(t, int64(7), users.actor)
	assert.False(t, users.rejected)
}

type passwordUserStub struct{ userManagementService }

func (*passwordUserStub) UpdateUser(context.Context, int64, string, bool, []int64, *bool) error {
	return nil
}

type passwordRepositoryStub struct {
	*store.Store
	id   int64
	hash string
}

func (s *passwordRepositoryStub) SetLocalCredential(_ context.Context, id int64, hash string) error {
	s.id, s.hash = id, hash
	return nil
}
func TestUpdateAdminUserSetsPasswordWithoutExternalAuthentication(t *testing.T) {
	repository := &passwordRepositoryStub{}
	form := url.Values{"role": {"admin"}, "account_enabled": {"on"}, "local_password": {"a-long-password-123"}, "local_password_confirm": {"a-long-password-123"}}
	request := httptest.NewRequest("POST", "/admin/users/7", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("id", "7")

	request = auth.WithUser(request, domain.User{ID: 7, Role: "admin"})
	response := httptest.NewRecorder()

	UpdateAdminUser(&passwordUserStub{}, nil, auth.NewLocal(repository, "http://localhost"), &Views{}, slog.Default())(response, request)
	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, int64(7), repository.id)
	assert.NotEmpty(t, repository.hash)
	assert.NotEqual(t, form.Get("local_password"), repository.hash)
}
func TestAdminAuthenticationTemplates(t *testing.T) {
	views, err := NewViews(web.Assets, slog.Default(), "test", "test", nil, RuntimeInfo{})

	require.NoError(t, err)

	data := ViewData{Runtime: RuntimeInfo{AuthModeOverride: "oidc"}}
	data.ApplicationSettings.Authentication.Mode = "none"
	html, err := renderTemplateHTML(views, "admin_configuration", "content", data)

	require.NoError(t, err)
	assert.Contains(t, string(html), "Authentication mode (active)")
	assert.Contains(t, string(html), "<option>OpenID Connect (OIDC)</option>")
	assert.Contains(t, string(html), "Saved fallback mode")
	assert.Contains(t, string(html), "Runtime authentication override active")
	assert.Contains(t, string(html), "PDF rendering")
	assert.Contains(t, string(html), "Test endpoint")
	assert.NotContains(t, string(html), "auth-recovery-form")

	html, err = renderTemplateHTML(views, "admin_users", "content", data)

	require.NoError(t, err)
	assert.Contains(t, string(html), `<details class="admin-user-local-password" data-admin-user-local-password>`)
}
