package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/icons"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/revision"
	"github.com/gi8lino/lore/internal/service"
	"github.com/gi8lino/lore/themes"
)

var sharedTemplateFiles = []string{
	"templates/layout.gohtml",
	"templates/public_layout.gohtml",
	"templates/header.gohtml",
	"templates/sidebar.gohtml",
	"templates/admin_sidebar.gohtml",
	"templates/account_menu.gohtml",
	"templates/dialogs.gohtml",
	"templates/navigation.gohtml",
	"templates/lists.gohtml",
	"templates/page_contents.gohtml",
	"templates/revisions.gohtml",
}

var pageTemplateNames = []string{
	"login",
	"setup",
	"shared_page",
	"home",
	"page",
	"edit",
	"search",
	"settings",
	"admin",
	"admin_configuration",
	"admin_rendering",
	"admin_health",
	"admin_templates",
	"admin_audit",
	"admin_snippets",
	"admin_pages",
	"admin_import",
	"graph",
	"admin_users",
	"admin_groups",
	"admin_tags",
	"admin_tokens",
	"admin_exports",
	"admin_images",
	"admin_navigation",
	"admin_bin",
}

// publicViewData builds shared data for unauthenticated setup and login pages.
func publicViewData(views *Views, title string) (ViewData, error) {
	preferences := service.DefaultUserPreferences()
	activeTheme := themes.DefaultTheme
	preferences.Theme = activeTheme
	themeData, err := json.Marshal(views.themes)
	if err != nil {
		return ViewData{}, err
	}

	return ViewData{
		Title:        title,
		Preferences:  preferences,
		Version:      views.version,
		AssetVersion: views.assetVersion,
		Commit:       views.commit,
		Runtime:      views.runtime,
		ThemeData:    template.JS(themeData),
		Themes:       views.themes,
		ActiveTheme:  activeTheme,
	}, nil
}

// RuntimeInfo contains non-secret runtime configuration safe to show to administrators.
type RuntimeInfo struct {
	// ListenAddress is the configured HTTP listen address.
	ListenAddress string
	// PublicURL is the externally visible wiki URL.
	PublicURL string
	// PDFURL is the optional deployment-level PDF endpoint override.
	PDFURL string
	// AuthModeOverride is the optional deployment-level recovery override.
	AuthModeOverride string
	// OIDCClientSecretConfigured reports whether the OIDC client secret is available.
	OIDCClientSecretConfigured bool
	// SessionSecretConfigured reports whether a valid OIDC session secret is available.
	SessionSecretConfigured bool
	// LocalLoginEnabled reports whether the deployment exposes break-glass local login.
	LocalLoginEnabled bool
	// ThemeDirectory is the optional external theme directory.
	ThemeDirectory string
}

// Views contains the shared server-rendered HTML dependencies.
type Views struct {
	// templates maps page names to parsed template sets with the shared layout and partials.
	templates map[string]*template.Template
	// logger records template rendering failures.
	logger *slog.Logger
	// version is the application version exposed in rendered pages.
	version string
	// commit is the application commit exposed for diagnostics.
	commit string
	// themes contains the themes available to the browser.
	themes []themes.Theme
	// runtime contains non-secret runtime configuration shown to administrators.
	runtime RuntimeInfo
	// assetVersion fingerprints embedded browser assets for cache-safe URLs.
	assetVersion string
}

// ViewDataLoader assembles the shared data required by authenticated HTML views.
type ViewDataLoader struct {
	preferenceUseCases preferenceService
	navigationUseCases navigationService
	catalogUseCases    sidebarCatalogService
	settingsUseCases   settingsService
	knowledgeUseCases  knowledgeSidebarService
}

