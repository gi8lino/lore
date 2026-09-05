package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
)

type mediaKind int

const (
	imageMedia mediaKind = iota
	attachmentMedia
)

type mediaErrorText struct {
	title       string
	empty       string
	tooLarge    string
	unsupported string
	forbidden   string
	noun        string
}

var mediaErrorTexts = map[mediaKind]mediaErrorText{
	imageMedia: {
		title:       "Image validation failed.",
		empty:       "The selected image is empty.",
		tooLarge:    "Images may be at most 10 MiB.",
		unsupported: "Only JPEG, PNG, GIF, and WebP images are supported.",
		forbidden:   "You can only delete images you uploaded.",
		noun:        "Image",
	},
	attachmentMedia: {
		title:       "Attachment validation failed.",
		empty:       "The selected file is empty.",
		tooLarge:    "Files may be at most 25 MiB.",
		unsupported: "Use PDF, TXT, Markdown, JSON, YAML, CSV, LOG, TOML, or ZIP files.",
		forbidden:   "You can only delete files you uploaded.",
		noun:        "Attachment",
	},
}

// writeMediaUploadProblem translates service upload failures into HTTP problems.
func writeMediaUploadProblem(
	logger *slog.Logger,
	w http.ResponseWriter,
	err error,
	kind mediaKind,
) bool {
	if err == nil {
		return false
	}

	text := mediaErrorTexts[kind]

	switch {
	case errors.Is(err, service.ErrEmptyFile):
		httpresponse.Problem(w,
			http.StatusBadRequest,
			text.title,
			httpresponse.NewFieldProblem("file", text.empty),
		)
	case errors.Is(err, service.ErrFileTooLarge):
		httpresponse.Problem(w,
			http.StatusRequestEntityTooLarge,
			text.title,
			httpresponse.NewFieldProblem("file", text.tooLarge),
		)
	case errors.Is(err, service.ErrUnsupportedFileType):
		httpresponse.Problem(w,
			http.StatusUnsupportedMediaType,
			text.title,
			httpresponse.NewFieldProblem("file", text.unsupported),
		)
	default:
		writeUnexpectedProblem(logger, w, err)
	}

	return true
}

// writeMediaDeleteProblem translates media policy failures into HTTP problems.
func writeMediaDeleteProblem(
	logger *slog.Logger,
	w http.ResponseWriter,
	err error,
	kind mediaKind,
) bool {
	if err == nil {
		return false
	}

	text := mediaErrorTexts[kind]
	inUse, isInUse := errors.AsType[*service.MediaInUseError](err)

	switch {
	case errors.Is(err, service.ErrNotFound):
		httpresponse.Problem(w, http.StatusNotFound, text.noun+" not found.")
	case errors.Is(err, service.ErrMediaForbidden):
		httpresponse.Problem(w, http.StatusForbidden, text.forbidden)
	case isInUse:
		httpresponse.Problem(w,
			http.StatusConflict,
			fmt.Sprintf("%s is still referenced %d time(s).", text.noun, inUse.References),
		)
	default:
		writeUnexpectedProblem(logger, w, err)
	}

	return true
}

// writeValidationProblem maps application validation failures to field problems.
func writeValidationProblem(w http.ResponseWriter, err error, title string) bool {
	validation, ok := errors.AsType[*service.ValidationError](err)
	if !ok {
		return false
	}

	problems := make([]httpresponse.FieldProblem, 0, len(validation.Fields))

	for _, field := range validation.Fields {
		problems = append(problems, httpresponse.NewFieldProblem(field.Field, field.Message))
	}

	httpresponse.Problem(w, http.StatusUnprocessableEntity, title, problems...)
	return true
}

// writePageProblem translates page-domain errors into HTTP problems.
func writePageProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	if writeValidationProblem(w, err, "Page validation failed.") {
		return
	}
	switch {
	case errors.Is(err, service.ErrNotFound):
		httpresponse.Problem(w, http.StatusNotFound, "Page not found.")
	case errors.Is(err, service.ErrForbidden):
		httpresponse.Problem(w, http.StatusForbidden, "The page operation is not permitted.")
	case errors.Is(err, service.ErrPageInBin):
		httpresponse.Problem(w,
			http.StatusConflict,
			"This page path is currently in the recycle bin.",
			httpresponse.NewFieldProblem("slug", "Restore the deleted page or choose a different path."),
		)
	case errors.Is(err, service.ErrDiscussionsDisabled):
		httpresponse.Problem(w, http.StatusForbidden, "Page discussions are disabled.")
	default:
		writeUnexpectedProblem(logger, w, err)
	}
}

// writePageSaveProblem translates errors specific to creating or updating a page.
func writePageSaveProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrForbidden) {
		httpresponse.Problem(w,
			http.StatusForbidden,
			"You cannot assign this page to one or more selected groups.",
			httpresponse.NewFieldProblem(
				"group_ids",
				"One or more selected groups are not assignable by this user.",
			),
		)
		return
	}
	writePageProblem(logger, w, err)
}

// writeUnexpectedProblem logs an unexpected failure and writes a safe HTTP problem.
func writeUnexpectedProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	logger.Error("request failed", "event", "request_failed", "error", err)
	httpresponse.Problem(w,
		http.StatusInternalServerError,
		"The request could not be processed.")
}
