package model

import (
	"errors"
	"time"
)

var (
	// ErrNotFound indicates that a requested domain object does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists indicates that a unique domain object already exists.
	ErrAlreadyExists = errors.New("already exists")
	// ErrForbidden indicates that a requested domain mutation is not permitted.
	ErrForbidden = errors.New("forbidden")
	// ErrRegistrationDisabled indicates that a new external identity may not create an account.
	ErrRegistrationDisabled = errors.New("user registration is disabled")
	// ErrIdentityApprovalRequired indicates that an OIDC identity awaits administrator approval.
	ErrIdentityApprovalRequired = errors.New("OIDC identity requires administrator approval")
	// ErrIdentityRejected indicates that an administrator rejected an OIDC identity.
	ErrIdentityRejected = errors.New("OIDC identity was rejected")
	// ErrPageInBin indicates that a page path is occupied by a recycled page.
	ErrPageInBin = errors.New("page path is in recycle bin")
)

const (
	// NavigationDensityComfortable is the default roomy sidebar layout.
	NavigationDensityComfortable = "comfortable"
	// NavigationDensityCompact reduces vertical navigation spacing.
	NavigationDensityCompact = "compact"
	// DefaultSidebarWidth is the default desktop sidebar width in CSS pixels.
	DefaultSidebarWidth = 280
	// MinSidebarWidth is the smallest supported desktop sidebar width.
	MinSidebarWidth = 220
	// MaxSidebarWidth is the largest supported desktop sidebar width.
	MaxSidebarWidth = 420
)

// AdminStats contains high-level object counts shown on the administration page.
type AdminStats struct {
	// Users is the number of wiki users.
	Users int64
	// Groups is the number of user groups.
	Groups int64
	// Pages is the number of active wiki pages.
	Pages int64
	// DeletedPages is the number of pages currently in the recycle bin.
	DeletedPages int64
	// Tags is the number of tags.
	Tags int64
	// Images is the number of uploaded images.
	Images int64
	// Tokens is the number of active or expired API tokens.
	Tokens int64
}

// AdminUser contains a user and the groups currently assigned to the account.
type AdminUser struct {
	// User is the wiki account.
	User User
	// Groups contains group names assigned to the user.
	Groups []string
	// OIDCIdentities contains external OIDC identities bound to the user.
	OIDCIdentities []OIDCIdentity
	// LastLogin is the most recent successful authentication time.
	LastLogin time.Time
	// HasLoggedIn reports whether LastLogin represents an actual login.
	HasLoggedIn bool
}

// Group describes one administratively managed user group.
type Group struct {
	// ID is the stable identifier.
	ID int64 `json:"id"`
	// Name is the unique group name.
	Name string `json:"name"`
	// UserCount is the number of users assigned to the group.
	UserCount int64 `json:"user_count,omitempty"`
	// PageCount is the number of pages assigned to the group.
	PageCount int64 `json:"page_count,omitempty"`
}

// TagInfo describes one tag and its current page usage count.
type TagInfo struct {
	// ID is the stable identifier.
	ID int64
	// Name is the normalized tag name.
	Name string
	// PageCount is the number of pages using the tag.
	PageCount int64
}

// RenderingSettings controls wiki content presentation and optional Markdown rendering features.
type RenderingSettings struct {
	// WikiLinks enables [[Wiki Link]] resolution.
	WikiLinks bool
	// Callouts enables !!! callout blocks.
	Callouts bool
	// Tabs enables Material-style === tab blocks.
	Tabs bool
	// Details enables ??? collapsible detail blocks.
	Details bool
	// Tables enables GitHub-flavored Markdown tables.
	Tables bool
	// TableStyles enables theme-aware table color directives.
	TableStyles bool
	// TableSorting enables sortable table directives.
	TableSorting bool
	// TableFiltering enables filterable table directives.
	TableFiltering bool
	// Strikethrough enables GitHub-flavored ~~strikethrough~~.
	Strikethrough bool
	// TaskLists enables GitHub-flavored task lists.
	TaskLists bool
	// Autolinks enables automatic URL and email links.
	Autolinks bool
	// SyntaxHighlighting enables server-side fenced-code highlighting.
	SyntaxHighlighting bool
	// ContentLanguage is the BCP 47 language tag applied to wiki content and the editor.
	ContentLanguage string
	// CodingLigatures enables supported OpenType coding ligatures in editor and code fonts.
	CodingLigatures bool
	// Mermaid enables browser-side Mermaid diagram rendering.
	Mermaid bool
	// Footnotes enables Markdown footnotes.
	Footnotes bool
	// DefinitionLists enables Markdown definition lists.
	DefinitionLists bool
	// Typographer enables smart punctuation substitutions.
	Typographer bool
}