// NewViewDataLoader constructs the shared authenticated view-data loader.
func NewViewDataLoader(
	preferences preferenceService,
	navigation navigationService,
	catalog sidebarCatalogService,
	settings settingsService,
	knowledge knowledgeSidebarService,
) *ViewDataLoader {
	return &ViewDataLoader{
		preferenceUseCases: preferences,
		navigationUseCases: navigation,
		catalogUseCases:    catalog,
		settingsUseCases:   settings,
		knowledgeUseCases:  knowledge,
	}
}

// ViewData contains the data shared by server-rendered wiki templates.
type ViewData struct {
	// Title is the page title displayed in the browser chrome.
	Title string
	// User is the authenticated user rendering the page.
	User service.User
	// Preferences contains the current user's presentation preferences.
	Preferences service.UserPreferences
	// Page is the current wiki page when one is being viewed or edited.
	Page *service.Page
	// PageFavorite reports whether the current user has pinned the current page.
	PageFavorite bool
	// HTML is the sanitized rendered Markdown for the current page.
	HTML template.HTML
	// PageContents contains heading links for the current rendered page.
	PageContents []markdown.Heading
	// Subpages contains the generated navigation subtree below the current page.
	Subpages []navigation.Node
	// Pages contains the primary page collection for the current view.
	Pages []service.Page
	// Favorites contains the current user's favorite pages.
	Favorites []service.Page
	// SidebarPinned contains favorite pages shown above the navigation tree.
	SidebarPinned []service.Page
	// SidebarRecent contains recently viewed pages shown above the navigation tree.
	SidebarRecent []service.Page
	// Recent contains recently changed pages.
	Recent []service.Page
	// Popular contains the most viewed pages.
	Popular []service.Page
	// RecentEdits contains pages the current user recently changed.
	RecentEdits []service.RecentEdit
	// Drafts contains the current user's private server-side page drafts.
	Drafts []service.PageDraft
	// SavedSearches contains named smart collections for the current user.
	SavedSearches []service.SavedSearch
	// Notifications contains recent inbox items for the current user.
	Notifications []service.Notification
	// UnreadNotifications is the current unread inbox count.
	UnreadNotifications int
	// Backlinks contains pages linking to the current page.
	Backlinks []service.Page
	// OutgoingLinks contains wiki links from the current page.
	OutgoingLinks []service.PageLink
	// BrokenLinks contains outgoing wiki links with no current target.
	BrokenLinks []service.PageLink
	// Comments contains anchored discussion items for the current page.
	Comments []service.PageComment
	// Related contains pages related to the current page by tag.
	Related []service.Page
	// LatestRevision is the newest revision shown in the page summary.
	LatestRevision *revision.Revision
	// RevisionCount is the total number of revisions for the current page.
	RevisionCount int
	// Revisions contains revision history rendered in the on-demand history dialog.
	Revisions []revision.Revision
	// RevisionSlug is the page path used by revision history actions.
	RevisionSlug string
	// Images contains uploaded media shown in settings or administration.
	Images []MediaItem
	// UserTokens contains personal access tokens owned by the current user.
	UserTokens []service.APIToken
	// AdminSection identifies the active administration navigation section.
	AdminSection string
	// AdminStats contains high-level persisted object counts for administrators.
	AdminStats service.AdminStats
	// ApplicationSettings contains mutable application-wide settings for administrators.
	ApplicationSettings service.ApplicationSettings
	// RenderingPreviews contains sanitized examples for administrator rendering controls.
	RenderingPreviews map[string]template.HTML
	// DocumentationHealth contains actionable wiki quality findings.
	DocumentationHealth service.DocumentationHealth
	// RenderingLanguages lists content languages available to administrators.
	RenderingLanguages []renderingLanguageOption
	// PageContentLanguage is the effective language for the current page/editor.
	PageContentLanguage string
	// AdminUsers contains users and group memberships for administrators.
	AdminUsers []service.AdminUser
	// PendingOIDCIdentities contains verified OIDC identities awaiting an administrator decision.
	PendingOIDCIdentities []service.PendingOIDCIdentity
	// OIDCIdentityCount is the number of active external OIDC bindings.
	OIDCIdentityCount int
	// Groups contains administratively managed user groups.
	Groups []service.Group
	// PageTemplates contains reusable templates available to page authors.
	PageTemplates []service.PageTemplate
	// KnowledgeSnippets contains reusable variables and Markdown snippets.
	KnowledgeSnippets []service.KnowledgeSnippet
	// PageStatuses contains lifecycle statuses available to page editors.
	PageStatuses []string
	// EditorTemplate is the selected template used to prefill a new page.
	EditorTemplate *service.PageTemplate
	// EditorInitialSlug pre-fills a requested path for a new page.
	EditorInitialSlug string
	// AdminTags contains tags and page usage counts for administrators.
	AdminTags []service.TagInfo
	// AdminTokens contains all personal access tokens for administrators.
	AdminTokens []service.APIToken
	// AuditEvents contains recent administrative audit events.
	AuditEvents []service.AuditEvent
	// AdminPages contains all pages available for administrative export.
	AdminPages []service.Page
	// DeletedPages contains pages currently held in the recycle bin.
	DeletedPages []service.DeletedPage
	// AdminNavigation contains top-level navigation sections and their persisted icons.
	AdminNavigation []service.NavigationItem
	// Tags contains tags exposed by the current view.
	Tags []string
	// Query is the active search query.
	Query string
	// AuthError contains a browser-facing authentication or setup validation error.
	AuthError string
	// AuthNext is the validated local path restored after interactive authentication.
	AuthNext string
	// LocalCredentialAuthenticated reports whether this request used a local session.
	LocalCredentialAuthenticated bool
	// Navigation is the slug-derived sidebar navigation tree.
	Navigation []navigation.Node
	// Version is the application version shown in the footer.
	Version string
	// AssetVersion fingerprints embedded browser assets for cache-safe URLs.
	AssetVersion string
	// Commit is the application commit shown in administration.
	Commit string
	// Runtime contains non-secret runtime configuration for administrators.
	Runtime RuntimeInfo
	// ThemeData is the JSON theme catalog consumed by the browser.
	ThemeData template.JS
	// Themes lists the theme titles available in settings.
	Themes []themes.Theme
	// ActiveTheme is the current user's validated theme title.
	ActiveTheme string
	// CanEdit reports whether the current user may create or edit pages.
	CanEdit bool
	// RenderMermaid reports whether browser-side Mermaid rendering is enabled.
	RenderMermaid bool
}

