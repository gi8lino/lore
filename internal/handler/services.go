package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gi8lino/lore/internal/model"
	"github.com/gi8lino/lore/internal/revision"
	"github.com/gi8lino/lore/internal/service"
)

type viewDataService interface {
	Load(*http.Request, *Views, string) (ViewData, error)
}

type administrationService interface {
	Stats(context.Context) (model.AdminStats, error)
	TagInfos(context.Context) ([]model.TagInfo, error)
	DeleteTag(context.Context, int64) error
	DocumentationHealth(context.Context, time.Time) (model.DocumentationHealth, error)
	AuditEvents(context.Context, int) ([]model.AuditEvent, error)
}

// Catalog interfaces are intentionally consumer-oriented instead of mirroring
// every method exposed by service.Catalog.
type pageContentService interface {
	GetPage(context.Context, string) (model.Page, error)
}

type pageLookupService interface {
	pageContentService
	ResolvePageAlias(context.Context, string) (string, error)
}

type pageListService interface {
	ListPages(context.Context, int) ([]model.Page, error)
}

type pageSearchService interface {
	Search(context.Context, string, int) ([]model.Page, error)
}

type homeCatalogService interface {
	pageListService
	Favorites(context.Context, int64) ([]model.Page, error)
	RecentViewed(context.Context, int64, int) ([]model.Page, error)
	RecentEdited(context.Context, int64, int) ([]model.RecentEdit, error)
	Popular(context.Context, int) ([]model.Page, error)
}

type draftListService interface {
	List(context.Context, int64, int) ([]model.PageDraft, error)
}

type editorDraftService interface {
	Draft(context.Context, int64, string) (model.PageDraft, error)
	Save(context.Context, service.PageDraftSaveInput) (model.PageDraft, error)
	Delete(context.Context, int64, string) error
}

type draftDiscardService interface {
	Delete(context.Context, int64, string) error
}

type sidebarCatalogService interface {
	Favorites(context.Context, int64) ([]model.Page, error)
	RecentViewed(context.Context, int64, int) ([]model.Page, error)
}

type pageViewCatalogService interface {
	pageLookupService
	pageSearchService
	RecordView(context.Context, string, int64) error
	IsFavorite(context.Context, string, int64) (bool, error)
	Backlinks(context.Context, string) ([]model.Page, error)
	PageLinks(context.Context, string) ([]model.PageLink, error)
	LatestRevision(context.Context, string) (record revision.Revision, count int, err error)
	PageComments(context.Context, string) ([]model.PageComment, error)
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
	PageInventory(context.Context) ([]model.Page, error)
}

type pagePermalinkService interface {
	PageSlugByID(context.Context, int64) (string, error)
}

type groupReader interface {
	Groups(context.Context) ([]model.Group, error)
	AssignableGroups(context.Context, model.User) ([]model.Group, error)
	GroupMembers(context.Context, int64) ([]model.User, error)
}

type groupWriter interface {
	CreateGroup(context.Context, string) (model.Group, error)
	DeleteGroup(context.Context, int64) error
	AddGroupMember(context.Context, int64, int64) error
	RemoveGroupMember(context.Context, int64, int64) error
}

// Knowledge interfaces expose only the slice of knowledge functionality a
// handler needs.
type knowledgeContentService interface {
	KnowledgeSnippetByName(context.Context, string, string) (model.KnowledgeSnippet, error)
}

type knowledgeSidebarService interface {
	SavedSearches(context.Context, int64) ([]model.SavedSearch, error)
	Notifications(context.Context, int64, int) (notifications []model.Notification, unread int, err error)
}

type knowledgeGraphService interface {
	KnowledgeGraph(context.Context, int) (model.KnowledgeGraph, error)
}

type knowledgeSnippetReader interface {
	knowledgeContentService
	KnowledgeSnippets(context.Context) ([]model.KnowledgeSnippet, error)
}

type knowledgeSnippetService interface {
	knowledgeSnippetReader
	SaveKnowledgeSnippet(context.Context, int64, int64, string, string, string, string) (model.KnowledgeSnippet, error)
	DeleteKnowledgeSnippet(context.Context, int64, int64) error
}

type savedSearchService interface {
	SaveSavedSearch(context.Context, int64, int64, string, string, bool) error
	DeleteSavedSearch(context.Context, int64, int64) error
}

type notificationService interface {
	Notifications(context.Context, int64, int) (notifications []model.Notification, unread int, err error)
	MarkNotificationRead(context.Context, int64, int64) error
}

// Media readers and writers are separated so read-only exports and downloads
// do not receive upload/delete capabilities.
type imageContentService interface {
	ImageContent(context.Context, int64) (model.ImageData, error)
}

type imageListService interface {
	Images(context.Context) ([]model.Image, error)
}

type userImageService interface {
	ImagesByUser(context.Context, int64) ([]model.Image, error)
}

type imageService interface {
	imageContentService
	Images(context.Context) ([]model.Image, error)
	ImagesByUser(context.Context, int64) ([]model.Image, error)
	UploadImage(context.Context, string, []byte, model.User) (model.Image, error)
	DeleteImage(context.Context, int64, model.User) error
}

