package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/icons"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/service"
	"golang.org/x/net/http/httpguts"
)

type renderingLanguageOption struct {
	Code  string
	Label string
}

var renderingLanguageOptions = []renderingLanguageOption{
	{Code: "en", Label: "English"},
	{Code: "en-US", Label: "English (United States)"},
	{Code: "en-GB", Label: "English (United Kingdom)"},
	{Code: "de", Label: "German"},
	{Code: "de-CH", Label: "German (Switzerland)"},
	{Code: "de-AT", Label: "German (Austria)"},
	{Code: "fr", Label: "French"},
	{Code: "fr-CH", Label: "French (Switzerland)"},
	{Code: "it", Label: "Italian"},
	{Code: "it-CH", Label: "Italian (Switzerland)"},
	{Code: "es", Label: "Spanish"},
	{Code: "nl", Label: "Dutch"},
	{Code: "pt", Label: "Portuguese"},
}

// Administration renders the administrator overview.
func Administration(
	viewDataUseCases viewDataService,
	administrationUseCases administrationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Administration", "overview")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		stats, err := administrationUseCases.Stats(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AdminStats = stats

		render(views, w, "admin", data)
	}
}

// AdminConfiguration renders runtime and application-wide configuration.
func AdminConfiguration(
	viewDataUseCases viewDataService,
	groupUseCases groupReader,
	userUseCases oidcIdentityService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Configuration", "configuration")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		groups, err := groupUseCases.Groups(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.Groups = groups
		data.ApplicationSettings.Authentication.OIDCGroupMappings, err = userUseCases.OIDCGroupMappings(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		render(views, w, "admin_configuration", data)
	}
}

// AdminRendering renders administrator-controlled Markdown rendering settings.
func AdminRendering(viewDataUseCases viewDataService, renderer *md.Renderer, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Rendering", "rendering")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		previews, err := renderingPreviews(renderer)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.RenderingPreviews = previews
		data.RenderingLanguages = renderingLanguageOptions

		render(views, w, "admin_rendering", data)
	}
}

// SaveAdminRendering updates administrator-controlled Markdown rendering settings.
func SaveAdminRendering(settingsUseCases settingsService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid rendering form.")
			return
		}

		settings := renderingSettingsFromForm(r)
		if !isRenderingLanguage(settings.ContentLanguage) {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Rendering validation failed.",
				httpresponse.NewFieldProblem("content_language", "Choose a supported content language."),
			)
			return
		}
		tableEnhancementsEnabled := settings.TableStyles || settings.TableSorting || settings.TableFiltering
		if !settings.Tables && tableEnhancementsEnabled {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Rendering validation failed.",
				httpresponse.NewFieldProblem(
					"tables",
					"Table colors, sorting, and filtering require Markdown tables to be enabled.",
				),
			)
			return
		}
		if err := settingsUseCases.SaveRenderingSettings(r.Context(), settings, admin.ID); err != nil {
			writeAdminProblem(logger, w, err, "Rendering settings")
			return
		}

		http.Redirect(w, r, "/admin/rendering", http.StatusSeeOther)
	}
}

// renderingSettingsFromForm parses mutable rendering settings from a form.
func renderingSettingsFromForm(r *http.Request) service.RenderingSettings {
	return service.RenderingSettings{
		WikiLinks:          r.FormValue("wiki_links") == "on",
		Callouts:           r.FormValue("callouts") == "on",
		Tabs:               r.FormValue("tabs") == "on",
		Details:            r.FormValue("details") == "on",
		Tables:             r.FormValue("tables") == "on",
		TableStyles:        r.FormValue("table_styles") == "on",
		TableSorting:       r.FormValue("table_sorting") == "on",
		TableFiltering:     r.FormValue("table_filtering") == "on",
		Strikethrough:      r.FormValue("strikethrough") == "on",
		TaskLists:          r.FormValue("task_lists") == "on",
		Autolinks:          r.FormValue("autolinks") == "on",
		SyntaxHighlighting: r.FormValue("syntax_highlighting") == "on",
		ContentLanguage:    strings.TrimSpace(r.FormValue("content_language")),
		CodingLigatures:    r.FormValue("coding_ligatures") == "on",
		Mermaid:            r.FormValue("mermaid") == "on",
		Footnotes:          r.FormValue("footnotes") == "on",
		DefinitionLists:    r.FormValue("definition_lists") == "on",
		Typographer:        r.FormValue("typographer") == "on",
	}
}