// NewViews parses each page template with the shared layout and partials once at startup.
func NewViews(
	appFS fs.FS,
	logger *slog.Logger,
	version, commit string,
	availableThemes []themes.Theme,
	runtime RuntimeInfo,
) (*Views, error) {
	assetVersion, err := fingerprintAssets(appFS)
	if err != nil {
		return nil, fmt.Errorf("fingerprint web assets: %w", err)
	}

	logoSVG, err := fs.ReadFile(appFS, "lore.svg")
	if err != nil {
		return nil, fmt.Errorf("read Lore logo: %w", err)
	}

	funcs := template.FuncMap{
		"join":       strings.Join,
		"timeago":    timeAgo,
		"filesize":   fileSize,
		"hasgroup":   hasGroup,
		"hasgroupid": hasGroupID,
		"icon":       icons.SVG,
		"logo": func() template.HTML {
			return template.HTML(logoSVG)
		},
	}
	templates := make(map[string]*template.Template, len(pageTemplateNames))

	for _, name := range pageTemplateNames {
		files := make([]string, 0, len(sharedTemplateFiles)+1)
		files = append(files, sharedTemplateFiles...)
		files = append(files, "templates/"+name+".gohtml")

		parsed, err := template.New(name).Funcs(funcs).ParseFS(appFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", name, err)
		}

		templates[name] = parsed
	}

	return &Views{
		templates:    templates,
		logger:       logger,
		version:      version,
		commit:       commit,
		themes:       availableThemes,
		runtime:      runtime,
		assetVersion: assetVersion,
	}, nil
}

