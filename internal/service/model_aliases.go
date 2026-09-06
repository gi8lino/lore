package service

import "github.com/gi8lino/lore/internal/model"

// Application-facing records are aliases of persistence-agnostic domain
// models. The model package owns shared data shapes so services never depend
// on PostgreSQL implementation types.
type (
	AdminStats             = model.AdminStats
	AdminUser              = model.AdminUser
	APIToken               = model.APIToken
	ApplicationSettings    = model.ApplicationSettings
	Attachment             = model.Attachment
	AttachmentData         = model.AttachmentData
	AuditEvent             = model.AuditEvent
	AuthenticationSettings = model.AuthenticationSettings
	DeletedPage            = model.DeletedPage
	DocumentationHealth    = model.DocumentationHealth
	Group                  = model.Group
	Image                  = model.Image
	ImageData              = model.ImageData
	IssuedToken            = model.IssuedToken
	KnowledgeGraph         = model.KnowledgeGraph
	KnowledgeSnippet       = model.KnowledgeSnippet
	MovePageOptions        = model.MovePageOptions
	NavigationItem         = model.NavigationItem
	Notification           = model.Notification
	OIDCGroupMapping       = model.OIDCGroupMapping
	OIDCIdentity           = model.OIDCIdentity
	Page                   = model.Page
	PageComment            = model.PageComment
	PageLink               = model.PageLink
	PageMetadata           = model.PageMetadata
	PageProperty           = model.PageProperty
	PageShareLink          = model.PageShareLink
	PageTemplate           = model.PageTemplate
	PendingOIDCIdentity    = model.PendingOIDCIdentity
	RecentEdit             = model.RecentEdit
	PageDraft              = model.PageDraft
	RenderingSettings      = model.RenderingSettings
	SavedSearch            = model.SavedSearch
	TagInfo                = model.TagInfo
	User                   = model.User
	UserPreferences        = model.UserPreferences
)

var (
	ErrAlreadyExists = model.ErrAlreadyExists
	ErrForbidden     = model.ErrForbidden
	ErrNotFound      = model.ErrNotFound
	ErrPageInBin     = model.ErrPageInBin
)

const (
	MaxSidebarWidth              = model.MaxSidebarWidth
	MinSidebarWidth              = model.MinSidebarWidth
	NavigationDensityComfortable = model.NavigationDensityComfortable
	NavigationDensityCompact     = model.NavigationDensityCompact
)

// PageStatuses returns the supported page lifecycle states.
func PageStatuses() []string { return model.PageStatuses() }

// DefaultUserPreferences returns preferences for a new user.
func DefaultUserPreferences() UserPreferences { return model.DefaultUserPreferences() }

// ValidPageStatus reports whether value is a supported page lifecycle state.
func ValidPageStatus(value string) bool { return model.ValidPageStatus(value) }

// ValidUserRole reports whether value is a supported account role.
func ValidUserRole(value string) bool { return model.ValidUserRole(value) }

// ValidNavigationDensity reports whether value is a supported navigation density.
func ValidNavigationDensity(value string) bool { return model.ValidNavigationDensity(value) }

// ValidSidebarWidth reports whether width is inside the supported desktop range.
func ValidSidebarWidth(width int) bool { return model.ValidSidebarWidth(width) }

// ValidReviewIntervalDays reports whether days is a supported review interval.
func ValidReviewIntervalDays(days int) bool { return model.ValidReviewIntervalDays(days) }
