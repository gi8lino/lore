package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/model"
	"github.com/gi8lino/lore/internal/service"
)

// MediaItem contains image metadata plus its stable browser URL.
type MediaItem struct {
	// ID is the stable database identifier used in image URLs.
	ID int64 `json:"id"`
	// Filename is the sanitized image filename.
	Filename string `json:"filename"`
	// ContentType is the validated image MIME type.
	ContentType string `json:"content_type"`
	// SizeBytes is the stored image size in bytes.
	SizeBytes int64 `json:"size_bytes"`
	// UploadedBy is the identifier of the user that uploaded the image.
	UploadedBy int64 `json:"uploaded_by"`
	// Uploader is the display name of the user that uploaded the image.
	Uploader string `json:"uploader"`
	// CreatedAt is the upload timestamp formatted by templates or clients.
	CreatedAt time.Time `json:"created_at"`
	// UsageCount is the number of Markdown references to the image across all pages.
	UsageCount int64 `json:"usage_count"`
	// URL is the stable authenticated browser URL for the image.
	URL string `json:"url"`
}

// ListImages returns uploaded image metadata for editors and administrators.
func ListImages(mediaUseCases imageService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		images, err := mediaUseCases.Images(r.Context())
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, mediaItems(images))
	}
}

// UploadImage validates and stores an uploaded raster image.
func UploadImage(mediaUseCases imageService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)

		r.Body = http.MaxBytesReader(w, r.Body, service.MaxImageBytes+(1<<20))
		file, header, err := r.FormFile("file")
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Image validation failed.",
				httpresponse.NewFieldProblem("file", "Choose an image file."),
			)
			return
		}

		defer file.Close() // nolint:errcheck

		data, err := io.ReadAll(io.LimitReader(file, service.MaxImageBytes+1))
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		image, err := mediaUseCases.UploadImage(r.Context(), header.Filename, data, user)
		if writeMediaUploadProblem(logger, w, err, imageMedia) {
			return
		}

		items := mediaItems([]model.Image{image})

		httpresponse.Respond(w, http.StatusCreated, items[0])
	}
}

// ServeImage writes one stored image with immutable private caching headers.
func ServeImage(mediaUseCases imageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}

		image, err := mediaUseCases.ImageContent(r.Context(), id)
		if err != nil {
			if err == model.ErrNotFound {
				httpresponse.Problem(w, http.StatusNotFound, "Not found.")
				return
			}

			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}

		w.Header().Set("Content-Type", image.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(image.Data)))
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")

		_, _ = w.Write(image.Data)
	}
}

// DeleteImage removes an owned unused image or any image when requested by an administrator.
func DeleteImage(mediaUseCases imageService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid image identifier.")
			return
		}

		err = mediaUseCases.DeleteImage(r.Context(), id, user)
		if writeMediaDeleteProblem(logger, w, err, imageMedia) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// mediaItems converts store image metadata into browser-facing media items.
func mediaItems(images []model.Image) []MediaItem {
	items := make([]MediaItem, 0, len(images))

	for _, image := range images {
		items = append(items, MediaItem{
			ID:          image.ID,
			Filename:    image.Filename,
			ContentType: image.ContentType,
			SizeBytes:   image.SizeBytes,
			UploadedBy:  image.UploadedBy,
			Uploader:    image.Uploader,
			CreatedAt:   image.CreatedAt,
			UsageCount:  image.UsageCount,
			URL:         mediaURL(image.ID, image.Filename),
		})
	}

	return items
}

// mediaURL builds the stable authenticated URL used in Markdown image references.
func mediaURL(id int64, filename string) string {
	return "/media/" + strconv.FormatInt(id, 10) + "/" + url.PathEscape(filename)
}
