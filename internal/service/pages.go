package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/icons"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/revision"
)

// FieldError describes one invalid application input field.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError collects application-level input failures.
type ValidationError struct {
	Fields []FieldError
}

// Error returns the first validation failure or a generic fallback.
func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	return e.Fields[0].Message
}

// newValidationError creates a single-field validation error.
func newValidationError(field, message string) error {
	return &ValidationError{Fields: []FieldError{{Field: field, Message: message}}}
}

// PageSaveInput contains transport-independent page mutation fields.
type PageSaveInput struct {
	PreviousSlug       string
	Slug               string
	Title              string
	Icon               string
	Language           string
	Markdown           string
	Message            string
	Tags               []string
	GroupIDs           []int64
	Status             string
	OwnerGroupID       int64
	ReviewIntervalDays int
	MarkReviewed       bool
	DeprecatedTarget   string
	Properties         map[string]string
	Actor              domain.User
}

// ImportedPage contains one transport-independent page discovered by an importer.
type ImportedPage struct {
	Slug     string
	Title    string
	Markdown string
	Source   string
}

// BulkPageInput contains one mutation to apply to a set of pages.
type BulkPageInput struct {
	Action  string
	Slugs   []string
	Status  string
	Tag     string
	GroupID int64
	Target  string
	Actor   domain.User
}

// pageRepository is the persistence contract required by page use cases.
// Keeping it here makes the application service independently testable and
// prevents unrelated store capabilities from becoming implicit dependencies.
type pageRepository interface {
	AddPageComment(context.Context, string, int64, string, string) (domain.PageComment, error)
	ApplicationSettings(context.Context) (domain.ApplicationSettings, error)
	BulkAddPageTag(context.Context, []string, string) error
	BulkAssignPageGroup(context.Context, []string, int64) error
	BulkDeletePages(context.Context, []string, int64) error
	BulkSetPageStatus(context.Context, []string, string) error
	DeletePage(context.Context, string, int64) error
	GetPage(context.Context, string) (domain.Page, error)
	LogAudit(context.Context, int64, string, string, string, string) error
	MarkPageReviewed(context.Context, string) error
	MovePage(context.Context, string, string, domain.MovePageOptions, domain.User) error
	NotifyMentions(context.Context, int64, string, string, string) error
	ResolvePageComment(context.Context, int64, bool) error
	Revision(context.Context, string, int) (revision.Revision, error)
	SavePage(context.Context, string, string, string, string, string, string, string, []string, []string, []int64, domain.PageMetadata, map[string]string, domain.User) (domain.Page, error)
}

// Pages coordinates page mutations and their application-level side effects.
type Pages struct {
	repository pageRepository
}

// NewPages constructs the page application service.
func NewPages(repository pageRepository) *Pages {
	return &Pages{repository: repository}
}

// Save validates and persists a page, then records audit and mention side effects.
func (s *Pages) Save(ctx context.Context, input PageSaveInput) (domain.Page, error) {
	page, err := s.save(ctx, input)
	if err != nil {
		return domain.Page{}, err
	}

	action := "page.updated"

	if input.PreviousSlug == "" {
		action = "page.created"
	} else if strings.TrimSpace(input.PreviousSlug) != page.Slug {
		action = "page.renamed"
	}

	_ = s.repository.LogAudit(ctx, input.Actor.ID, action, "page", page.Slug, page.Title)
	_ = s.repository.NotifyMentions(
		ctx,
		input.Actor.ID,
		input.Markdown,
		"Mention in "+page.Title,
		"/pages/"+page.Slug,
	)

	return page, nil
}

// validPageWorkflowSettings reports whether page lifecycle and review metadata are internally valid.
func validPageWorkflowSettings(input PageSaveInput) bool {
	if !domain.ValidPageStatus(input.Status) {
		return false
	}
	if !domain.ValidReviewIntervalDays(input.ReviewIntervalDays) {
		return false
	}

	return input.OwnerGroupID >= 0
}

