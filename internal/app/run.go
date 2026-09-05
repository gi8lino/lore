package app

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
	"github.com/gi8lino/lore/internal/service"
	"github.com/gi8lino/lore/internal/store"
	"github.com/gi8lino/lore/themes"
)

// Run configures dependencies and serves the wiki until the context is canceled.
func Run(
	ctx context.Context,
	appFS fs.FS,
	listenAddress string,
	databaseURL string,
	publicURL string,
	pdfURL string,
	authModeOverride auth.AuthMode,
	trustedUsernameHeaders []string,
	trustedEmailHeaders []string,
	trustedDisplayNameHeaders []string,
	oidcIssuer string,
	oidcClientID string,
	oidcClientSecret string,
	sessionSecret string,
	localLogin bool,
	themeDirectory string,
	logFormat logging.LogFormat,
	debug bool,
	accessLog bool,
	overrides map[string]any,
	version, commit string,
	stdout io.Writer,
) error {
	logger := logging.Setup(logFormat, debug, stdout)
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

	availableThemes, err := themes.Load(themeDirectory)
	if err != nil {
		setupLogger.Error(
			"load themes",
			"event", "theme_load_failed",
			"error", err,
		)
		return err
	}

	database, err := store.Open(ctx, databaseURL, setupLogger)
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
			ModeOverride: authModeOverride,
			TrustedProxy: auth.TrustedProxyHeaders{
				Username:    trustedUsernameHeaders,
				Email:       trustedEmailHeaders,
				DisplayName: trustedDisplayNameHeaders,
			},
			OIDC: auth.OIDCConfig{
				ClientID:      oidcClientID,
				ClientSecret:  oidcClientSecret,
				Issuer:        oidcIssuer,
				SessionSecret: sessionSecret,
				PublicURL:     publicURL,
			},
			LocalLoginEnabled: localLogin,
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
		ListenAddress:              listenAddress,
		PublicURL:                  publicURL,
		PDFURL:                     pdfURL,
		AuthModeOverride:           string(authModeOverride),
		OIDCClientSecretConfigured: oidcClientSecret != "",
		SessionSecretConfigured:    len(sessionSecret) >= 32,
		LocalLoginEnabled:          localLogin,
		ThemeDirectory:             themeDirectory,
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
		accessLog,
	)
	if err := server.Run(ctx, listenAddress, router, setupLogger, server.WithMaxHeaderValueCount(100)); err != nil {
		setupLogger.Error("run server", "event", "server_run_failed", "error", err)
		return err
	}

	return nil
}
