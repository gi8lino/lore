package handler

import (
	"html/template"
	"slices"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/revision"
	"github.com/gi8lino/lore/themes"
)

// ViewData contains the data shared by server-rendered Lore templates.
type ViewData struct {
	// Title is the page title displayed in the browser chrome.
	Title string
	// User is the authenticated user rendering the page.
	User domain.User
	// Preferences contains the current user's presentation preferences.
	Preferences domain.UserPreferences
	// TypographySize is the effective content typography preset after applying the application default.
	TypographySize string
	// Page is the current page when one is being viewed or edited.
	Page *domain.Page
	// PageFavorite reports whether the current user has pinned the current page.
	PageFavorite bool
	// PageWatchScope is page or subtree when the current user watches this path.
	PageWatchScope string
	// PageReviewRequest is the active lightweight approval workflow item.
	PageReviewRequest domain.PageReviewRequest
	// CanReviewPage reports whether the current user may decide the pending review.
	CanReviewPage bool
	// CanManageReview reports whether the current user may edit or cancel the pending request.
	CanManageReview bool
	// ReviewGroups contains collaboration groups available as review targets.
	ReviewGroups []domain.Group
	// HTML is the sanitized rendered Markdown for the current page.
	HTML template.HTML
	// PageContents contains heading links for the current rendered page.
	PageContents []markdown.Heading
	// Pages contains the primary page collection for the current view.
	Pages []domain.Page
	// Favorites contains the current user's favorite pages.
	Favorites []domain.Page
	// SidebarPinned contains favorite pages shown above the navigation tree.
	SidebarPinned []domain.Page
	// SidebarRecent contains recently viewed pages shown above the navigation tree.
	SidebarRecent []domain.Page
	// Recent contains recently changed pages.
	Recent []domain.Page
	// Popular contains the most viewed pages.
	Popular []domain.Page
	// RecentEdits contains pages the current user recently changed.
	RecentEdits []domain.RecentEdit
	// Drafts contains the current user's private server-side page drafts.
	Drafts []domain.PageDraft
	// SavedSearches contains named smart collections for the current user.
	SavedSearches []domain.SavedSearch
	// Notifications contains recent inbox items for the current user.
	Notifications []domain.Notification
	// UnreadNotifications is the current unread inbox count.
	UnreadNotifications int
	// Backlinks contains pages linking to the current page.
	Backlinks []domain.Page
	// OutgoingLinks contains wiki links from the current page.
	OutgoingLinks []domain.PageLink
	// BrokenLinks contains outgoing wiki links with no current target.
	BrokenLinks []domain.PageLink
	// Comments contains anchored discussion items for the current page.
	Comments []domain.PageComment
	// Related contains pages related to the current page by tag.
	Related []domain.Page
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
	UserTokens []domain.APIToken
	// AdminSection identifies the active administration navigation section.
	AdminSection string
	// AdminStats contains high-level persisted object counts for administrators.
	AdminStats domain.AdminStats
	// ApplicationSettings contains mutable application-wide settings for administrators.
	ApplicationSettings domain.ApplicationSettings
	// PDFHeaders contains administrator-safe PDF request-header metadata.
	PDFHeaders []domain.PDFHeader
	// RenderingPreviews contains sanitized examples for administrator rendering controls.
	RenderingPreviews map[string]template.HTML
	// DocumentationHealth contains actionable documentation quality findings.
	DocumentationHealth domain.DocumentationHealth
	// RenderingLanguages lists content languages available to administrators.
	RenderingLanguages []renderingLanguageOption
	// PageContentLanguage is the effective language for the current page/editor.
	PageContentLanguage string
	// AdminUsers contains users and group memberships for administrators.
	AdminUsers []domain.AdminUser
	// PendingOIDCIdentities contains verified OIDC identities awaiting an administrator decision.
	PendingOIDCIdentities []domain.PendingOIDCIdentity
	// OIDCIdentityCount is the number of active external OIDC bindings.
	OIDCIdentityCount int
	// Groups contains administratively managed user groups.
	Groups []domain.Group
	// PageTemplates contains reusable templates available to page authors.
	PageTemplates []domain.PageTemplate
	// PageAccessRules contains inherited path access rules for administrators.
	PageAccessRules []domain.PageAccessRule
	// Webhooks contains outgoing administrator integrations.
	Webhooks []domain.Webhook
	// WebhookDeliveries contains recent outgoing delivery attempts.
	WebhookDeliveries []domain.WebhookDelivery
	// WebhookEvents contains supported event names.
	WebhookEvents []string
	// WebhookDraft provides enabled defaults for the create form.
	WebhookDraft domain.Webhook
	// PageVariables contains distinct variables resolved in this reading page.
	PageVariables []pageVariable
	// KnowledgeSnippets contains reusable variables and Markdown snippets.
	KnowledgeSnippets []domain.KnowledgeSnippet
	// PageStatuses contains lifecycle statuses available to page editors.
	PageStatuses []string
	// EditorTemplate is the selected template used to prefill a new page.
	EditorTemplate *domain.PageTemplate
	// EditorInitialSlug pre-fills a requested path for a new page.
	EditorInitialSlug string
	// EditorParentPath is the selected parent location in the guided page-path picker.
	EditorParentPath string
	// EditorPathSegment is the stable final path segment for edits and explicit new-page links.
	EditorPathSegment string
	// PagePathOptions contains existing page and folder locations available as parents.
	PagePathOptions []pagePathOption
	// NewPageParent is the active page path inherited by contextual new-page actions.
	NewPageParent string
	// AdminTags contains tags and page usage counts for administrators.
	AdminTags []domain.TagInfo
	// AdminTokens contains all personal access tokens for administrators.
	AdminTokens []domain.APIToken
	// AuditEvents contains recent administrative audit events.
	AuditEvents []domain.AuditEvent
	// AdminPages contains all pages available for administrative export.
	AdminPages []domain.Page
	// DeletedPages contains pages currently held in the recycle bin.
	DeletedPages []domain.DeletedPage
	// AdminNavigation contains top-level navigation sections and their persisted icons.
	AdminNavigation []domain.NavigationItem
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

type pageTemplateView struct {
	domain.PageTemplate
	Groups       []domain.Group
	PageStatuses []string
}

type webhookView struct {
	domain.Webhook
	AvailableEvents         []string
	EncryptionKeyConfigured bool
}

type pagePathOption struct {
	Slug  string
	Label string
}

// pagePathOptions flattens the navigation tree into selectable parent paths.
func pagePathOptions(tree []navigation.Node, excludedSlug string) []pagePathOption {
	var options []pagePathOption
	var appendNodes func([]navigation.Node, []string)

	appendNodes = func(nodes []navigation.Node, ancestors []string) {
		for _, node := range nodes {
			if excludedSlug != "" &&
				(node.Slug == excludedSlug || strings.HasPrefix(node.Slug, excludedSlug+"/")) {
				continue
			}

			labels := append(slices.Clone(ancestors), node.Title)
			options = append(options, pagePathOption{
				Slug:  node.Slug,
				Label: strings.Join(labels, " / "),
			})
			appendNodes(node.Children, labels)
		}
	}

	appendNodes(tree, nil)
	return options
}

// hasPagePathOption reports whether a parent path can be selected.
func hasPagePathOption(options []pagePathOption, slug string) bool {
	if slug == "" {
		return true
	}
	return slices.ContainsFunc(options, func(option pagePathOption) bool {
		return option.Slug == slug
	})
}
