package store

import "github.com/gi8lino/lore/internal/model"

// Persistence-facing aliases keep SQL code concise while domain records live in internal/model.
type (
	AdminStats             = model.AdminStats
	AdminUser              = model.AdminUser
	Group                  = model.Group
	TagInfo                = model.TagInfo
	RenderingSettings      = model.RenderingSettings
	AuthenticationSettings = model.AuthenticationSettings
	ApplicationSettings    = model.ApplicationSettings
	Attachment             = model.Attachment
	AttachmentData         = model.AttachmentData
	AuditEvent             = model.AuditEvent
	BrokenWikiLink         = model.BrokenWikiLink
	DocumentationHealth    = model.DocumentationHealth
	Image                  = model.Image
	ImageData              = model.ImageData
	PageMetadata           = model.PageMetadata
	PageProperty           = model.PageProperty
	KnowledgeSnippet       = model.KnowledgeSnippet
	SavedSearch            = model.SavedSearch
	Notification           = model.Notification
	PageComment            = model.PageComment
	GraphNode              = model.GraphNode
	GraphEdge              = model.GraphEdge
	KnowledgeGraph         = model.KnowledgeGraph
	RecentEdit             = model.RecentEdit
	PageDraft              = model.PageDraft
	MovePageOptions        = model.MovePageOptions
	NavigationItem         = model.NavigationItem
	OIDCIdentity           = model.OIDCIdentity
	OIDCGroupMapping       = model.OIDCGroupMapping
	PendingOIDCIdentity    = model.PendingOIDCIdentity
	PageLink               = model.PageLink
	PageTemplate           = model.PageTemplate
	UserPreferences        = model.UserPreferences
	PageShareLink          = model.PageShareLink
	User                   = model.User
	Page                   = model.Page
	DeletedPage            = model.DeletedPage
	APIToken               = model.APIToken
	IssuedToken            = model.IssuedToken
)

var (
	ErrNotFound                 = model.ErrNotFound
	ErrAlreadyExists            = model.ErrAlreadyExists
	ErrForbidden                = model.ErrForbidden
	ErrRegistrationDisabled     = model.ErrRegistrationDisabled
	ErrIdentityApprovalRequired = model.ErrIdentityApprovalRequired
	ErrIdentityRejected         = model.ErrIdentityRejected
	ErrPageInBin                = model.ErrPageInBin
)

const (
	NavigationDensityComfortable = model.NavigationDensityComfortable
	NavigationDensityCompact     = model.NavigationDensityCompact
	DefaultSidebarWidth          = model.DefaultSidebarWidth
	MinSidebarWidth              = model.MinSidebarWidth
	MaxSidebarWidth              = model.MaxSidebarWidth
)

// PageStatuses returns the supported page lifecycle statuses.
func PageStatuses() []string { return model.PageStatuses() }

// ValidPageStatus reports whether a page status is supported.
func ValidPageStatus(value string) bool { return model.ValidPageStatus(value) }

// DefaultUserPreferences returns presentation defaults for a new user.
func DefaultUserPreferences() UserPreferences { return model.DefaultUserPreferences() }
