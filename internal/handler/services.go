package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gi8lino/lore/internal/revision"
	"github.com/gi8lino/lore/internal/service"
)

type viewDataService interface {
	Load(*http.Request, *Views, string) (ViewData, error)
}

type administrationService interface {
	Stats(context.Context) (service.AdminStats, error)
	TagInfos(context.Context) ([]service.TagInfo, error)
	DeleteTag(context.Context, int64) error
	DocumentationHealth(context.Context, time.Time) (service.DocumentationHealth, error)
	AuditEvents(context.Context, int) ([]service.AuditEvent, error)
}

// Catalog interfaces are intentionally consumer-oriented instead of mirroring
// every method exposed by service.Catalog.
type pageContentService interface {
	GetPage(context.Context, string) (service.Page, error)
}

type pageLookupService interface {
	pageContentService
	ResolvePageAlias(context.Context, string) (string, error)
}

type pageListService interface {
	ListPages(context.Context, int) ([]service.Page, error)
}

type pageSearchService interface {
	Search(context.Context, string, int) ([]service.Page, error)
}

type homeCatalogService interface {
	pageListService
	Favorites(context.Context, int64) ([]service.Page, error)
	RecentViewed(context.Context, int64, int) ([]service.Page, error)
	RecentEdited(context.Context, int64, int) ([]service.RecentEdit, error)
	Popular(context.Context, int) ([]service.Page, error)
}

type draftListService interface {
	List(context.Context, int64, int) ([]service.PageDraft, error)
}

type editorDraftService interface {
	Draft(context.Context, int64, string) (service.PageDraft, error)
	Save(context.Context, service.PageDraftSaveInput) (service.PageDraft, error)
	Delete(context.Context, int64, string) error
}

type draftDiscardService interface {
	Delete(context.Context, int64, string) error
}

type sidebarCatalogService interface {
	Favorites(context.Context, int64) ([]service.Page, error)
	RecentViewed(context.Context, int64, int) ([]service.Page, error)
}

type pageViewCatalogService interface {
	pageLookupService
	pageSearchService
	RecordView(context.Context, string, int64) error
	IsFavorite(context.Context, string, int64) (bool, error)
	Backlinks(context.Context, string) ([]service.Page, error)
	PageLinks(context.Context, string) ([]service.PageLink, error)
	LatestRevision(context.Context, string) (revision.Revision, int, error)
	PageComments(context.Context, string) ([]service.PageComment, error)
}

type favoriteService interface {
	SetFavorite(context.Context, string, int64, bool) error
}

type pageRevisionService interface {
	Revisions(context.Context, string) ([]revision.Revision, error)
}

type pageTagService interface {
	Tags(context.Context) ([]string, error)
}

type pageAliasService interface {
	PageAliases(context.Context) (map[string]string, error)
}

type pageInventoryService interface {
	PageInventory(context.Context) ([]service.Page, error)
}

type pagePermalinkService interface {
	PageSlugByID(context.Context, int64) (string, error)
}

type groupReader interface {
	Groups(context.Context) ([]service.Group, error)
	AssignableGroups(context.Context, service.User) ([]service.Group, error)
	GroupMembers(context.Context, int64) ([]service.User, error)
}

type groupWriter interface {
	CreateGroup(context.Context, string) (service.Group, error)
	DeleteGroup(context.Context, int64) error
	AddGroupMember(context.Context, int64, int64) error
	RemoveGroupMember(context.Context, int64, int64) error
}

// Knowledge interfaces expose only the slice of knowledge functionality a
// handler needs.
type knowledgeContentService interface {
	KnowledgeSnippetByName(context.Context, string, string) (service.KnowledgeSnippet, error)
}

type knowledgeSidebarService interface {
	SavedSearches(context.Context, int64) ([]service.SavedSearch, error)
	Notifications(context.Context, int64, int) ([]service.Notification, int, error)
}

type knowledgeGraphService interface {
	KnowledgeGraph(context.Context, int) (service.KnowledgeGraph, error)
}

type knowledgeSnippetReader interface {
	knowledgeContentService
	KnowledgeSnippets(context.Context) ([]service.KnowledgeSnippet, error)
}

type knowledgeSnippetService interface {
	knowledgeSnippetReader
	SaveKnowledgeSnippet(context.Context, int64, int64, string, string, string, string) (service.KnowledgeSnippet, error)
	DeleteKnowledgeSnippet(context.Context, int64, int64) error
}

type savedSearchService interface {
	SaveSavedSearch(context.Context, int64, int64, string, string, bool) error
	DeleteSavedSearch(context.Context, int64, int64) error
}

type notificationService interface {
	Notifications(context.Context, int64, int) ([]service.Notification, int, error)
	MarkNotificationRead(context.Context, int64, int64) error
}

// Media readers and writers are separated so read-only exports and downloads
// do not receive upload/delete capabilities.
type imageContentService interface {
	ImageContent(context.Context, int64) (service.ImageData, error)
}

type imageListService interface {
	Images(context.Context) ([]service.Image, error)
}

type userImageService interface {
	ImagesByUser(context.Context, int64) ([]service.Image, error)
}

