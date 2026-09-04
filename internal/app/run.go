package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/containeroo/httpgrace/server"
	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/config"
	"github.com/gi8lino/lore/internal/handler"
	"github.com/gi8lino/lore/internal/logging"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/routes"
	"github.com/gi8lino/lore/internal/service"
	"github.com/gi8lino/lore/internal/store"
	"github.com/gi8lino/lore/themes"
)

// Run configures dependencies and serves the wiki until the context is canceled.
func Run(ctx context.Context, appFS fs.FS, version, commit string, args []string, stdout, stderr io.Writer) error {
	cfg, err := config.Parse(args, version)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			_, _ = fmt.Fprint(stdout, err.Error())
			return nil
		}
		_, _ = fmt.Fprintln(stderr, err)
		return err
	}

	logger := logging.Setup(cfg.LogFormat, cfg.Debug, stdout)
	setupLogger := logger.With("component", "setup")
	setupLogger.Info(
		"starting Lore",
		"event", "app_starting",
		"version", version,
		"commit", commit,
	)

	availableThemes, err := themes.Load(cfg.ThemeDirectory)
	if err != nil {
		setupLogger.Error(
			"load themes",
			"event", "theme_load_failed",
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

	// Construct application services here so internal/app remains the single
	// composition root. The routing layer only receives ready-to-use
	// dependencies and decides which handlers consume them.
	administrationUseCases := service.NewAdministration(database)
	catalogUseCases := service.NewCatalog(database)
	draftUseCases := service.NewDrafts(database)
	groupUseCases := service.NewGroups(database)
	knowledgeUseCases := service.NewKnowledge(database)
	mediaUseCases := service.NewMedia(database)
	navigationUseCases := service.NewNavigation(database)
	pageUseCases := service.NewPages(database)
	preferenceUseCases := service.NewPreferences(database)
	recycleBinUseCases := service.NewRecycleBin(database)
	settingsUseCases := service.NewSettings(database)
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
				SessionSecret: cfg.SessionSecret,
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
		ListenAddress:              cfg.ListenAddress,
		PublicURL:                  cfg.PublicURL,
		AuthModeOverride:           string(cfg.AuthModeOverride),
		OIDCClientSecretConfigured: cfg.OIDCClientSecret != "",
		SessionSecretConfigured:    len(cfg.SessionSecret) >= 32,
		LocalLoginEnabled:          cfg.LocalLogin,
		ThemeDirectory:             cfg.ThemeDirectory,
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
		catalogUseCases,
		draftUseCases,
		groupUseCases,
		knowledgeUseCases,
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