// AuthenticationSettings controls browser authentication without storing secrets.
type AuthenticationSettings struct {
	// Mode selects local, trusted-proxy, or OIDC authentication.
	Mode string
	// OIDCIssuer is the OIDC discovery issuer URL.
	OIDCIssuer string
	// OIDCClientID is the public OIDC client identifier.
	OIDCClientID string
	// OIDCGroupClaim is the top-level claim containing external group names.
	OIDCGroupClaim string
	// OIDCGroupSync enables synchronization of explicitly mapped OIDC groups.
	OIDCGroupSync bool
	// OIDCGroupsAuthoritative removes mapped memberships that disappear from the claim.
	OIDCGroupsAuthoritative bool
	// OIDCGroupMappings maps external group values to Lore groups.
	OIDCGroupMappings []OIDCGroupMapping
	// TrustedUsernameHeaders lists trusted-proxy username headers in priority order.
	TrustedUsernameHeaders []string
	// TrustedEmailHeaders lists trusted-proxy email headers in priority order.
	TrustedEmailHeaders []string
	// TrustedDisplayNameHeaders lists trusted-proxy display-name headers in priority order.
	TrustedDisplayNameHeaders []string
}

// ApplicationSettings contains mutable application-wide settings.
type ApplicationSettings struct {
	// AllowUserRegistration permits new OIDC and trusted-proxy identities to create wiki accounts.
	AllowUserRegistration bool
	// DiscussionsEnabled enables page comments and anchored discussions.
	DiscussionsEnabled bool
	// Authentication contains non-secret browser authentication settings.
	Authentication AuthenticationSettings
	// Rendering contains administrator-controlled Markdown rendering features.
	Rendering RenderingSettings
}