// isRenderingLanguage reports whether a configured content language is exposed by the admin UI.
func isRenderingLanguage(value string) bool {
	for _, option := range renderingLanguageOptions {
		if option.Code == value {
			return true
		}
	}
	return false
}

// AdminDocumentationHealth renders actionable wiki documentation-quality findings.
func AdminDocumentationHealth(
	viewDataUseCases viewDataService,
	administrationUseCases administrationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Documentation health", "health")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		health, err := administrationUseCases.DocumentationHealth(r.Context(), time.Now().AddDate(0, -6, 0))
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.DocumentationHealth = health

		render(views, w, "admin_health", data)
	}
}

// AdminAudit renders recent application audit events.
func AdminAudit(
	viewDataUseCases viewDataService,
	administrationUseCases administrationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Audit log", "audit")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		events, err := administrationUseCases.AuditEvents(r.Context(), 500)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AuditEvents = events

		render(views, w, "admin_audit", data)
	}
}

// AdminUsers renders user roles and group memberships.
func AdminUsers(
	viewDataUseCases viewDataService,
	userUseCases adminUserOverviewService,
	groupUseCases groupReader,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Users", "users")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		users, err := userUseCases.Users(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		groups, err := groupUseCases.Groups(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		identities, err := userUseCases.OIDCIdentities(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		identitiesByUser := make(map[int64][]service.OIDCIdentity)

		for _, identity := range identities {
			identitiesByUser[identity.UserID] = append(identitiesByUser[identity.UserID], identity)
		}
		for index := range users {
			users[index].OIDCIdentities = identitiesByUser[users[index].User.ID]
		}

		pendingIdentities, err := userUseCases.PendingOIDCIdentities(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		identityCount := len(identities)
		data.AdminUsers = users
		data.Groups = groups
		data.PendingOIDCIdentities = pendingIdentities
		data.OIDCIdentityCount = identityCount

		render(views, w, "admin_users", data)
	}
}

// AdminGroups renders group management.
func AdminGroups(
	viewDataUseCases viewDataService,
	groupUseCases groupReader,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Groups", "groups")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		groups, err := groupUseCases.Groups(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.Groups = groups

		render(views, w, "admin_groups", data)
	}
}

// AdminPageTemplates renders reusable page-template management.
func AdminPageTemplates(
	viewDataUseCases viewDataService,
	templateUseCases templateService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Page templates", "templates")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		templates, err := templateUseCases.PageTemplates(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.PageTemplates = templates

		render(views, w, "admin_templates", data)
	}
}

// CreateAdminPageTemplate creates a reusable Markdown page template.
func CreateAdminPageTemplate(templateUseCases templateService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid template form.")
			return
		}
		if _, err := templateUseCases.CreatePageTemplate(
			r.Context(),
			r.FormValue("name"),
			r.FormValue("description"),
			r.FormValue("markdown"),
		); err != nil {
			writeAdminProblem(logger, w, err, "Page template")
			return
		}

		http.Redirect(w, r, "/admin/templates", http.StatusSeeOther)
	}
}

// UpdateAdminPageTemplate updates one reusable Markdown page template.
func UpdateAdminPageTemplate(templateUseCases templateService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid template identifier.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid template form.")
			return
		}
		if err := templateUseCases.UpdatePageTemplate(
			r.Context(),
			id,
			r.FormValue("name"),
			r.FormValue("description"),
			r.FormValue("markdown"),
		); err != nil {
			writeAdminProblem(logger, w, err, "Page template")
			return
		}

		http.Redirect(w, r, "/admin/templates", http.StatusSeeOther)
	}
}

// DeleteAdminPageTemplate deletes one reusable page template.
func DeleteAdminPageTemplate(templateUseCases templateService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid template identifier.")
			return
		}
		if err := templateUseCases.DeletePageTemplate(r.Context(), id); err != nil {
			writeAdminProblem(logger, w, err, "Page template")
			return
		}

		http.Redirect(w, r, "/admin/templates", http.StatusSeeOther)
	}
}

