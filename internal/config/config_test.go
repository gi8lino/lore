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

func parseTestConfig(args []string) (Config, error) {
	flags := tinyflags.NewFlagSet("lore serve", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	return resolve(), nil
}
