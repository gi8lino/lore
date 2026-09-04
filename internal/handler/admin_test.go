package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/service"
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
		"oidc_group_source":            {" /admins ", "/family"},
		"oidc_group_id":                {"7", "9"},
		"trusted_username_headers":     {"X-User, X-Backup-User, x-user"},
		"trusted_email_headers":        {"X-Email"},
		"trusted_display_name_headers": {"X-Name"},
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
	assert.Equal(t, []service.OIDCGroupMapping{
		{OIDCGroup: "/admins", GroupID: 7},
		{OIDCGroup: "/family", GroupID: 9},
	}, settings.OIDCGroupMappings)
	assert.Equal(t, []string{"X-User", "X-Backup-User"}, settings.TrustedUsernameHeaders)
	assert.Equal(t, []string{"X-Email"}, settings.TrustedEmailHeaders)
	assert.Equal(t, []string{"X-Name"}, settings.TrustedDisplayNameHeaders)
}

func TestAuthenticationSettingsProblemsRejectsInvalidGroupMappings(t *testing.T) {
	t.Parallel()

	settings := service.AuthenticationSettings{
		Mode:                    "oidc",
		OIDCIssuer:              "https://identity.example.com",
		OIDCClientID:            "lore",
		OIDCGroupSync:           true,
		OIDCGroupsAuthoritative: true,
		OIDCGroupMappings: []service.OIDCGroupMapping{
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

	settings := service.AuthenticationSettings{
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