// fingerprintAssets returns a stable short hash for the complete embedded web filesystem.
func fingerprintAssets(appFS fs.FS) (string, error) {
	hash := sha256.New()
	err := fs.WalkDir(appFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(appFS, path)
		if err != nil {
			return err
		}

		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})

		return nil
	})
	if err != nil {
		return "", err
	}

	sum := hash.Sum(nil)

	return hex.EncodeToString(sum[:8]), nil
}

// Load builds the common template data used by every browser page.
func (l *ViewDataLoader) Load(r *http.Request, views *Views, title string) (ViewData, error) {
	user, _ := auth.User(r)

	preferences, err := l.preferenceUseCases.Preferences(r.Context(), user.ID)
	if err != nil {
		return ViewData{}, err
	}

	var pageNavigation []navigation.Node
	var sidebarPinned []service.Page
	var sidebarRecent []service.Page

	if !strings.HasPrefix(r.URL.Path, "/admin") {
		pages, err := l.navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			return ViewData{}, err
		}

		navigationIcons, err := l.navigationUseCases.NavigationIcons(r.Context())
		if err != nil {
			return ViewData{}, err
		}

		expanded := preferences.ExpandedNavigation

		if !preferences.RememberNavigationState {
			expanded = nil
		}

		navigationPages := make([]navigation.Page, 0, len(pages))

		for _, page := range pages {
			navigationPages = append(navigationPages, navigation.Page{
				Slug:  page.Slug,
				Title: page.Title,
				Icon:  page.Icon,
			})
		}

		pageNavigation = navigation.Build(navigationPages, navigation.Options{
			ActiveSlug:     activeNavigationSlug(r.URL.Path),
			Expanded:       expanded,
			ShowPageCounts: preferences.ShowNavigationPageCounts,
			Icons:          navigationIcons,
		})

		if preferences.ShowPinnedPages {
			sidebarPinned, err = l.catalogUseCases.Favorites(r.Context(), user.ID)
			if err != nil {
				return ViewData{}, err
			}
		}
		if preferences.ShowRecentlyViewed {
			sidebarRecent, err = l.catalogUseCases.RecentViewed(r.Context(), user.ID, 8)
			if err != nil {
				return ViewData{}, err
			}

			sidebarRecent = pagesWithout(sidebarRecent, sidebarPinned, 5)
		}
	}

	applicationSettings, err := l.settingsUseCases.ApplicationSettings(r.Context())
	if err != nil {
		return ViewData{}, err
	}

	activeTheme := themes.DefaultTheme

	if selected, ok := themes.Find(views.themes, preferences.Theme); ok {
		activeTheme = selected.Title
	}

	preferences.Theme = activeTheme

	themeData, err := json.Marshal(views.themes)
	if err != nil {
		return ViewData{}, err
	}

	savedSearches, err := l.knowledgeUseCases.SavedSearches(r.Context(), user.ID)
	if err != nil {
		return ViewData{}, err
	}

	notifications, unreadNotifications, err := l.knowledgeUseCases.Notifications(r.Context(), user.ID, 8)
	if err != nil {
		return ViewData{}, err
	}

	return ViewData{
		Title:               title,
		User:                user,
		Preferences:         preferences,
		Navigation:          pageNavigation,
		SidebarPinned:       sidebarPinned,
		SidebarRecent:       sidebarRecent,
		SavedSearches:       savedSearches,
		Notifications:       notifications,
		UnreadNotifications: unreadNotifications,
		PageStatuses:        service.PageStatuses(),
		Version:             views.version,
		AssetVersion:        views.assetVersion,
		Commit:              views.commit,
		Runtime:             views.runtime,
		ThemeData:           template.JS(themeData),
		Themes:              views.themes,
		ActiveTheme:         activeTheme,
		ApplicationSettings: applicationSettings,
		CanEdit:             user.Role == "admin" || user.Role == "editor",
		RenderMermaid:       applicationSettings.Rendering.Mermaid,
		PageContentLanguage: applicationSettings.Rendering.ContentLanguage,
	}, nil
}