type attachmentService interface {
	Attachments(context.Context) ([]model.Attachment, error)
	AttachmentContent(context.Context, int64) (model.AttachmentData, error)
	UploadAttachment(context.Context, string, []byte, model.User) (model.Attachment, error)
	DeleteAttachment(context.Context, int64, model.User) error
}

type navigationService interface {
	NavigationPages(context.Context) ([]model.Page, error)
	NavigationItems(context.Context) ([]model.NavigationItem, error)
	NavigationIcons(context.Context) (map[string]string, error)
	SetNavigationIcon(context.Context, string, string) error
}

// Page mutation interfaces follow the individual workflows rather than
// exposing the complete Pages service to every write handler.
type pageWriterService interface {
	Save(context.Context, service.PageSaveInput) (model.Page, error)
	Delete(context.Context, string, model.User) error
}

type pageMoveService interface {
	Move(context.Context, string, string, model.MovePageOptions, model.User) error
}

type pageReviewService interface {
	Review(context.Context, string, model.User) error
}

type pageRevisionWriter interface {
	RestoreRevision(context.Context, string, int, model.User) (model.Page, error)
}

type pageDiscussionWriter interface {
	AddComment(context.Context, string, string, string, model.User) error
	ResolveComment(context.Context, int64, bool) error
}

type pageImportService interface {
	Import(context.Context, []service.ImportedPage, string, model.User) (int, error)
}

type pageBulkService interface {
	Bulk(context.Context, service.BulkPageInput) error
}

type preferenceService interface {
	Preferences(context.Context, int64) (model.UserPreferences, error)
	SavePreferences(context.Context, int64, model.UserPreferences) error
	SetShowPageContents(context.Context, int64, bool) error
	SetExpandedNavigation(context.Context, int64, []string) error
	SetSidebarWidth(context.Context, int64, int) error
}

type recycleBinService interface {
	DeletedPages(context.Context) ([]model.DeletedPage, error)
	RestorePage(context.Context, string) error
	PermanentlyDeletePage(context.Context, string) error
}

type settingsService interface {
	ApplicationSettings(context.Context) (model.ApplicationSettings, error)
	SaveApplicationSettings(context.Context, model.ApplicationSettings, int64) error
	SavePDFSettings(context.Context, string, int64) error
	SaveAuthenticationSettings(context.Context, model.AuthenticationSettings, int64) error
	SaveRenderingSettings(context.Context, model.RenderingSettings, int64) error
	RecordLocalPasswordUpdated(context.Context, model.User)
}

type sharingService interface {
	CreatePageShareLink(context.Context, string, model.User) (service.IssuedPageShareLink, error)
	PageShareLink(context.Context, string) (model.PageShareLink, error)
}

type systemService interface {
	Ping(context.Context) error
	SetupRequired(context.Context) (bool, error)
	RecordSetupCompleted(context.Context, model.User)
}

type templateService interface {
	PageTemplates(context.Context) ([]model.PageTemplate, error)
	PageTemplate(context.Context, int64) (model.PageTemplate, error)
	CreatePageTemplate(context.Context, string, string, string) (model.PageTemplate, error)
	UpdatePageTemplate(context.Context, int64, string, string, string) error
	DeletePageTemplate(context.Context, int64) error
}

type tokenService interface {
	Tokens(context.Context) ([]model.APIToken, error)
	UserTokens(context.Context, int64) ([]model.APIToken, error)
	CreateToken(context.Context, string, int64, int64, *time.Time) (model.IssuedToken, error)
	DeleteUserToken(context.Context, int64, int64) error
	DeleteToken(context.Context, int64) error
}

// User administration is split into directory, account management, and OIDC
// identity capabilities.
type userDirectoryService interface {
	User(context.Context, int64) (model.User, error)
	SearchUsers(context.Context, string, int) ([]model.User, error)
}

type userManagementService interface {
	Users(context.Context) ([]model.AdminUser, error)
	UserGroups(context.Context, int64) ([]model.Group, error)
	UpdateUser(context.Context, int64, string, bool, []int64, *bool) error
	RevokeUserSessions(context.Context, int64, int64) error
}

type oidcIdentityService interface {
	OIDCIdentities(context.Context) ([]model.OIDCIdentity, error)
	OIDCGroupMappings(context.Context) ([]model.OIDCGroupMapping, error)
	PendingOIDCIdentities(context.Context) ([]model.PendingOIDCIdentity, error)
	ApprovePendingOIDCIdentity(context.Context, int64, int64) (model.User, error)
	LinkPendingOIDCIdentity(context.Context, int64, int64, int64) (model.User, error)
	SetPendingOIDCIdentityRejected(context.Context, int64, bool, int64) error
	RemoveOIDCIdentity(context.Context, int64, string, string, int64) error
	HasLocalCredential(context.Context, int64) (bool, error)
}

type adminUserOverviewService interface {
	userManagementService
	oidcIdentityService
}