// save validates and persists a page without emitting side effects.
func (s *Pages) save(ctx context.Context, input PageSaveInput) (domain.Page, error) {
	input.PreviousSlug = strings.TrimSpace(input.PreviousSlug)
	input.Slug = md.Slug(input.Slug)
	input.Title = strings.TrimSpace(input.Title)
	input.Icon = strings.TrimSpace(input.Icon)
	input.Language = strings.TrimSpace(input.Language)
	input.DeprecatedTarget = md.Slug(input.DeprecatedTarget)
	validation := &ValidationError{}

	if input.Slug == "" {
		validation.Fields = append(validation.Fields, FieldError{Field: "slug", Message: "A page path is required."})
	}
	if input.Title == "" {
		validation.Fields = append(validation.Fields, FieldError{Field: "title", Message: "Title is required."})
	}
	if !icons.IsNavigationIcon(input.Icon) {
		validation.Fields = append(validation.Fields, FieldError{
			Field:   "icon",
			Message: "Choose an icon from the available Lucide icons.",
		})
	}
	if input.Language != "" && !validContentLanguage(input.Language) {
		validation.Fields = append(validation.Fields, FieldError{
			Field:   "language",
			Message: "Choose a supported content language.",
		})
	}
	if !validPageWorkflowSettings(input) {
		validation.Fields = append(validation.Fields, FieldError{
			Field:   "status",
			Message: "Choose valid page workflow settings.",
		})
	}

	if len(validation.Fields) > 0 {
		return domain.Page{}, validation
	}

	return s.repository.SavePage(
		ctx,
		input.PreviousSlug,
		input.Slug,
		input.Title,
		input.Icon,
		input.Language,
		input.Markdown,
		input.Message,
		input.Tags,
		md.Links(input.Markdown),
		input.GroupIDs,
		domain.PageMetadata{
			Status:             input.Status,
			OwnerGroupID:       input.OwnerGroupID,
			ReviewIntervalDays: input.ReviewIntervalDays,
			MarkReviewed:       input.MarkReviewed,
			DeprecatedTarget:   input.DeprecatedTarget,
		},
		input.Properties,
		input.Actor,
	)
}

// Delete moves a page to the recycle bin and records the action.
func (s *Pages) Delete(ctx context.Context, slug string, actor domain.User) error {
	slug = strings.TrimSpace(slug)
	if err := s.repository.DeletePage(ctx, slug, actor.ID); err != nil {
		return err
	}

	_ = s.repository.LogAudit(ctx, actor.ID, "page.deleted", "page", slug, "Moved page to recycle bin")

	return nil
}

// Move moves a page or subtree and records the action.
func (s *Pages) Move(
	ctx context.Context,
	oldSlug, newSlug string,
	options domain.MovePageOptions,
	actor domain.User,
) error {
	oldSlug = strings.TrimSpace(oldSlug)
	newSlug = md.Slug(newSlug)
	if oldSlug == "" || newSlug == "" {
		return &ValidationError{Fields: []FieldError{{Field: "slug", Message: "A destination path is required."}}}
	}
	if err := s.repository.MovePage(ctx, oldSlug, newSlug, options, actor); err != nil {
		return err
	}

	_ = s.repository.LogAudit(ctx, actor.ID, "page.moved", "page", newSlug, oldSlug+" → "+newSlug)

	return nil
}

// Review records a completed documentation review and its audit event.
func (s *Pages) Review(ctx context.Context, slug string, actor domain.User) error {
	slug = strings.TrimSpace(slug)
	if err := s.repository.MarkPageReviewed(ctx, slug); err != nil {
		return err
	}

	_ = s.repository.LogAudit(ctx, actor.ID, "page.reviewed", "page", slug, "Documentation review completed")

	return nil
}

