package auth

import (
	"context"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureBrowserAuth(t *testing.T) {
	t.Parallel()

	configured, err := ConfigureBrowserAuth(
		context.Background(),
		BrowserConfig{ModeOverride: AuthModeNone},
		nil,
	)

	require.NoError(t, err)
	assert.NotNil(t, configured.Authenticator)
	assert.NotNil(t, configured.Login)
	assert.NotNil(t, configured.Callback)
	assert.NotNil(t, configured.Validate)
	assert.NotNil(t, configured.Local)
	assert.NotNil(t, configured.LocalLoginAllowed)
}

func TestBrowserAuthenticatorForSettings(t *testing.T) {
	t.Parallel()

	browser := &browserAuthenticator{none: NewNone(nil), local: NewLocal(nil, "")}

	t.Run("no authentication", func(t *testing.T) {
		authenticator, err := browser.authenticatorForSettings(
			context.Background(),
			domain.AuthenticationSettings{Mode: string(AuthModeNone)},
		)

		require.NoError(t, err)
		assert.IsType(t, &None{}, authenticator)
	})

	t.Run("local", func(t *testing.T) {
		authenticator, err := browser.authenticatorForSettings(
			context.Background(),
			domain.AuthenticationSettings{Mode: string(AuthModeLocal)},
		)

		require.NoError(t, err)
		assert.IsType(t, &Local{}, authenticator)
	})

	t.Run("trusted proxy", func(t *testing.T) {
		authenticator, err := browser.authenticatorForSettings(
			context.Background(),
			domain.AuthenticationSettings{
				Mode:                   string(AuthModeTrustedProxy),
				TrustedUsernameHeaders: []string{"X-User"},
			},
		)

		require.NoError(t, err)
		assert.IsType(t, &TrustedProxy{}, authenticator)
	})

	t.Run("rejects unknown mode", func(t *testing.T) {
		_, err := browser.authenticatorForSettings(
			context.Background(),
			domain.AuthenticationSettings{Mode: "invalid"},
		)
		require.Error(t, err)
	})
}

func TestBrowserAuthenticatorValidatesOIDCSecrets(t *testing.T) {
	t.Parallel()

	settings := domain.AuthenticationSettings{
		Mode:         string(AuthModeOIDC),
		OIDCIssuer:   "https://identity.example.com",
		OIDCClientID: "lore",
	}
	browser := &browserAuthenticator{}

	assert.EqualError(t, browser.validateSettings(settings), "OIDC client secret is not configured")

	browser.oidcConfig.ClientSecret = "client-secret"

	assert.EqualError(t, browser.validateSettings(settings), "OIDC session secret must be at least 32 characters")
}