// Attachment contains metadata for a stored non-image file.
type Attachment struct {
	ID          int64     `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedBy  int64     `json:"uploaded_by"`
	Uploader    string    `json:"uploader"`
	CreatedAt   time.Time `json:"created_at"`
	UsageCount  int64     `json:"usage_count"`
}

// AttachmentData contains stored attachment bytes.
type AttachmentData struct {
	Attachment
	Data []byte
}

// AuditEvent describes one administratively visible application action.
type AuditEvent struct {
	ID         int64
	Actor      string
	Action     string
	ObjectType string
	ObjectKey  string
	Detail     string
	CreatedAt  time.Time
}

// BrokenWikiLink describes a wiki link whose target page does not exist.
type BrokenWikiLink struct {
	SourceSlug  string
	SourceTitle string
	TargetSlug  string
}

// DocumentationHealth groups page-quality findings for administrators.
type DocumentationHealth struct {
	BrokenLinks   []BrokenWikiLink
	OrphanPages   []Page
	UntaggedPages []Page
	UniconedPages []Page
	StalePages    []Page
	ReviewDue     []Page
	DraftPages    []Page
	Deprecated    []Page
}

// Image contains metadata for one uploaded wiki image.
type Image struct {
	// ID is the stable identifier used in image URLs.
	ID int64 `json:"id"`
	// Filename is the sanitized original image filename.
	Filename string `json:"filename"`
	// ContentType is the validated image MIME type.
	ContentType string `json:"content_type"`
	// SizeBytes is the stored image size in bytes.
	SizeBytes int64 `json:"size_bytes"`
	// UploadedBy is the identifier of the user that uploaded the image.
	UploadedBy int64 `json:"uploaded_by"`
	// Uploader is the display name of the user that uploaded the image.
	Uploader string `json:"uploader"`
	// CreatedAt is the upload timestamp.
	CreatedAt time.Time `json:"created_at"`
	// UsageCount is the number of Markdown references to the image across all pages.
	UsageCount int64 `json:"usage_count"`
}

// ImageData contains the binary payload and response metadata for one image.
type ImageData struct {
	// Filename is the sanitized image filename.
	Filename string
	// ContentType is the validated image MIME type.
	ContentType string
	// Data is the complete stored image payload.
	Data []byte
}

// PageMetadata contains optional workflow metadata attached to a page.
type PageMetadata struct {
	Status             string `json:"status"`
	OwnerGroupID       int64  `json:"owner_group_id,omitempty"`
	ReviewIntervalDays int    `json:"review_interval_days,omitempty"`
	MarkReviewed       bool   `json:"mark_reviewed,omitempty"`
	DeprecatedTarget   string `json:"deprecated_target,omitempty"`
}

// PageProperty is one searchable structured metadata value attached to a page.
type PageProperty struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// KnowledgeSnippet is a reusable variable or Markdown snippet.
type KnowledgeSnippet struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SavedSearch is a named user search that can be surfaced in navigation.
type SavedSearch struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Query  string `json:"query"`
	Pinned bool   `json:"pinned"`
}

// Notification is a lightweight user inbox item.
type Notification struct {
	ID        int64      `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	URL       string     `json:"url"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// PageComment is a discussion item optionally anchored to selected page text.
type PageComment struct {
	ID        int64      `json:"id"`
	PageID    int64      `json:"page_id"`
	Author    string     `json:"author"`
	Anchor    string     `json:"anchor"`
	Body      string     `json:"body"`
	Resolved  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// GraphNode is one page in the wiki relationship graph.
type GraphNode struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// GraphEdge is one wiki-link relationship between pages.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// KnowledgeGraph contains graph nodes and link edges.
type KnowledgeGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// RecentEdit describes a page recently edited by one user.
type RecentEdit struct {
	Page
	RevisionMessage string
}

// PageDraft is a private, autosaved editor state owned by one user.
type PageDraft struct {
	ID              int64               `json:"id"`
	Key             string              `json:"key"`
	PageID          int64               `json:"page_id,omitempty"`
	BaseRevision    int                 `json:"base_revision"`
	CurrentRevision int                 `json:"current_revision"`
	Stale           bool                `json:"stale"`
	Title           string              `json:"title"`
	Slug            string              `json:"slug"`
	PageSlug        string              `json:"page_slug,omitempty"`
	Values          map[string][]string `json:"values,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// MovePageOptions controls safe page-tree refactoring.
type MovePageOptions struct {
	MoveChildren        bool
	UpdateIncomingLinks bool
	KeepAliases         bool
}

// NavigationItem describes one page or synthetic folder in the navigation tree.
type NavigationItem struct {
	// Path is the complete navigation path.
	Path string
	// Title is the page title or the raw path segment for synthetic folders.
	Title string
	// Icon is the explicitly selected Lucide icon name.
	Icon string
	// Page reports whether the path maps to a real wiki page.
	Page bool
}

// OIDCIdentity binds one Lore account to a stable identity from an OIDC issuer.
type OIDCIdentity struct {
	UserID    int64
	Issuer    string
	Subject   string
	CreatedAt time.Time
}

// OIDCGroupMapping maps one external OIDC group value to a Lore group.
type OIDCGroupMapping struct {
	OIDCGroup string
	GroupID   int64
	GroupName string
}

// PendingOIDCIdentity is a verified but not yet accepted OIDC identity.
type PendingOIDCIdentity struct {
	ID                   int64
	Issuer               string
	Subject              string
	Username             string
	Email                string
	DisplayName          string
	Status               string
	FirstSeenAt          time.Time
	LastSeenAt           time.Time
	SuggestedUserID      int64
	SuggestedUsername    string
	SuggestedDisplayName string
}

// PageLink describes one wiki link recorded for a source page.
type PageLink struct {
	TargetSlug  string
	TargetTitle string
	Exists      bool
}

// PageTemplate is reusable Markdown content offered when creating a page.
type PageTemplate struct {
	ID          int64
	Name        string
	Description string
	Markdown    string
}

// UserPreferences contains presentation preferences for one wiki user.
type UserPreferences struct {
	// Theme is the filename-derived title of the user's selected theme.
	Theme string
	// ShowPageContents controls whether wiki pages render a heading table of contents.
	ShowPageContents bool
	// NavigationDensity controls vertical spacing in the page tree.
	NavigationDensity string
	// SidebarWidth is the desktop sidebar width in CSS pixels.
	SidebarWidth int
	// ShowNavigationGuides controls tree indentation guide lines.
	ShowNavigationGuides bool
	// RememberNavigationState persists expanded and collapsed navigation folders.
	RememberNavigationState bool
	// ShowPinnedPages displays favorites at the top of the sidebar.
	ShowPinnedPages bool
	// ShowRecentlyViewed displays recently viewed pages above the page tree.
	ShowRecentlyViewed bool
	// ShowNavigationPageCounts displays descendant page counts for folders.
	ShowNavigationPageCounts bool
	// ExpandedNavigation contains folder slugs the user explicitly left expanded.
	ExpandedNavigation []string
}

// PageShareLink identifies the page exposed by one public permalink.
type PageShareLink struct {
	PageID int64
	Slug   string
	Title  string
}

// User represents an authenticated wiki account.
type User struct {
	// ID is the stable identifier.
	ID int64 `json:"id"`
	// Username is the unique login name.
	Username string `json:"username"`
	// Email is the account email address.
	Email string `json:"email"`
	// DisplayName is the human-readable account name.
	DisplayName string `json:"display_name"`
	// Role controls the account authorization level.
	Role string `json:"role"`
}

// Page represents the current state of a wiki page.
type Page struct {
	// ID is the stable identifier.
	ID int64 `json:"id"`
	// Slug is the unique URL path for the page.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Icon is the optional Lucide icon displayed with the page title.
	Icon string `json:"icon,omitempty"`
	// Language optionally overrides the wiki-wide content language.
	Language string `json:"language,omitempty"`
	// Markdown is the current Markdown body.
	Markdown string `json:"markdown_content,omitempty"`
	// CreatedBy is the identifier of the user that created the page.
	CreatedBy int64 `json:"created_by"`
	// UpdatedBy is the identifier of the user that last updated the page.
	UpdatedBy int64 `json:"updated_by"`
	// Author is the display name of the last editor.
	Author string `json:"author"`
	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the most recent update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
	// Tags contains normalized page tags.
	Tags []string `json:"tags"`
	// Groups contains the collaboration groups responsible for the page.
	Groups []Group `json:"groups,omitempty"`
	// ViewCount is the total recorded page views.
	ViewCount int64 `json:"view_count"`
	// Status is the page lifecycle state.
	Status string `json:"status"`
	// OwnerGroupID optionally assigns documentation ownership to a collaboration group.
	OwnerGroupID int64 `json:"owner_group_id,omitempty"`
	// OwnerGroup is the human-readable owner group name.
	OwnerGroup string `json:"owner_group,omitempty"`
	// LastReviewedAt records the most recent explicit documentation review.
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	// ReviewIntervalDays configures when the page should be reviewed again.
	ReviewIntervalDays int `json:"review_interval_days,omitempty"`
	// DeprecatedTarget points readers of a deprecated page to its replacement.
	DeprecatedTarget string `json:"deprecated_target,omitempty"`
	// Properties contains searchable structured page metadata.
	Properties []PageProperty `json:"properties,omitempty"`
	// Rank is the search relevance score.
	Rank float32 `json:"rank,omitempty"`
}

// DeletedPage describes a page currently held in the administrator recycle bin.
type DeletedPage struct {
	Page
	DeletedAt time.Time
	DeletedBy string
}

// APIToken contains non-secret metadata for one personal access token.
type APIToken struct {
	// ID is the stable identifier.
	ID int64 `json:"id"`
	// Name is the user-supplied token label.
	Name string `json:"name"`
	// UserID is the account authenticated by the token.
	UserID int64 `json:"user_id"`
	// Username is the login name authenticated by the token.
	Username string `json:"username"`
	// CreatedBy is the account that issued the token.
	CreatedBy int64 `json:"created_by"`
	// Creator is the display name of the account that issued the token.
	Creator string `json:"creator"`
	// CreatedAt is the issuance timestamp.
	CreatedAt time.Time `json:"created_at"`
	// LastUsed is the most recent successful authentication timestamp.
	LastUsed *time.Time `json:"last_used,omitempty"`
	// ExpiresAt is the optional expiration timestamp.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// IssuedToken contains token metadata and the one-time plaintext secret.
type IssuedToken struct {
	// Token contains the non-secret token metadata.
	Token APIToken `json:"token"`
	// Secret is the plaintext token shown only immediately after creation.
	Secret string `json:"secret"`
}

// PageStatuses returns the supported page lifecycle statuses.
func PageStatuses() []string {
	return []string{"draft", "verified", "deprecated", "archived"}
}

// ValidPageStatus reports whether a page status is supported.
func ValidPageStatus(value string) bool {
	for _, status := range PageStatuses() {
		if value == status {
			return true
		}
	}
	return false
}

// DefaultUserPreferences returns the presentation defaults used before a user saves preferences.
func DefaultUserPreferences() UserPreferences {
	return UserPreferences{
		ShowPageContents:         true,
		NavigationDensity:        NavigationDensityComfortable,
		SidebarWidth:             DefaultSidebarWidth,
		ShowNavigationGuides:     true,
		RememberNavigationState:  true,
		ShowPinnedPages:          true,
		ShowRecentlyViewed:       false,
		ShowNavigationPageCounts: false,
		ExpandedNavigation:       []string{},
	}
}