type imageService interface {
	imageContentService
	Images(context.Context) ([]service.Image, error)
	ImagesByUser(context.Context, int64) ([]service.Image, error)
	UploadImage(context.Context, string, []byte, service.User) (service.Image, error)
	DeleteImage(context.Context, int64, service.User) error
}

type attachmentService interface {
	Attachments(context.Context) ([]service.Attachment, error)
	AttachmentContent(context.Context, int64) (service.AttachmentData, error)
	UploadAttachment(context.Context, string, []byte, service.User) (service.Attachment, error)
	DeleteAttachment(context.Context, int64, service.User) error
}

type navigationService interface {
	NavigationPages(context.Context) ([]service.Page, error)
	NavigationItems(context.Context) ([]service.NavigationItem, error)
	NavigationIcons(context.Context) (map[string]string, error)
	SetNavigationIcon(context.Context, string, string) error
}

// Page mutation interfaces follow the individual workflows rather than
// exposing the complete Pages service to every write handler.
type pageWriterService interface {
	Save(context.Context, service.PageSaveInput) (service.Page, error)
	Delete(context.Context, string, service.User) error
}

type pageMoveService interface {
	Move(context.Context, string, string, service.MovePageOptions, service.User) error
}

type pageReviewService interface {
	Review(context.Context, string, service.User) error
}

type pageRevisionWriter interface {
	RestoreRevision(context.Context, string, int, service.User) (service.Page, error)
}

type pageDiscussionWriter interface {
	AddComment(context.Context, string, string, string, service.User) error
	ResolveComment(context.Context, int64, bool) error
}

type pageImportService interface {
	Import(context.Context, []service.ImportedPage, string, service.User) (int, error)
}

type pageBulkService interface {
	Bulk(context.Context, service.BulkPageInput) error
}

type preferenceService interface {
	Preferences(context.Context, int64) (service.UserPreferences, error)
	SavePreferences(context.Context, int64, service.UserPreferences) error
	SetShowPageContents(context.Context, int64, bool) error
	SetExpandedNavigation(context.Context, int64, []string) error
	SetSidebarWidth(context.Context, int64, int) error
}

type recycleBinService interface {
	DeletedPages(context.Context) ([]service.DeletedPage, error)
	RestorePage(context.Context, string) error
	PermanentlyDeletePage(context.Context, string) error
}

type settingsService interface {
	ApplicationSettings(context.Context) (service.ApplicationSettings, error)
	SaveApplicationSettings(context.Context, service.ApplicationSettings, int64) error
	SaveAuthenticationSettings(context.Context, service.AuthenticationSettings, int64) error
	SaveRenderingSettings(context.Context, service.RenderingSettings, int64) error
	RecordLocalPasswordUpdated(context.Context, service.User)
}

type sharingService interface {
	CreatePageShareLink(context.Context, string, service.User) (service.IssuedPageShareLink, error)
	PageShareLink(context.Context, string) (service.PageShareLink, error)
}

type systemService interface {
	Ping(context.Context) error
	SetupRequired(context.Context) (bool, error)
	RecordSetupCompleted(context.Context, service.User)
}

type templateService interface {
	PageTemplates(context.Context) ([]service.PageTemplate, error)
	PageTemplate(context.Context, int64) (service.PageTemplate, error)
	CreatePageTemplate(context.Context, string, string, string) (service.PageTemplate, error)
	UpdatePageTemplate(context.Context, int64, string, string, string) error
	DeletePageTemplate(context.Context, int64) error
}

type tokenService interface {
	Tokens(context.Context) ([]service.APIToken, error)
	UserTokens(context.Context, int64) ([]service.APIToken, error)
	CreateToken(context.Context, string, int64, int64, *time.Time) (service.IssuedToken, error)
	DeleteUserToken(context.Context, int64, int64) error
	DeleteToken(context.Context, int64) error
}

// User administration is split into directory, account management, and OIDC
// identity capabilities.
type userDirectoryService interface {
	User(context.Context, int64) (service.User, error)
	SearchUsers(context.Context, string, int) ([]service.User, error)
}

type userManagementService interface {
	Users(context.Context) ([]service.AdminUser, error)
	UserGroups(context.Context, int64) ([]service.Group, error)
	UpdateUser(context.Context, int64, string, []int64) error
}

type oidcIdentityService interface {
	OIDCIdentities(context.Context) ([]service.OIDCIdentity, error)
	OIDCGroupMappings(context.Context) ([]service.OIDCGroupMapping, error)
	PendingOIDCIdentities(context.Context) ([]service.PendingOIDCIdentity, error)
	ApprovePendingOIDCIdentity(context.Context, int64, int64) (service.User, error)
	LinkPendingOIDCIdentity(context.Context, int64, int64, int64) (service.User, error)
	SetPendingOIDCIdentityRejected(context.Context, int64, bool, int64) error
	RemoveOIDCIdentity(context.Context, int64, string, string, int64) error
	HasLocalCredential(context.Context, int64) (bool, error)
}

type adminUserOverviewService interface {
	userManagementService
	oidcIdentityService
}