// RestoreRevision creates a new page revision from a persisted historical revision.
func (s *Pages) RestoreRevision(ctx context.Context, slug string, number int, actor domain.User) (domain.Page, error) {
	if number <= 0 {
		return domain.Page{}, &ValidationError{Fields: []FieldError{{Field: "revision", Message: "Invalid revision."}}}
	}

	page, err := s.repository.GetPage(ctx, slug)
	if err != nil {
		return domain.Page{}, err
	}

	record, err := s.repository.Revision(ctx, slug, number)
	if err != nil {
		return domain.Page{}, err
	}

	groupIDs := make([]int64, 0, len(page.Groups))

	for _, group := range page.Groups {
		groupIDs = append(groupIDs, group.ID)
	}

	properties := make(map[string]string, len(page.Properties))

	for _, property := range page.Properties {
		properties[property.Key] = property.Value
	}

	page, err = s.save(ctx, PageSaveInput{
		PreviousSlug:       page.Slug,
		Slug:               page.Slug,
		Title:              page.Title,
		Icon:               page.Icon,
		Language:           page.Language,
		Markdown:           record.Markdown,
		Message:            "Restore revision " + fmt.Sprint(number),
		Tags:               page.Tags,
		GroupIDs:           groupIDs,
		Status:             page.Status,
		OwnerGroupID:       page.OwnerGroupID,
		ReviewIntervalDays: page.ReviewIntervalDays,
		DeprecatedTarget:   page.DeprecatedTarget,
		Properties:         properties,
		Actor:              actor,
	})
	if err != nil {
		return domain.Page{}, err
	}

	_ = s.repository.LogAudit(
		ctx,
		actor.ID,
		"page.revision_restored",
		"page",
		page.Slug,
		"Restored revision "+fmt.Sprint(number),
	)

	return page, nil
}

// AddComment adds a discussion comment and emits mention notifications.
func (s *Pages) AddComment(ctx context.Context, slug, anchor, body string, actor domain.User) error {
	settings, err := s.repository.ApplicationSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.DiscussionsEnabled {
		return ErrDiscussionsDisabled
	}

	slug = strings.TrimSpace(slug)
	body = strings.TrimSpace(body)
	if _, err := s.repository.AddPageComment(ctx, slug, actor.ID, anchor, body); err != nil {
		return err
	}

	_ = s.repository.NotifyMentions(ctx, actor.ID, body, "Mention in "+slug, "/pages/"+slug+"#comments")

	return nil
}

// ResolveComment changes one discussion's resolution state.
func (s *Pages) ResolveComment(ctx context.Context, id int64, resolved bool) error {
	if id <= 0 {
		return &ValidationError{Fields: []FieldError{{Field: "comment", Message: "Invalid comment."}}}
	}

	settings, err := s.repository.ApplicationSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.DiscussionsEnabled {
		return ErrDiscussionsDisabled
	}

	return s.repository.ResolvePageComment(ctx, id, resolved)
}

// Import persists imported pages while retaining workflow metadata on replacements.
func (s *Pages) Import(ctx context.Context, candidates []ImportedPage, format string, actor domain.User) (int, error) {
	for _, candidate := range candidates {
		if err := s.importPage(ctx, candidate, actor); err != nil {
			return 0, err
		}
	}

	_ = s.repository.LogAudit(
		ctx,
		actor.ID,
		"pages.imported",
		"import",
		format,
		fmt.Sprintf("Imported %d pages", len(candidates)),
	)

	return len(candidates), nil
}

