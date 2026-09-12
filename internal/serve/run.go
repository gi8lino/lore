package serve

import (
	"context"
	"io"
	"io/fs"

	"github.com/containeroo/httpgrace/server"
	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/handler"
	"github.com/gi8lino/lore/internal/logging"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/routes"
	"github.com/gi8lino/lore/internal/secrets"
	"github.com/gi8lino/lore/internal/service"
	"github.com/gi8lino/lore/internal/store"
	"github.com/gi8lino/lore/themes"
)

// Run configures dependencies and serves Lore until the context is canceled.
func Run(
	ctx context.Context,
	appFS fs.FS,
	cfg Config,
	overrides map[string]any,
	version, commit string,
	stdout io.Writer,
) error {
	logger := logging.Setup(cfg.LogFormat, cfg.Debug, stdout)
	setupLogger := logger.With("component", "setup")

	setupLogger.Info(
		"starting Lore",
		"event", "app_starting",
		"version", version,
		"commit", commit,
	)

	if len(overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", overrides,
		)
	}

	availableThemes, err := themes.Load(cfg.ThemeDirectory)
	if err != nil {
		setupLogger.Error(
			"load themes",
			"event", "theme_load_failed",
			"error", err,
		)
		return err
	}

	secretCipher, err := secrets.New(cfg.EncryptionKey)
	if err != nil {
		setupLogger.Error(
			"configure application encryption",
			"event", "application_encryption_failed",
			"error", err,
		)
		return err
	}

	database, err := store.Open(ctx, cfg.DatabaseURL, setupLogger)
	if err != nil {
		setupLogger.Error(
			"open database",
			"event",
			"database_open_failed",
			"error", err,
		)
		return err
	}

	defer database.Close()

	// Construct application services here so internal/serve remains the single
	// composition root. The routing layer only receives ready-to-use
	// dependencies and decides which handlers consume them.
	administrationUseCases := service.NewAdministration(database)
	accessUseCases := service.NewAccess(database)
	catalogUseCases := service.NewCatalog(database)
	draftUseCases := service.NewDrafts(database)
	groupUseCases := service.NewGroups(database)
	knowledgeUseCases := service.NewKnowledge(database)
	notificationUseCases := service.NewNotifications(database)
	mediaUseCases := service.NewMedia(database)
	navigationUseCases := service.NewNavigation(database)
	webhookUseCases := service.NewWebhooks(database, secretCipher, logger.With("component", "webhooks"), cfg.PublicURL)
	pageUseCases := service.NewPages(database, logger, webhookUseCases)
	preferenceUseCases := service.NewPreferences(database)
	recycleBinUseCases := service.NewRecycleBin(database)
	settingsUseCases := service.NewSettings(database, secretCipher)
	systemUseCases := service.NewSystem(database)
	templateUseCases := service.NewTemplates(database)
	tokenUseCases := service.NewTokens(database)
	userUseCases := service.NewUsers(database)
	viewDataUseCases := handler.NewViewDataLoader(
		preferenceUseCases,
		navigationUseCases,
		catalogUseCases,
		settingsUseCases,
		knowledgeUseCases,
		notificationUseCases,
		accessUseCases,
	)

	browserAuth, err := auth.ConfigureBrowserAuth(
		ctx,
		auth.BrowserConfig{
			ModeOverride: cfg.AuthModeOverride,
			TrustedProxy: auth.TrustedProxyHeaders{
				Username:    cfg.TrustedUsernameHeaders,
				Email:       cfg.TrustedEmailHeaders,
				DisplayName: cfg.TrustedDisplayNameHeaders,
			},
			OIDC: auth.OIDCConfig{
				ClientID:      cfg.OIDCClientID,
				ClientSecret:  cfg.OIDCClientSecret,
				Issuer:        cfg.OIDCIssuer,
				SessionSecret: cfg.OIDCSessionSecret,
				PublicURL:     cfg.PublicURL,
			},
			LocalLoginEnabled: cfg.LocalLogin,
		},
		database,
	)
	if err != nil {
		setupLogger.Error(
			"configure browser auth",
			"event", "browser_auth_failed",
			"error", err,
		)
		return err
	}

	bearerAuth := auth.NewBearer(database)

	views, err := handler.NewViews(appFS, logger, version, commit, availableThemes, handler.RuntimeInfo{
		ListenAddress:                     cfg.ListenAddress,
		PublicURL:                         cfg.PublicURL,
		PDFURL:                            cfg.PDFURL,
		AuthModeOverride:                  string(cfg.AuthModeOverride),
		OIDCIssuerOverride:                cfg.OIDCIssuer,
		OIDCClientIDOverride:              cfg.OIDCClientID,
		TrustedUsernameHeadersOverride:    cfg.TrustedUsernameHeaders,
		TrustedEmailHeadersOverride:       cfg.TrustedEmailHeaders,
		TrustedDisplayNameHeadersOverride: cfg.TrustedDisplayNameHeaders,
		OIDCClientSecretConfigured:        cfg.OIDCClientSecret != "",
		OIDCSessionSecretConfigured:       len(cfg.OIDCSessionSecret) >= 32,
		EncryptionKeyConfigured:           secretCipher.Configured(),
		LocalLoginEnabled:                 cfg.LocalLogin,
		ThemeDirectory:                    cfg.ThemeDirectory,
	})
	if err != nil {
		setupLogger.Error(
			"create views",
			"event", "views_create_failed",
			"error", err,
		)
		return err
	}

	ctx, stop := server.SignalContext(ctx)
	defer stop()

	router := routes.New(
		appFS,
		views,
		markdown.New(),
		browserAuth,
		bearerAuth,
		administrationUseCases,
		accessUseCases,
		catalogUseCases,
		draftUseCases,
		groupUseCases,
		knowledgeUseCases,
		notificationUseCases,
		mediaUseCases,
		navigationUseCases,
		pageUseCases,
		preferenceUseCases,
		recycleBinUseCases,
		settingsUseCases,
		systemUseCases,
		templateUseCases,
		tokenUseCases,
		userUseCases,
		webhookUseCases,
		viewDataUseCases,
		logger.With("component", "server"),
		cfg.AccessLog,
	)
	if err := server.Run(ctx, cfg.ListenAddress, router, setupLogger, server.WithMaxHeaderValueCount(100)); err != nil {
		setupLogger.Error("run server", "event", "server_run_failed", "error", err)
		return err
	}

	return nil
}
