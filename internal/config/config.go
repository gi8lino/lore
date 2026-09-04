package config

import (
	"fmt"
	"net"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/logging"
)

var trustedUsernameHeaders = []string{
	"X-Forwarded-User",
	"X-Auth-Request-User",
	"Remote-User",
}

var trustedEmailHeaders = []string{
	"X-Forwarded-Email",
	"X-Auth-Request-Email",
	"X-Authentik-Email",
}

var trustedDisplayNameHeaders = []string{
	"X-Forwarded-Name",
	"X-Auth-Request-Preferred-Username",
	"X-Authentik-Name",
}

// Config contains deployment-level runtime configuration for the wiki application.
type Config struct {
	// ListenAddress is the TCP address used by the HTTP server.
	ListenAddress string
	// DatabaseURL is the PostgreSQL connection URL.
	DatabaseURL string
	// PublicURL is the externally visible base URL.
	PublicURL string
	// AuthModeOverride forces one browser authentication mode for recovery when non-empty.
	AuthModeOverride auth.AuthMode
	// TrustedUsernameHeaders are used only by the trusted-proxy runtime override.
	TrustedUsernameHeaders []string
	// TrustedEmailHeaders are used only by the trusted-proxy runtime override.
	TrustedEmailHeaders []string
	// TrustedDisplayNameHeaders are used only by the trusted-proxy runtime override.
	TrustedDisplayNameHeaders []string
	// OIDCIssuer is used only by the OIDC runtime override.
	OIDCIssuer string
	// OIDCClientID is used only by the OIDC runtime override.
	OIDCClientID string
	// OIDCClientSecret is the deployment-managed OIDC client secret.
	OIDCClientSecret string
	// SessionSecret signs browser session cookies.
	SessionSecret string
	// LocalLogin enables the local recovery login alongside the configured mode.
	LocalLogin bool
	// ThemeDirectory optionally overlays embedded themes with TOML files from disk.
	ThemeDirectory string
	// LogFormat selects structured text or JSON logging.
	LogFormat logging.LogFormat
	// Debug enables verbose diagnostic logging.
	Debug bool
	// AccessLog enables HTTP request logging.
	AccessLog bool
	// Overridden records configuration values explicitly overridden by flags or environment variables.
	Overridden map[string]any
}

// Parse parses command-line arguments and LORE_ environment variables into Config.
func Parse(args []string, version string) (Config, error) {
	cfg := Config{}
	flags := tinyflags.NewFlagSet("lore", tinyflags.ContinueOnError)
	flags.Version(version)
	flags.EnvPrefix("LORE")

	// Server
	listen := flags.TCPAddr("listen-address", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}, "Address on which the web server listens").
		Short("a").
		Placeholder("ADDR").
		Value()
	flags.StringVar(&cfg.DatabaseURL, "database-url", "", "PostgreSQL connection URL").
		Required().
		Placeholder("URL").
		Value()
	flags.StringVar(&cfg.PublicURL, "public-url", "http://localhost:8080", "Externally visible base URL").
		Placeholder("URL").
		Value()
	flags.BoolVar(&cfg.LocalLogin, "local-login", false, "Enable the local recovery login alongside the configured authentication mode").
		Value()
	flags.StringVar(&cfg.ThemeDirectory, "theme-directory", "", "Directory containing custom theme TOML files that override or extend embedded themes").
		Placeholder("DIR").
		Value()

		// Auth
	authModeFlag := tinyflags.Enum(
		flags,
		"auth-mode",
		auth.AuthModeNone,
		"Emergency override for the database-managed authentication mode",
		auth.AuthModeNone,
		auth.AuthModeTrustedProxy,
		auth.AuthModeOIDC,
	).
		Placeholder("MODE")

		// Trusted-proxy
	flags.StringSliceVar(&cfg.TrustedUsernameHeaders, "trusted-username-headers", trustedUsernameHeaders, "Trusted-proxy username headers used only with the authentication override").
		Value()
	flags.StringSliceVar(&cfg.TrustedEmailHeaders, "trusted-email-headers", trustedEmailHeaders, "Trusted-proxy email headers used only with the authentication override").
		Value()
	flags.StringSliceVar(&cfg.TrustedDisplayNameHeaders, "trusted-display-name-headers", trustedDisplayNameHeaders, "Trusted-proxy display-name headers used only with the authentication override").
		Value()

		// OIDC
	flags.StringVar(&cfg.OIDCIssuer, "oidc-issuer", "", "OIDC issuer used only with the authentication override").
		Placeholder("URL").
		Value()
	flags.StringVar(&cfg.OIDCClientID, "oidc-client-id", "", "OIDC client ID used only with the authentication override").
		Value()
	flags.StringVar(&cfg.OIDCClientSecret, "oidc-client-secret", "", "OIDC client secret used when OIDC is enabled in the administration UI").
		Value()
	flags.StringVar(&cfg.SessionSecret, "session-secret", "", "Secret used to sign OIDC login sessions").
		Validate(func(value string) error {
			if value == "" {
				return nil
			}
			if len(value) < 32 {
				return fmt.Errorf("session secret must be at least 32 characters")
			}
			return nil
		}).
		Value()

		// Logging
	logFormat := flags.String("log-format", "json", "Log output format").
		Choices(string(logging.LogFormatText), string(logging.LogFormatJSON)).
		Short("l").
		Placeholder("FORMAT").
		Value()
	flags.BoolVar(&cfg.Debug, "debug", false, "Enable verbose diagnostic logging").
		Short("d").
		Value()
	flags.BoolVar(&cfg.AccessLog, "access-log", false, "Enable HTTP request access logging").
		Value()

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	if authModeFlag.Changed() {
		cfg.AuthModeOverride = *authModeFlag.Value()
	}

	cfg.ListenAddress = (*listen).String()
	cfg.LogFormat = logging.LogFormat(*logFormat)
	cfg.Overridden = flags.OverriddenValues()
	return cfg, nil
}