// importPage persists one import candidate while retaining existing metadata.
func (s *Pages) importPage(ctx context.Context, candidate ImportedPage, actor domain.User) error {
	slug := md.Slug(candidate.Slug)
	if slug == "" {
		return newValidationError("slug", fmt.Sprintf("Invalid imported page path %q.", candidate.Slug))
	}

	input := PageSaveInput{
		Slug:       slug,
		Title:      candidate.Title,
		Markdown:   candidate.Markdown,
		Message:    "Imported from " + candidate.Source,
		Status:     "verified",
		Properties: map[string]string{},
		Actor:      actor,
	}
	current, err := s.repository.GetPage(ctx, slug)

	if err == nil {
		input.Icon = current.Icon
		input.Language = current.Language
		input.Tags = current.Tags
		input.Status = current.Status
		input.OwnerGroupID = current.OwnerGroupID
		input.ReviewIntervalDays = current.ReviewIntervalDays
		input.DeprecatedTarget = current.DeprecatedTarget
		input.GroupIDs = make([]int64, 0, len(current.Groups))

		for _, group := range current.Groups {
			input.GroupIDs = append(input.GroupIDs, group.ID)
		}
		for _, property := range current.Properties {
			input.Properties[property.Key] = property.Value
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	_, err = s.save(ctx, input)

	return err
}

// Bulk applies one administrative mutation and records a single audit event.
func (s *Pages) Bulk(ctx context.Context, input BulkPageInput) error {
	if len(input.Slugs) == 0 {
		return newValidationError("pages", "Select at least one page.")
	}

	var err error

	switch input.Action {
	case "status":
		if !domain.ValidPageStatus(input.Status) {
			return newValidationError("status", "Choose a valid page status.")
		}
		err = s.repository.BulkSetPageStatus(ctx, input.Slugs, input.Status)
	case "tag":
		if strings.TrimSpace(input.Tag) == "" {
			return newValidationError("tag", "Enter a tag.")
		}
		err = s.repository.BulkAddPageTag(ctx, input.Slugs, input.Tag)
	case "group":
		if input.GroupID <= 0 {
			return newValidationError("group_id", "Choose a valid group.")
		}
		err = s.repository.BulkAssignPageGroup(ctx, input.Slugs, input.GroupID)
	case "move":
		err = s.bulkMove(ctx, input.Slugs, input.Target, input.Actor)
	case "delete":
		err = s.repository.BulkDeletePages(ctx, input.Slugs, input.Actor.ID)
	default:
		return newValidationError("action", "Choose a valid bulk action.")
	}

	if err != nil {
		return err
	}

	_ = s.repository.LogAudit(
		ctx,
		input.Actor.ID,
		"page.bulk_"+input.Action,
		"page",
		strings.Join(input.Slugs, ","),
		fmt.Sprintf("%d pages", len(input.Slugs)),
	)

	return nil
}

// bulkMove relocates selected pages beneath a normalized target path.
func (s *Pages) bulkMove(ctx context.Context, slugs []string, target string, actor domain.User) error {
	target = md.Slug(target)
	if target == "" {
		return newValidationError("target", "A target path is required.")
	}

	orderedSlugs := slices.Clone(slugs)

	slices.SortFunc(orderedSlugs, compareMoveSlugs)

	for _, slug := range orderedSlugs {
		destination := strings.Trim(target, "/") + "/" + path.Base(slug)
		if err := s.repository.MovePage(
			ctx,
			slug,
			destination,
			domain.MovePageOptions{UpdateIncomingLinks: true, KeepAliases: true},
			actor,
		); err != nil {
			return err
		}
	}

	return nil
}

// ErrDiscussionsDisabled indicates that page discussions are globally disabled.
var ErrDiscussionsDisabled = errors.New("page discussions are disabled")

// validContentLanguage reports whether value is supported by PostgreSQL search configuration.
func validContentLanguage(value string) bool {
	switch value {
	case "arabic", "chinese", "danish", "dutch", "english", "finnish", "french", "german", "greek",
		"hungarian", "italian", "japanese", "norwegian", "portuguese", "romanian", "russian", "spanish",
		"swedish", "turkish":
		return true
	default:
		return false
	}
}

// compareMoveSlugs puts longer paths first so descendants move before ancestors.
func compareMoveSlugs(left, right string) int {
	return cmp.Compare(len(right), len(left))
}
