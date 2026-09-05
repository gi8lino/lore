package routes

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/handler"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/middleware"
	"github.com/gi8lino/lore/internal/service"
)

// addRoutes registers the complete HTTP surface and applies route-specific access policies.
func addRoutes(
	mux *http.ServeMux,
	appFS fs.FS,
	views *handler.Views,
	renderer *markdown.Renderer,
	browserAuth auth.BrowserAuth,
	administrationUseCases *service.Administration,
	catalogUseCases *service.Catalog,
	draftUseCases *service.Drafts,
	groupUseCases *service.Groups,
	knowledgeUseCases *service.Knowledge,
	mediaUseCases *service.Media,
	navigationUseCases *service.Navigation,
	pageUseCases *service.Pages,
	preferenceUseCases *service.Preferences,
	recycleBinUseCases *service.RecycleBin,
	settingsUseCases *service.Settings,
	systemUseCases *service.System,
	templateUseCases *service.Templates,
	tokenUseCases *service.Tokens,
	userUseCases *service.Users,
	viewDataUseCases *handler.ViewDataLoader,
	logger *slog.Logger,
	browserAuthn middleware.Middleware,
	mediaAuthn middleware.Middleware,
	apiAuthn middleware.Middleware,
	adminAuthz middleware.Middleware,
	editorAuthz middleware.Middleware,
) {
	// Public infrastructure and authentication routes.
	mux.HandleFunc("GET /healthz", handler.Health(systemUseCases))
	mux.Handle("GET /assets/", handler.Assets(appFS))
	mux.Handle("GET /sw.js", handler.ServiceWorker(appFS))
	mux.Handle("GET /auth/login", browserAuth.Login)
	mux.Handle("GET /auth/local", handler.LocalLogin(settingsUseCases, systemUseCases, browserAuth, views))
	mux.Handle("POST /auth/local", handler.LocalLogin(settingsUseCases, systemUseCases, browserAuth, views))
	mux.Handle("GET /setup", handler.Setup(settingsUseCases, systemUseCases, browserAuth, views))
	mux.Handle("POST /setup", handler.Setup(settingsUseCases, systemUseCases, browserAuth, views))
	if browserAuth.Callback != nil {
		mux.Handle("GET /auth/callback", browserAuth.Callback)
	}

	// Browser routes require the configured browser authenticator.
	mux.Handle("POST /auth/logout", browserAuthn(auth.Logout(browserAuth.Local)))
	mux.Handle("GET /{$}", browserAuthn(handler.Home(viewDataUseCases, catalogUseCases, draftUseCases, views)))
	mux.Handle("GET /search", browserAuthn(handler.Search(viewDataUseCases, catalogUseCases, views)))
	mux.Handle("GET /graph", browserAuthn(handler.KnowledgeGraphPage(viewDataUseCases, views)))
	mux.Handle("GET /p/{id}", browserAuthn(handler.PagePermalink(catalogUseCases, logger)))
	mux.Handle(
		"GET /settings",
		browserAuthn(handler.Settings(viewDataUseCases, userUseCases, tokenUseCases, mediaUseCases, browserAuth.Local, views)),
	)
	mux.Handle(
		"GET /admin",
		browserAuthn(adminAuthz(handler.Administration(viewDataUseCases, administrationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/configuration",
		browserAuthn(adminAuthz(handler.AdminConfiguration(viewDataUseCases, groupUseCases, userUseCases, views))),
	)
	mux.Handle(
		"GET /admin/rendering",
		browserAuthn(adminAuthz(handler.AdminRendering(viewDataUseCases, renderer, views))),
	)
	mux.Handle(
		"GET /admin/health",
		browserAuthn(adminAuthz(handler.AdminDocumentationHealth(viewDataUseCases, administrationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/templates",
		browserAuthn(adminAuthz(handler.AdminPageTemplates(viewDataUseCases, templateUseCases, views))),
	)
	mux.Handle(
		"GET /admin/audit",
		browserAuthn(adminAuthz(handler.AdminAudit(viewDataUseCases, administrationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/snippets",
		browserAuthn(adminAuthz(handler.AdminSnippets(viewDataUseCases, knowledgeUseCases, views))),
	)
	mux.Handle("POST /admin/snippets", browserAuthn(adminAuthz(handler.SaveAdminSnippet(knowledgeUseCases, logger))))
	mux.Handle(
		"POST /admin/snippets/{id}",
		browserAuthn(adminAuthz(handler.SaveAdminSnippet(knowledgeUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/snippets/{id}/delete",
		browserAuthn(adminAuthz(handler.DeleteAdminSnippet(knowledgeUseCases, logger))),
	)
	mux.Handle(
		"GET /admin/pages",
		browserAuthn(adminAuthz(handler.AdminPages(viewDataUseCases, catalogUseCases, groupUseCases, views))),
	)
	mux.Handle(
		"POST /admin/pages/bulk",
		browserAuthn(adminAuthz(handler.BulkAdminPages(pageUseCases, catalogUseCases, mediaUseCases, logger))),
	)
	mux.Handle("GET /admin/import", browserAuthn(adminAuthz(handler.AdminImport(viewDataUseCases, views))))
	mux.Handle("POST /admin/import", browserAuthn(adminAuthz(handler.ImportPages(pageUseCases, logger))))
	mux.Handle(
		"POST /admin/templates",
		browserAuthn(adminAuthz(handler.CreateAdminPageTemplate(templateUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/templates/{id}",
		browserAuthn(adminAuthz(handler.UpdateAdminPageTemplate(templateUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/templates/{id}/delete",
		browserAuthn(adminAuthz(handler.DeleteAdminPageTemplate(templateUseCases, logger))),
	)
	mux.Handle("POST /admin/rendering", browserAuthn(adminAuthz(handler.SaveAdminRendering(settingsUseCases, logger))))
	mux.Handle(
		"GET /admin/users",
		browserAuthn(adminAuthz(handler.AdminUsers(viewDataUseCases, userUseCases, groupUseCases, views))),
	)
	mux.Handle(
		"GET /admin/groups",
		browserAuthn(adminAuthz(handler.AdminGroups(viewDataUseCases, groupUseCases, views))),
	)
	mux.Handle(
		"GET /admin/navigation",
		browserAuthn(adminAuthz(handler.AdminNavigation(viewDataUseCases, navigationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/bin",
		browserAuthn(adminAuthz(handler.AdminBin(viewDataUseCases, recycleBinUseCases, views))),
	)
	mux.Handle(
		"GET /admin/tags",
		browserAuthn(adminAuthz(handler.AdminTags(viewDataUseCases, administrationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/tokens",
		browserAuthn(adminAuthz(handler.AdminTokens(viewDataUseCases, userUseCases, tokenUseCases, views))),
	)
	mux.Handle(
		"GET /admin/exports",
		browserAuthn(adminAuthz(handler.AdminExports(viewDataUseCases, navigationUseCases, views))),
	)
	mux.Handle(
		"GET /admin/images",
		browserAuthn(adminAuthz(handler.AdminImages(viewDataUseCases, mediaUseCases, views))),
	)
	mux.Handle("POST /admin/settings", browserAuthn(adminAuthz(handler.SaveAdminSettings(settingsUseCases, logger))))
	mux.Handle(
		"POST /admin/authentication",
		browserAuthn(adminAuthz(handler.SaveAdminAuthentication(settingsUseCases, browserAuth, views))),
	)

	mux.Handle("GET /media/{id}/{name...}", mediaAuthn(handler.ServeImage(mediaUseCases)))
	mux.Handle("GET /attachments/{id}/{name...}", mediaAuthn(handler.ServeAttachment(mediaUseCases)))
	mux.Handle("POST /settings/preferences", browserAuthn(handler.SavePreferences(preferenceUseCases, views)))
	mux.Handle("POST /settings/local-password", browserAuthn(handler.ChangeLocalPassword(browserAuth.Local, logger)))
	mux.Handle(
		"POST /settings/preferences/page-contents",
		browserAuthn(handler.SavePageContentsPreference(preferenceUseCases, views)),
	)
	mux.Handle(
		"POST /settings/preferences/navigation-state",
		browserAuthn(handler.SaveNavigationState(preferenceUseCases, logger)),
	)
	mux.Handle(
		"POST /settings/preferences/sidebar-width",
		browserAuthn(handler.SaveSidebarWidth(preferenceUseCases, logger)),
	)
	mux.Handle("POST /settings/saved-searches", browserAuthn(handler.CreateSavedSearch(knowledgeUseCases, logger)))
	mux.Handle(
		"POST /settings/saved-searches/{id}/delete",
		browserAuthn(handler.DeleteSavedSearch(knowledgeUseCases, logger)),
	)
	mux.Handle("POST /settings/tokens", browserAuthn(handler.CreatePersonalToken(tokenUseCases, logger)))
	mux.Handle("DELETE /settings/tokens/{id}", browserAuthn(handler.DeletePersonalToken(tokenUseCases, logger)))
	mux.Handle(
		"POST /admin/users/{id}",
		browserAuthn(adminAuthz(handler.UpdateAdminUser(userUseCases, settingsUseCases, browserAuth.Local, views, logger))),
	)
	mux.Handle(
		"POST /admin/users/{id}/sessions/revoke",
		browserAuthn(adminAuthz(handler.RevokeAdminUserSessions(userUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/users/{id}/oidc/remove",
		browserAuthn(adminAuthz(handler.RemoveAdminOIDCIdentity(userUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/oidc/pending/{id}/approve",
		browserAuthn(adminAuthz(handler.ApprovePendingOIDCIdentity(userUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/oidc/pending/{id}/link",
		browserAuthn(adminAuthz(handler.LinkPendingOIDCIdentity(userUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/oidc/pending/{id}/reject",
		browserAuthn(adminAuthz(handler.RejectPendingOIDCIdentity(userUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/oidc/pending/{id}/reopen",
		browserAuthn(adminAuthz(handler.ReopenPendingOIDCIdentity(userUseCases, logger))),
	)
	mux.Handle("POST /admin/groups", browserAuthn(adminAuthz(handler.CreateAdminGroup(groupUseCases, logger))))
	mux.Handle(
		"POST /admin/navigation",
		browserAuthn(adminAuthz(handler.SaveAdminNavigationIcon(navigationUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/bin/restore/{slug...}",
		browserAuthn(adminAuthz(handler.RestoreAdminPage(recycleBinUseCases, logger))),
	)
	mux.Handle(
		"POST /admin/bin/delete/{slug...}",
		browserAuthn(adminAuthz(handler.PermanentlyDeleteAdminPage(recycleBinUseCases, logger))),
	)
	mux.Handle("GET /api/admin/users", apiAuthn(adminAuthz(handler.SearchAdminUsers(userUseCases, logger))))
	mux.Handle("GET /api/icons", apiAuthn(editorAuthz(handler.SearchIcons())))
	mux.Handle(
		"GET /api/admin/groups/{id}/members",
		apiAuthn(adminAuthz(handler.AdminGroupMembers(groupUseCases, logger))),
	)
	mux.Handle(
		"POST /api/admin/groups/{id}/members",
		apiAuthn(adminAuthz(handler.AddAdminGroupMember(groupUseCases, userUseCases, logger))),
	)
	mux.Handle(
		"DELETE /api/admin/groups/{id}/members/{userID}",
		apiAuthn(adminAuthz(handler.RemoveAdminGroupMember(groupUseCases, logger))),
	)
	mux.Handle("POST /admin/groups/{id}/delete", browserAuthn(adminAuthz(handler.DeleteAdminGroup(groupUseCases, logger))))
	mux.Handle(
		"POST /admin/tags/{id}/delete",
		browserAuthn(adminAuthz(handler.DeleteAdminTag(administrationUseCases, logger))),
	)
	mux.Handle("POST /admin/tokens", browserAuthn(adminAuthz(handler.CreateAdminToken(tokenUseCases, logger))))
	mux.Handle("DELETE /admin/tokens/{id}", browserAuthn(adminAuthz(handler.DeleteAdminToken(tokenUseCases, logger))))
	mux.Handle(
		"POST /admin/export",
		browserAuthn(adminAuthz(handler.ExportPages(catalogUseCases, navigationUseCases, mediaUseCases, logger))),
	)
	mux.Handle(
		"GET /export/markdown/{slug...}",
		browserAuthn(handler.ExportPageMarkdown(catalogUseCases, mediaUseCases, logger)),
	)
	mux.Handle(
		"GET /export/pdf/{slug...}",
		browserAuthn(handler.ExportPagePDF(
			catalogUseCases,
			settingsUseCases,
			navigationUseCases,
			knowledgeUseCases,
			mediaUseCases,
			renderer,
			views,
			logger,
		)),
	)
	mux.Handle("POST /pages/delete/{slug...}", browserAuthn(adminAuthz(handler.DeletePageForm(pageUseCases, views))))
	mux.Handle("POST /pages/move/{slug...}", browserAuthn(editorAuthz(handler.MovePageForm(pageUseCases, logger))))
	mux.Handle("POST /pages/review/{slug...}", browserAuthn(editorAuthz(handler.ReviewPageForm(pageUseCases, logger))))
	mux.Handle("POST /page-comments/{slug...}", browserAuthn(handler.AddPageComment(pageUseCases, views)))
	mux.Handle(
		"POST /page-comments/resolve/{id}",
		browserAuthn(editorAuthz(handler.ResolvePageComment(pageUseCases, views))),
	)
	mux.Handle(
		"GET /pages/new",
		browserAuthn(editorAuthz(handler.EditPage(
			viewDataUseCases,
			catalogUseCases,
			groupUseCases,
			knowledgeUseCases,
			templateUseCases,
			views,
		))),
	)
	mux.Handle(
		"GET /edit/{slug...}",
		browserAuthn(editorAuthz(handler.EditPage(
			viewDataUseCases,
			catalogUseCases,
			groupUseCases,
			knowledgeUseCases,
			templateUseCases,
			views,
		))),
	)
	mux.Handle(
		"POST /pages",
		browserAuthn(editorAuthz(handler.SavePageForm(pageUseCases, draftUseCases, views))),
	)
	mux.Handle("POST /pages/{slug...}", browserAuthn(handler.FavoritePage(catalogUseCases, views)))
	mux.Handle("GET /revisions/{slug...}", browserAuthn(handler.RevisionHistory(catalogUseCases, views)))
	mux.Handle(
		"POST /revisions/{number}/restore/{slug...}",
		browserAuthn(editorAuthz(handler.RestoreRevision(pageUseCases, views))),
	)
	mux.Handle(
		"GET /pages/{slug...}",
		browserAuthn(handler.ViewPage(
			viewDataUseCases,
			catalogUseCases,
			settingsUseCases,
			knowledgeUseCases,
			renderer,
			views,
		)),
	)

	// API routes accept either bearer-token or browser authentication.
	mux.Handle("GET /api/pages", apiAuthn(handler.ListPages(catalogUseCases, logger)))
	mux.Handle("POST /api/pages", apiAuthn(editorAuthz(handler.SavePage(pageUseCases, logger))))
	mux.Handle(
		"POST /api/preview",
		apiAuthn(editorAuthz(handler.PreviewMarkdown(
			settingsUseCases,
			navigationUseCases,
			catalogUseCases,
			knowledgeUseCases,
			renderer,
			views,
			logger,
		))),
	)
	mux.Handle("GET /api/drafts/{key}", apiAuthn(editorAuthz(handler.GetPageDraft(draftUseCases, logger))))
	mux.Handle("PUT /api/drafts/{key}", apiAuthn(editorAuthz(handler.SavePageDraft(draftUseCases, logger))))
	mux.Handle("DELETE /api/drafts/{key}", apiAuthn(editorAuthz(handler.DeletePageDraft(draftUseCases, logger))))
	mux.Handle("GET /api/pages/{slug...}", apiAuthn(handler.GetPage(catalogUseCases, logger)))
	mux.Handle("PUT /api/pages/{slug...}", apiAuthn(editorAuthz(handler.SavePage(pageUseCases, logger))))
	mux.Handle("DELETE /api/pages/{slug...}", apiAuthn(adminAuthz(handler.DeletePage(pageUseCases, logger))))
	mux.Handle(
		"DELETE /api/admin/bin/{slug...}",
		apiAuthn(adminAuthz(handler.PermanentlyDeletePage(recycleBinUseCases, logger))),
	)
	mux.Handle("GET /api/search", apiAuthn(handler.SearchAPI(catalogUseCases, logger)))
	mux.Handle("GET /api/graph", apiAuthn(handler.KnowledgeGraphAPI(knowledgeUseCases, logger)))
	mux.Handle(
		"GET /api/editor/catalog",
		apiAuthn(editorAuthz(handler.EditorCatalog(navigationUseCases, knowledgeUseCases, catalogUseCases, logger))),
	)
	mux.Handle("GET /api/mentions/users", apiAuthn(handler.MentionUsers(userUseCases, logger)))
	mux.Handle("GET /api/notifications", apiAuthn(handler.NotificationsAPI(knowledgeUseCases, logger)))
	mux.Handle(
		"POST /api/notifications/{id}/read",
		apiAuthn(handler.MarkNotificationRead(knowledgeUseCases, logger)),
	)
	mux.Handle("GET /api/tags", apiAuthn(handler.Tags(catalogUseCases, logger)))
	mux.Handle("GET /api/groups", apiAuthn(handler.GroupsAPI(groupUseCases, logger)))
	mux.Handle("GET /api/images", apiAuthn(editorAuthz(handler.ListImages(mediaUseCases, logger))))
	mux.Handle("GET /api/attachments", apiAuthn(editorAuthz(handler.ListAttachments(mediaUseCases, logger))))
	mux.Handle("POST /api/attachments", apiAuthn(editorAuthz(handler.UploadAttachment(mediaUseCases, logger))))
	mux.Handle(
		"DELETE /api/attachments/{id}",
		apiAuthn(editorAuthz(handler.DeleteAttachment(mediaUseCases, logger))),
	)
	mux.Handle("POST /api/images", apiAuthn(editorAuthz(handler.UploadImage(mediaUseCases, logger))))
	mux.Handle("DELETE /api/images/{id}", apiAuthn(editorAuthz(handler.DeleteImage(mediaUseCases, logger))))
	mux.Handle("GET /api/recent", apiAuthn(handler.Recent(catalogUseCases, logger)))
	mux.Handle(
		"POST /api/admin/export",
		apiAuthn(adminAuthz(handler.ExportPages(catalogUseCases, navigationUseCases, mediaUseCases, logger))),
	)
}