// AdminNavigation renders icon configuration for every navigation path.
func AdminNavigation(
	viewDataUseCases viewDataService,
	navigationUseCases navigationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Navigation", "navigation")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		items, err := navigationUseCases.NavigationItems(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AdminNavigation = items

		render(views, w, "admin_navigation", data)
	}
}

// SearchIcons serves generated Lucide picker results to page editors and administrators.
func SearchIcons() http.HandlerFunc {
	type result struct {
		Name  string `json:"name"`
		Label string `json:"label"`
		SVG   string `json:"svg"`
	}
	type response struct {
		Items   []result `json:"items"`
		HasMore bool     `json:"has_more"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		offset := 0

		if value := r.URL.Query().Get("offset"); value != "" {
			var err error
			offset, err = strconv.Atoi(value)
			if err != nil || offset < 0 {
				httpresponse.Problem(w, http.StatusBadRequest, "Icon search offset must be a non-negative integer.")
				return
			}
		}

		options, hasMore := icons.SearchPage(r.URL.Query().Get("q"), offset, 80)
		results := make([]result, 0, len(options))

		for _, option := range options {
			results = append(
				results,
				result{Name: option.Name, Label: option.Label, SVG: string(icons.SVG(option.Name, 22))},
			)
		}

		httpresponse.Respond(w, http.StatusOK, response{Items: results, HasMore: hasMore})
	}
}

// SaveAdminNavigationIcon stores the selected icon for one navigation path.
func SaveAdminNavigationIcon(navigationUseCases navigationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid navigation form.")
			return
		}

		path := strings.TrimSpace(r.FormValue("path"))
		icon := strings.TrimSpace(r.FormValue("icon"))
		if !icons.IsNavigationIcon(icon) {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Navigation validation failed.",
				httpresponse.NewFieldProblem("icon", "Choose an icon from the available Lucide icons."),
			)
			return
		}
		if err := navigationUseCases.SetNavigationIcon(r.Context(), path, icon); err != nil {
			writeAdminProblem(logger, w, err, "Navigation path")
			return
		}

		http.Redirect(w, r, "/admin/navigation", http.StatusSeeOther)
	}
}

// AdminTags renders tag management and usage counts.
func AdminTags(
	viewDataUseCases viewDataService,
	administrationUseCases administrationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Tags", "tags")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		tags, err := administrationUseCases.TagInfos(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AdminTags = tags

		render(views, w, "admin_tags", data)
	}
}

// AdminTokens renders administrator-managed personal access tokens.
func AdminTokens(
	viewDataUseCases viewDataService,
	userUseCases userManagementService,
	tokenUseCases tokenService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Access tokens", "tokens")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		users, err := userUseCases.Users(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		tokens, err := tokenUseCases.Tokens(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AdminUsers = users
		data.AdminTokens = tokens

		render(views, w, "admin_tokens", data)
	}
}

// AdminExports renders page export controls.
func AdminExports(
	viewDataUseCases viewDataService,
	navigationUseCases navigationService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Exports", "exports")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		pages, err := navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.AdminPages = pages

		render(views, w, "admin_exports", data)
	}
}

// AdminImages renders all uploaded images and their reference counts.
func AdminImages(
	viewDataUseCases viewDataService,
	mediaUseCases imageListService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Images", "images")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		images, err := mediaUseCases.Images(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.Images = mediaItems(images)

		render(views, w, "admin_images", data)
	}
}

// SaveAdminAuthentication updates database-managed browser authentication settings.
func SaveAdminAuthentication(
	settingsUseCases settingsService,
	browserAuth auth.BrowserAuth,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid authentication form.")
			return
		}

		settings := authenticationSettingsFromForm(r)
		problems := authenticationSettingsProblems(settings, views.runtime)
		if len(problems) > 0 {
			httpresponse.Problem(w, http.StatusUnprocessableEntity, "Authentication validation failed.", problems...)
			return
		}

		if err := browserAuth.Validate(r.Context(), settings); err != nil {
			views.logger.Warn(
				"authentication settings rejected",
				"event", "authentication_settings_rejected",
				"mode", settings.Mode,
				"error", err,
			)

			field := "auth_mode"
			message := "The authentication configuration could not be verified."

			if settings.Mode == string(auth.AuthModeOIDC) {
				field = "oidc_issuer"
				message = "OIDC provider discovery failed. Check the issuer and server connectivity."
			}

			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Authentication validation failed.",
				httpresponse.NewFieldProblem(field, message),
			)
			return
		}

		if err := settingsUseCases.SaveAuthenticationSettings(r.Context(), settings, admin.ID); err != nil {
			writeAdminProblem(views.logger, w, err, "Authentication settings")
			return
		}

		http.Redirect(w, r, "/admin/configuration", http.StatusSeeOther)
	}
}

// authenticationSettingsFromForm parses non-secret browser authentication settings.
func authenticationSettingsFromForm(r *http.Request) service.AuthenticationSettings {
	mappings := make([]service.OIDCGroupMapping, 0, len(r.Form["oidc_group_source"]))

	for index, source := range r.Form["oidc_group_source"] {
		source = strings.TrimSpace(source)
		if source == "" || index >= len(r.Form["oidc_group_id"]) {
			continue
		}

		groupID, _ := strconv.ParseInt(strings.TrimSpace(r.Form["oidc_group_id"][index]), 10, 64)
		mappings = append(mappings, service.OIDCGroupMapping{OIDCGroup: source, GroupID: groupID})
	}

	return service.AuthenticationSettings{
		Mode:                      strings.TrimSpace(r.FormValue("auth_mode")),
		OIDCIssuer:                strings.TrimSpace(r.FormValue("oidc_issuer")),
		OIDCClientID:              strings.TrimSpace(r.FormValue("oidc_client_id")),
		OIDCGroupClaim:            strings.TrimSpace(r.FormValue("oidc_group_claim")),
		OIDCGroupSync:             r.FormValue("oidc_group_sync") == "on",
		OIDCGroupsAuthoritative:   r.FormValue("oidc_groups_authoritative") == "on",
		OIDCGroupMappings:         mappings,
		OIDCAdminGroup:            strings.TrimSpace(r.FormValue("oidc_admin_group")),
		TrustedUsernameHeaders:    splitHeaderNames(r.FormValue("trusted_username_headers")),
		TrustedEmailHeaders:       splitHeaderNames(r.FormValue("trusted_email_headers")),
		TrustedDisplayNameHeaders: splitHeaderNames(r.FormValue("trusted_display_name_headers")),
		TrustedGroupHeaders:       splitHeaderNames(r.FormValue("trusted_group_headers")),
		TrustedAdminGroup:         strings.TrimSpace(r.FormValue("trusted_admin_group")),
	}
}

// authenticationSettingsProblems returns field-level validation errors for browser authentication settings.
func authenticationSettingsProblems(
	settings service.AuthenticationSettings,
	runtime RuntimeInfo,
) []httpresponse.FieldProblem {
	var problems []httpresponse.FieldProblem

	switch auth.AuthMode(settings.Mode) {
	case auth.AuthModeNone:
	case auth.AuthModeLocal:
	case auth.AuthModeTrustedProxy:
		if len(settings.TrustedUsernameHeaders) == 0 {
			problems = append(problems, httpresponse.NewFieldProblem(
				"trusted_username_headers",
				"Configure at least one username header.",
			))
		}
		if settings.TrustedAdminGroup != "" && len(settings.TrustedGroupHeaders) == 0 {
			problems = append(problems, httpresponse.NewFieldProblem("trusted_group_headers", "Configure at least one group header for external administrator elevation."))
		}
	case auth.AuthModeOIDC:
		if settings.OIDCIssuer == "" {
			problems = append(problems, httpresponse.NewFieldProblem("oidc_issuer", "OIDC issuer is required."))
		}
		if settings.OIDCClientID == "" {
			problems = append(problems, httpresponse.NewFieldProblem("oidc_client_id", "OIDC client ID is required."))
		}
		if !runtime.OIDCClientSecretConfigured {
			problems = append(problems, httpresponse.NewFieldProblem(
				"oidc_client_secret",
				"Configure LORE__OIDC_CLIENT_SECRET before enabling OIDC.",
			))
		}
		if !runtime.SessionSecretConfigured {
			problems = append(problems, httpresponse.NewFieldProblem(
				"session_secret",
				"Configure LORE__SESSION_SECRET with at least 32 characters before enabling OIDC.",
			))
		}
		usesOIDCGroups := settings.OIDCGroupSync || settings.OIDCAdminGroup != ""
		if usesOIDCGroups && settings.OIDCGroupClaim == "" {
			problems = append(problems, httpresponse.NewFieldProblem(
				"oidc_group_claim",
				"Configure the OIDC claim containing group memberships.",
			))
		}

		seenMappings := map[string]bool{}

		for _, mapping := range settings.OIDCGroupMappings {
			if mapping.OIDCGroup == "" || mapping.GroupID <= 0 {
				problems = append(problems, httpresponse.NewFieldProblem(
					"oidc_group_mapping",
					"Choose a Lore group for every OIDC group mapping.",
				))
				break
			}
			if seenMappings[mapping.OIDCGroup] {
				problems = append(problems, httpresponse.NewFieldProblem(
					"oidc_group_mapping",
					"Each OIDC group may only be mapped once.",
				))
				break
			}

			seenMappings[mapping.OIDCGroup] = true
		}
	default:
		problems = append(problems, httpresponse.NewFieldProblem("auth_mode", "Choose a supported authentication mode."))
	}

	for field, headers := range map[string][]string{
		"trusted_username_headers":     settings.TrustedUsernameHeaders,
		"trusted_email_headers":        settings.TrustedEmailHeaders,
		"trusted_display_name_headers": settings.TrustedDisplayNameHeaders,
		"trusted_group_headers":        settings.TrustedGroupHeaders,
	} {
		for _, header := range headers {
			if !httpguts.ValidHeaderFieldName(header) {
				problems = append(problems, httpresponse.NewFieldProblem(field, "Use valid HTTP header names separated by commas."))
				break
			}
		}
	}

	return problems
}

// splitHeaderNames normalizes a comma-separated ordered header list.
func splitHeaderNames(value string) []string {
	seen := map[string]bool{}
	headers := make([]string, 0)

	for _, value := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' }) {
		header := strings.TrimSpace(value)
		key := strings.ToLower(header)
		if header == "" || seen[key] {
			continue
		}

		seen[key] = true
		headers = append(headers, header)
	}

	return headers
}

// SaveAdminSettings updates mutable application-wide settings.
func SaveAdminSettings(settingsUseCases settingsService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid settings form.")
			return
		}

		settings := applicationSettingsFromForm(r)
		if err := settingsUseCases.SaveApplicationSettings(r.Context(), settings, admin.ID); err != nil {
			writeAdminProblem(logger, w, err, "Settings")
			return
		}

		http.Redirect(w, r, "/admin/configuration", http.StatusSeeOther)
	}
}

// applicationSettingsFromForm parses mutable application-wide settings from a form.
func applicationSettingsFromForm(r *http.Request) service.ApplicationSettings {
	return service.ApplicationSettings{
		AllowUserRegistration: r.FormValue("allow_user_registration") == "on",
		DiscussionsEnabled:    r.FormValue("discussions_enabled") == "on",
	}
}

// isExternalAuthenticationMode reports whether browser identity is asserted outside Lore.
func isExternalAuthenticationMode(mode string) bool {
	switch auth.AuthMode(mode) {
	case auth.AuthModeOIDC, auth.AuthModeTrustedProxy:
		return true
	default:
		return false
	}
}

// UpdateAdminUser updates one user's role, group memberships, and optional recovery login state.
func UpdateAdminUser(
	userUseCases userManagementService,
	settingsUseCases settingsService,
	local *auth.Local,
	views *Views,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"User validation failed.",
				httpresponse.NewFieldProblem("user_id", "Choose a valid user."),
			)
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid user form.")
			return
		}

		role := r.FormValue("role")
		if !service.ValidUserRole(role) {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid user role.")
			return
		}
		if userID == admin.ID && role != "admin" {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"User validation failed.",
				httpresponse.NewFieldProblem("role", "You cannot remove your own administrator role."),
			)
			return
		}

		enabled := r.FormValue("account_enabled") == "on"
		if userID == admin.ID && !enabled {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"User validation failed.",
				httpresponse.NewFieldProblem("account_enabled", "You cannot disable your own account."),
			)
			return
		}

		groupIDs := make([]int64, 0, len(r.Form["group_id"]))

		for _, value := range r.Form["group_id"] {
			groupID, err := strconv.ParseInt(value, 10, 64)
			if err != nil || groupID <= 0 {
				httpresponse.Problem(w,
					http.StatusBadRequest,
					"Group validation failed.",
					httpresponse.NewFieldProblem("group_id", "Choose a valid group."),
				)
				return
			}

			groupIDs = append(groupIDs, groupID)
		}

		password := r.FormValue("local_password")
		updateLocalCredential := r.FormValue("update_local_credential") == "true"
		if problems := localPasswordValidationProblems(
			password,
			r.FormValue("local_password_confirm"),
			"local_password",
			"local_password_confirm",
			false,
		); len(problems) > 0 {
			httpresponse.Problem(w, http.StatusUnprocessableEntity, "Local login validation failed.", problems...)
			return
		}

		var localCredentialEnabled *bool

		if updateLocalCredential {
			settings, err := settingsUseCases.ApplicationSettings(r.Context())
			if err != nil {
				writeUnexpectedProblem(logger, w, err)
				return
			}

			effectiveMode := settings.Authentication.Mode

			if views.runtime.AuthModeOverride != "" {
				effectiveMode = views.runtime.AuthModeOverride
			}

			if !isExternalAuthenticationMode(effectiveMode) {
				httpresponse.Problem(w,
					http.StatusBadRequest,
					"Local login state cannot be changed.",
					httpresponse.NewFieldProblem("local_credential_enabled", "Local recovery credentials can only be enabled or disabled while external authentication is active."),
				)
				return
			}

			enabled := password != "" || r.FormValue("local_credential_enabled") == "on"
			localCredentialEnabled = &enabled
		}

		if err := userUseCases.UpdateUser(r.Context(), userID, role, enabled, groupIDs, localCredentialEnabled); err != nil {
			writeAdminProblem(logger, w, err, "User")
			return
		}

		if password != "" {
			if err := local.SetPassword(r.Context(), userID, password); err != nil {
				writeAdminProblem(logger, w, err, "Local credential")
				return
			}
		}

		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	}
}

// RevokeAdminUserSessions signs an account out of local and OIDC browser sessions.
func RevokeAdminUserSessions(userUseCases userManagementService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid user.")
			return
		}
		if err := userUseCases.RevokeUserSessions(r.Context(), userID, admin.ID); err != nil {
			writeAdminProblem(logger, w, err, "User sessions")
			return
		}

		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	}
}

// ApprovePendingOIDCIdentity creates a Lore account for one verified identity request.
func ApprovePendingOIDCIdentity(userUseCases oidcIdentityService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		pendingID, err := pendingOIDCIdentityID(r)
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid identity request.")
			return
		}

		_, err = userUseCases.ApprovePendingOIDCIdentity(r.Context(), pendingID, admin.ID)
		if err != nil {
			writeAdminProblem(logger, w, err, "Identity request")
			return
		}

		http.Redirect(w, r, "/admin/users#pending-identities", http.StatusSeeOther)
	}
}

// LinkPendingOIDCIdentity replaces an existing user's issuer binding with a verified identity request.
func LinkPendingOIDCIdentity(userUseCases oidcIdentityService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		pendingID, err := pendingOIDCIdentityID(r)
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid identity request.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid identity link form.")
			return
		}

		userID, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Identity link validation failed.",
				httpresponse.NewFieldProblem("user_id", "Choose an existing Lore user."),
			)
			return
		}

		_, err = userUseCases.LinkPendingOIDCIdentity(r.Context(), pendingID, userID, admin.ID)
		if err != nil {
			writeAdminProblem(logger, w, err, "Identity request")
			return
		}

		http.Redirect(w, r, "/admin/users#pending-identities", http.StatusSeeOther)
	}
}

// RejectPendingOIDCIdentity blocks one verified identity request until an administrator reopens it.
func RejectPendingOIDCIdentity(userUseCases oidcIdentityService, logger *slog.Logger) http.HandlerFunc {
	return pendingOIDCIdentityStatusHandler(userUseCases, logger, true)
}

// ReopenPendingOIDCIdentity returns a rejected identity request to the pending queue.
func ReopenPendingOIDCIdentity(userUseCases oidcIdentityService, logger *slog.Logger) http.HandlerFunc {
	return pendingOIDCIdentityStatusHandler(userUseCases, logger, false)
}

// pendingOIDCIdentityStatusHandler updates one administrator decision.
func pendingOIDCIdentityStatusHandler(
	userUseCases oidcIdentityService,
	logger *slog.Logger,
	rejected bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		pendingID, err := pendingOIDCIdentityID(r)
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid identity request.")
			return
		}
		if err := userUseCases.SetPendingOIDCIdentityRejected(
			r.Context(),
			pendingID,
			rejected,
			admin.ID,
		); err != nil {
			writeAdminProblem(logger, w, err, "Identity request")
			return
		}

		http.Redirect(w, r, "/admin/users#pending-identities", http.StatusSeeOther)
	}
}

// pendingOIDCIdentityID parses the pending identity identifier from the route.
func pendingOIDCIdentityID(r *http.Request) (pendingID int64, err error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid pending OIDC identity")
	}

	return id, nil
}

// RemoveAdminOIDCIdentity disconnects one active OIDC binding from a Lore account.
func RemoveAdminOIDCIdentity(userUseCases oidcIdentityService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)
		userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid user identifier.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid identity form.")
			return
		}

		issuer := strings.TrimSpace(r.FormValue("issuer"))
		subject := strings.TrimSpace(r.FormValue("subject"))
		if issuer == "" || subject == "" {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid OIDC identity.")
			return
		}
		if err := userUseCases.RemoveOIDCIdentity(
			r.Context(),
			userID,
			issuer,
			subject,
			admin.ID,
		); err != nil {
			writeAdminProblem(logger, w, err, "OIDC identity")
			return
		}

		http.Redirect(w, r, "/admin/users#oidc-identities", http.StatusSeeOther)
	}
}

// CreateAdminGroup creates a new user group.
func CreateAdminGroup(groupUseCases groupWriter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid group form.")
			return
		}
		if _, err := groupUseCases.CreateGroup(r.Context(), r.FormValue("name")); err != nil {
			writeAdminProblem(logger, w, err, "Group")
			return
		}

		http.Redirect(w, r, "/admin/groups", http.StatusSeeOther)
	}
}

// DeleteAdminGroup deletes one user group.
func DeleteAdminGroup(groupUseCases groupWriter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Group validation failed.",
				httpresponse.NewFieldProblem("group_id", "Choose a valid group."),
			)
			return
		}
		if err := groupUseCases.DeleteGroup(r.Context(), id); err != nil {
			writeAdminProblem(logger, w, err, "Group")
			return
		}

		http.Redirect(w, r, "/admin/groups", http.StatusSeeOther)
	}
}

// DeleteAdminTag removes a tag and all page associations for it.
func DeleteAdminTag(administrationUseCases administrationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid tag identifier.")
			return
		}
		if err := administrationUseCases.DeleteTag(r.Context(), id); err != nil {
			writeAdminProblem(logger, w, err, "Tag")
			return
		}

		http.Redirect(w, r, "/admin/tags", http.StatusSeeOther)
	}
}

// administrationData builds common view data for administrator-only pages.
func administrationData(
	r *http.Request,
	viewDataUseCases viewDataService,
	views *Views,
	title, section string,
) (ViewData, error) {
	data, err := viewData(r, viewDataUseCases, views, title)
	if err != nil {
		return ViewData{}, err
	}

	data.AdminSection = section
	data.Navigation = nil

	return data, nil
}

// hasGroup reports whether a group name appears in a user's group list.
func hasGroup(groups []string, name string) bool {
	for _, group := range groups {
		if strings.EqualFold(group, name) {
			return true
		}
	}
	return false
}

// hasGroupID reports whether a group identifier appears in a page group list.
func hasGroupID(groups []service.Group, id int64) bool {
	for _, group := range groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

// writeAdminProblem translates expected administration errors into HTTP problems.
func writeAdminProblem(logger *slog.Logger, w http.ResponseWriter, err error, object string) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		httpresponse.Problem(w, http.StatusNotFound, object+" not found.")
	case errors.Is(err, service.ErrAlreadyExists):
		httpresponse.Problem(w, http.StatusConflict, object+" already exists.")
	case errors.Is(err, service.ErrForbidden):
		httpresponse.Problem(w, http.StatusForbidden, object+" cannot be changed in its current state.")
	default:
		writeUnexpectedProblem(logger, w, err)
	}
}

// AdminBin renders pages that have been moved to the recycle bin.
func AdminBin(
	viewDataUseCases viewDataService,
	recycleBinUseCases recycleBinService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Recycle bin", "bin")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		pages, err := recycleBinUseCases.DeletedPages(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.DeletedPages = pages

		render(views, w, "admin_bin", data)
	}
}

// RestoreAdminPage restores one page from the recycle bin.
func RestoreAdminPage(recycleBinUseCases recycleBinService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"A page path is required.",
				httpresponse.NewFieldProblem("slug", "Choose a page to restore."),
			)
			return
		}
		if err := recycleBinUseCases.RestorePage(r.Context(), slug); err != nil {
			writeAdminProblem(logger, w, err, "Page")
			return
		}

		http.Redirect(w, r, "/admin/bin", http.StatusSeeOther)
	}
}

// PermanentlyDeleteAdminPage removes one page from the recycle bin permanently.
func PermanentlyDeleteAdminPage(recycleBinUseCases recycleBinService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"A page path is required.",
				httpresponse.NewFieldProblem("slug", "Choose a page to delete permanently."),
			)
			return
		}
		if err := recycleBinUseCases.PermanentlyDeletePage(r.Context(), slug); err != nil {
			writeAdminProblem(logger, w, err, "Page")
			return
		}

		http.Redirect(w, r, "/admin/bin", http.StatusSeeOther)
	}
}

// SearchAdminUsers returns user matches for live administrator pickers.
func SearchAdminUsers(userUseCases userDirectoryService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(query)) < 2 {
			httpresponse.Respond(w, http.StatusOK, []service.User{})
			return
		}

		users, err := userUseCases.SearchUsers(r.Context(), query, 20)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, users)
	}
}

// AdminGroupMembers returns the current members of one group.
func AdminGroupMembers(groupUseCases groupReader, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || groupID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid group identifier.",
				httpresponse.NewFieldProblem("group_id", "Choose a valid group."),
			)
			return
		}

		members, err := groupUseCases.GroupMembers(r.Context(), groupID)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, members)
	}
}

type groupMemberRequest struct {
	UserID int64 `json:"user_id"`
}

// AddAdminGroupMember assigns one user to a group.
func AddAdminGroupMember(
	groupUseCases groupWriter,
	userUseCases userDirectoryService,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || groupID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid group identifier.",
				httpresponse.NewFieldProblem("group_id", "Choose a valid group."),
			)
			return
		}

		request, err := decode[groupMemberRequest](w, r)
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid member request.",
				httpresponse.NewFieldProblem("request", err.Error()),
			)
			return
		}
		if request.UserID <= 0 {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"A user is required.",
				httpresponse.NewFieldProblem("user_id", "Choose a person from the suggestions."),
			)
			return
		}
		if err := groupUseCases.AddGroupMember(r.Context(), groupID, request.UserID); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		user, err := userUseCases.User(r.Context(), request.UserID)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusCreated, user)
	}
}

// RemoveAdminGroupMember removes one user from a group.
func RemoveAdminGroupMember(groupUseCases groupWriter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || groupID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid group identifier.",
				httpresponse.NewFieldProblem("group_id", "Choose a valid group."),
			)
			return
		}

		userID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid user identifier.",
				httpresponse.NewFieldProblem("user_id", "Choose a valid person."),
			)
			return
		}
		if err := groupUseCases.RemoveGroupMember(r.Context(), groupID, userID); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