// viewData loads shared authenticated view data through the handler's narrow dependency.
func viewData(r *http.Request, loader viewDataService, views *Views, title string) (ViewData, error) {
	return loader.Load(r, views, title)
}

// pagesWithout returns up to limit pages excluding any page present in excluded.
func pagesWithout(pages, excluded []service.Page, limit int) []service.Page {
	excludedIDs := make(map[int64]bool, len(excluded))

	for _, page := range excluded {
		excludedIDs[page.ID] = true
	}

	result := make([]service.Page, 0, min(limit, len(pages)))

	for _, page := range pages {
		if excludedIDs[page.ID] {
			continue
		}

		result = append(result, page)
		if len(result) == limit {
			break
		}
	}

	return result
}

// activeNavigationSlug extracts the current page slug from browser page and editor routes.
func activeNavigationSlug(requestPath string) string {
	for _, prefix := range []string{"/pages/", "/edit/"} {
		if strings.HasPrefix(requestPath, prefix) {
			return strings.Trim(strings.TrimPrefix(requestPath, prefix), "/")
		}
	}
	return ""
}

// render executes a page layout into a buffer before writing the HTTP response.
func render(views *Views, w http.ResponseWriter, page string, data ViewData) {
	renderTemplate(views, w, page, "layout", data)
}

// renderPublic executes the minimal unauthenticated page layout.
func renderPublic(views *Views, w http.ResponseWriter, page string, data ViewData) {
	renderTemplate(views, w, page, "public-layout", data)
}

// renderFragment executes one named fragment from a parsed page template set.
func renderFragment(views *Views, w http.ResponseWriter, page, name string, data ViewData) {
	renderTemplate(views, w, page, name, data)
}

// renderTemplate executes a named template into a buffer before writing the HTTP response.
func renderTemplate(views *Views, w http.ResponseWriter, page, name string, data ViewData) {
	pageTemplate, ok := views.templates[page]
	if !ok {
		views.logger.Error("render template", "event", "template_missing", "page", page, "template", name)
		httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
		return
	}

	var output bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&output, name, data); err != nil {
		views.logger.Error(
			"render template",
			"event",
			"template_render_failed",
			"page",
			page,
			"template",
			name,
			"error",
			err,
		)
		httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if _, err := w.Write(output.Bytes()); err != nil {
		views.logger.Error(
			"write template response",
			"event",
			"template_write_failed",
			"page",
			page,
			"template",
			name,
			"error",
			err,
		)
	}
}

// renderTemplateHTML renders a trusted template fragment for insertion into rendered Markdown.
func renderTemplateHTML(views *Views, page, name string, data ViewData) (template.HTML, error) {
	pageTemplate, ok := views.templates[page]
	if !ok {
		return "", fmt.Errorf("page template %q not found", page)
	}

	var output bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&output, name, data); err != nil {
		return "", fmt.Errorf("render %s template %s: %w", page, name, err)
	}

	return template.HTML(output.String()), nil
}

// timeAgo formats recent timestamps as compact relative ages.
func timeAgo(value time.Time) string {
	duration := time.Since(value)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return strconv.Itoa(int(duration.Minutes())) + "m ago"
	case duration < 24*time.Hour:
		return strconv.Itoa(int(duration.Hours())) + "h ago"
	default:
		return value.Format("2006-01-02")
	}
}

// fileSize formats byte counts for compact template display.
func fileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}

	divisor := int64(unit)
	exponent := 0

	for value := size / unit; value >= unit && exponent < 3; value /= unit {
		divisor *= unit
		exponent++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGT"[exponent])
}
