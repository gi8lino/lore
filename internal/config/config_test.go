package config

import (
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentPrefix(t *testing.T) {
	t.Setenv("LORE__DATABASE_URL", "postgres://example/lore")
	t.Setenv("LORE__AUTH_MODE", string(auth.AuthModeTrustedProxy))

	cfg, err := parseTestConfig(nil)

	require.NoError(t, err)
	assert.Equal(t, "postgres://example/lore", cfg.DatabaseURL)
	assert.Equal(t, auth.AuthModeTrustedProxy, cfg.AuthModeOverride)
}

func TestAuthModeRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	_, err := parseTestConfig([]string{
		"--database-url", "postgres://example/lore",
		"--auth-mode", "invalid",
	})

	require.Error(t, err)
}

func TestAuthModeDefaultsToDatabaseManaged(t *testing.T) {
	t.Parallel()

	cfg, err := parseTestConfig([]string{"--database-url", "postgres://example/lore"})

	require.NoError(t, err)
	assert.Empty(t, cfg.AuthModeOverride)
}

func TestLocalRecoveryLoginCanBeEnabledFromEnvironment(t *testing.T) {
	t.Setenv("LORE__DATABASE_URL", "postgres://example/lore")
	t.Setenv("LORE__LOCAL_LOGIN", "true")

	cfg, err := parseTestConfig(nil)

	require.NoError(t, err)
	assert.True(t, cfg.LocalLogin)
}

func TestOIDCSecretsRemainDeploymentConfiguration(t *testing.T) {
	t.Parallel()

	cfg, err := parseTestConfig([]string{
		"--database-url", "postgres://example/lore",
		"--oidc-client-secret", "client-secret",
		"--session-secret", "0123456789abcdef0123456789abcdef",
	})

	require.NoError(t, err)
	assert.Equal(t, "client-secret", cfg.OIDCClientSecret)
	assert.Equal(t, "0123456789abcdef0123456789abcdef", cfg.SessionSecret)
}

func TestOverriddenValuesMaskSecrets(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("lore serve", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)
	databaseURL := "postgres://lore:database-secret@postgres:5432/lore?sslmode=disable"
	oidcSecret := "oidc-client-secret-value"
	sessionSecret := "0123456789abcdef0123456789abcdef"

	require.NoError(t, flags.Parse([]string{
		"--database-url", databaseURL,
		"--oidc-client-secret", oidcSecret,
		"--session-secret", sessionSecret,
	}))

	_ = resolve()

	overrides := flags.OverriddenValues()

	assert.Contains(t, overrides, "database-url")
	assert.Contains(t, overrides, "oidc-client-secret")
	assert.Contains(t, overrides, "session-secret")
	assert.NotEqual(t, databaseURL, overrides["database-url"])
	assert.NotEqual(t, oidcSecret, overrides["oidc-client-secret"])
	assert.NotEqual(t, sessionSecret, overrides["session-secret"])
}

func parseTestConfig(args []string) (Config, error) {
	flags := tinyflags.NewFlagSet("lore serve", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	return resolve(), nil
}

func TestPDFURLFromEnvironmentIncludesPath(t *testing.T) {
	t.Setenv("LORE__DATABASE_URL", "postgres://example/lore")
	t.Setenv("LORE__PDF_URL", "http://html2pdf:8080/custom/render?profile=wiki")

	cfg, err := parseTestConfig(nil)

	require.NoError(t, err)
	assert.Equal(t, "http://html2pdf:8080/custom/render?profile=wiki", cfg.PDFURL)
}

func TestPDFURLValidation(t *testing.T) {
	for _, value := range []string{"pdf:8080/render", "http://pdf:8080", "file:///render", "http://pdf/render#fragment", "http://user:password@pdf/render"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseTestConfig([]string{"--database-url", "postgres://example/lore", "--pdf-url", value})
			require.Error(t, err)
		})
	}

	cfg, err := parseTestConfig([]string{"--database-url", "postgres://example/lore"})

	require.NoError(t, err)
	assert.Empty(t, cfg.PDFURL)
}

func TestLocalAuthenticationOverride(t *testing.T) {
	t.Parallel()
	cfg, err := parseTestConfig([]string{"--database-url", "postgres://example/lore", "--auth-mode", "local"})
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeLocal, cfg.AuthModeOverride)
}

func TestLocalAuthenticationOverrideFromEnvironment(t *testing.T) {
	t.Setenv("LORE__DATABASE_URL", "postgres://example/lore")
	t.Setenv("LORE__AUTH_MODE", "local")
	cfg, err := parseTestConfig(nil)
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeLocal, cfg.AuthModeOverride)
}
