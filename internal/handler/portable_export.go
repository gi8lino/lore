package handler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/portable"
)

// portableArchiveExportMediaService exposes the media reads required by portable export.
type portableArchiveExportMediaService interface {
	ImageContent(context.Context, int64) (domain.ImageData, error)
	AttachmentContent(context.Context, int64) (domain.AttachmentData, error)
	Images(context.Context) ([]domain.Image, error)
	Attachments(context.Context) ([]domain.Attachment, error)
}

// portableResourceKind identifies the resource URL family represented by a reference.
type portableResourceKind string

const (
	portableMediaResource      portableResourceKind = "media"
	portableAttachmentResource portableResourceKind = "attachments"
)

// portableResourceReference identifies one stored resource reference in Markdown source.
type portableResourceReference struct {
	// Start is the byte offset where the resource URL begins.
	Start int
	// End is the byte offset immediately after the resource URL.
	End int
	// Kind identifies whether the reference points at an image or attachment.
	Kind portableResourceKind
	// ID is the Lore resource identifier.
	ID int64
}

// portableExportResource caches one exported binary and its archive path.
type portableExportResource struct {
	// Path is the resource's portable archive path.
	Path string
	// Filename is the portable filename restored on import.
	Filename string
}

// portableExportState tracks resources already written while pages are exported.
type portableExportState struct {
	// Archive receives portable ZIP entries.
	Archive *zip.Writer
	// Media reads referenced or complete resource payloads.
	Media portableArchiveExportMediaService
	// Images maps Lore image identifiers to already-written archive entries.
	Images map[int64]portableExportResource
	// Attachments maps Lore attachment identifiers to already-written archive entries.
	Attachments map[int64]portableExportResource
	// Manifest inventories every page and resource written to Archive.
	Manifest portable.Manifest
}

// createPortableExportArchive builds a temporary Kumbuka-compatible ZIP and returns its cleanup function.
func createPortableExportArchive(
	ctx context.Context,
	catalogUseCases pageContentService,
	mediaUseCases portableArchiveExportMediaService,
	slugs []string,
	includeAllResources bool,
) (archiveFile *os.File, modTime time.Time, cleanup func(), err error) {
	file, err := os.CreateTemp("", "lore-kumbuka-export-*.zip")
	if err != nil {
		return nil, time.Time{}, nil, err
	}

	name := file.Name()
	cleanup = func() {
		_ = file.Close()
		_ = os.Remove(name)
	}

	if err := writePortableExportArchive(ctx, catalogUseCases, mediaUseCases, file, slugs, includeAllResources); err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}

	return file, info.ModTime(), cleanup, nil
}

// writePortableExportArchive writes a Kumbuka-compatible portable archive to output.
func writePortableExportArchive(
	ctx context.Context,
	catalogUseCases pageContentService,
	mediaUseCases portableArchiveExportMediaService,
	output io.Writer,
	slugs []string,
	includeAllResources bool,
) error {
	archive := zip.NewWriter(output)
	state := &portableExportState{
		Archive:     archive,
		Media:       mediaUseCases,
		Images:      map[int64]portableExportResource{},
		Attachments: map[int64]portableExportResource{},
		Manifest:    portable.NewManifest(),
	}

	slugs = slices.Clone(slugs)
	sort.Strings(slugs)

	for _, slug := range slugs {
		pageData, err := catalogUseCases.GetPage(ctx, slug)
		if err != nil {
			_ = archive.Close()
			return err
		}

		cleanSlug := strings.Trim(path.Clean("/"+pageData.Slug), "/")
		if cleanSlug == "" || cleanSlug == "." {
			_ = archive.Close()
			return fmt.Errorf("invalid page slug %q", pageData.Slug)
		}

		markdownPath := path.Join("pages", cleanSlug+".md")
		metadataPath := path.Join("metadata", cleanSlug+".json")
		markdown, err := state.rewriteMarkdown(ctx, markdownPath, pageData.Markdown)
		if err != nil {
			_ = archive.Close()
			return err
		}

		if err := writePortableZipBytes(archive, markdownPath, []byte(markdown)); err != nil {
			_ = archive.Close()
			return err
		}
		if err := writePortableZipJSON(archive, metadataPath, portableMetadata(pageData, cleanSlug)); err != nil {
			_ = archive.Close()
			return err
		}

		state.Manifest.Pages = append(state.Manifest.Pages, portable.PageEntry{
			Slug:     cleanSlug,
			Markdown: markdownPath,
			Metadata: metadataPath,
		})
	}

	if includeAllResources {
		if err := state.exportAllResources(ctx); err != nil {
			_ = archive.Close()
			return err
		}
	}

	sort.Slice(state.Manifest.Media, func(i, j int) bool {
		return state.Manifest.Media[i].Path < state.Manifest.Media[j].Path
	})
	sort.Slice(state.Manifest.Attachments, func(i, j int) bool {
		return state.Manifest.Attachments[i].Path < state.Manifest.Attachments[j].Path
	})

	if err := writePortableZipJSON(archive, portable.ManifestPath, state.Manifest); err != nil {
		_ = archive.Close()
		return err
	}

	return archive.Close()
}

// portableMetadata converts one Lore page into metadata understood by Kumbuka's portable importer.
func portableMetadata(pageData domain.Page, slug string) portable.PageMetadata {
	tags := slices.Clone(pageData.Tags)
	sort.Strings(tags)

	groups := make([]string, 0, len(pageData.Groups))
	for _, group := range pageData.Groups {
		groups = append(groups, group.Name)
	}
	sort.Strings(groups)

	properties := make(map[string]string, len(pageData.Properties))
	for _, property := range pageData.Properties {
		properties[property.Key] = property.Value
	}

	return portable.PageMetadata{
		Slug:               slug,
		Title:              pageData.Title,
		Icon:               pageData.Icon,
		Language:           pageData.Language,
		Tags:               tags,
		Groups:             groups,
		Status:             pageData.Status,
		OwnerGroup:         pageData.OwnerGroup,
		ReviewIntervalDays: pageData.ReviewIntervalDays,
		DeprecatedTarget:   pageData.DeprecatedTarget,
		Properties:         properties,
	}
}

// rewriteMarkdown replaces Lore resource URLs with page-relative archive paths.
func (s *portableExportState) rewriteMarkdown(
	ctx context.Context,
	markdownPath, source string,
) (string, error) {
	var result strings.Builder

	for {
		reference, ok := nextPortableResourceReference(source)
		if !ok {
			result.WriteString(source)
			break
		}

		result.WriteString(source[:reference.Start])
		resource, err := s.exportResource(ctx, reference.Kind, reference.ID)
		if err != nil {
			return "", err
		}

		from := filepath.FromSlash(path.Dir(markdownPath))
		target := filepath.FromSlash(resource.Path)
		relative, err := filepath.Rel(from, target)
		if err != nil {
			return "", err
		}
		result.WriteString(filepath.ToSlash(relative))
		source = source[reference.End:]
	}

	return result.String(), nil
}

// exportAllResources adds every stored Lore upload that was not already referenced by an exported page.
func (s *portableExportState) exportAllResources(ctx context.Context) error {
	images, err := s.Media.Images(ctx)
	if err != nil {
		return fmt.Errorf("list images for portable export: %w", err)
	}
	sort.Slice(images, func(i, j int) bool { return images[i].ID < images[j].ID })
	for _, image := range images {
		if _, err := s.exportResource(ctx, portableMediaResource, image.ID); err != nil {
			return err
		}
	}

	attachments, err := s.Media.Attachments(ctx)
	if err != nil {
		return fmt.Errorf("list attachments for portable export: %w", err)
	}
	sort.Slice(attachments, func(i, j int) bool { return attachments[i].ID < attachments[j].ID })
	for _, attachment := range attachments {
		if _, err := s.exportResource(ctx, portableAttachmentResource, attachment.ID); err != nil {
			return err
		}
	}

	return nil
}

// exportResource writes one Lore resource once and returns its archive location.
func (s *portableExportState) exportResource(
	ctx context.Context,
	kind portableResourceKind,
	id int64,
) (portableExportResource, error) {
	switch kind {
	case portableMediaResource:
		if resource, ok := s.Images[id]; ok {
			return resource, nil
		}

		image, err := s.Media.ImageContent(ctx, id)
		if err != nil {
			return portableExportResource{}, &exportMediaError{cause: err}
		}
		resource := portableExportResource{
			Path:     path.Join(string(kind), strconv.FormatInt(id, 10), path.Base(image.Filename)),
			Filename: path.Base(image.Filename),
		}
		if err := writePortableZipBytes(s.Archive, resource.Path, image.Data); err != nil {
			return portableExportResource{}, err
		}
		s.Images[id] = resource
		s.Manifest.Media = append(s.Manifest.Media, portable.ResourceEntry{
			Path: resource.Path, Filename: resource.Filename,
		})
		return resource, nil

	case portableAttachmentResource:
		if resource, ok := s.Attachments[id]; ok {
			return resource, nil
		}

		attachment, err := s.Media.AttachmentContent(ctx, id)
		if err != nil {
			return portableExportResource{}, &exportAttachmentError{cause: err}
		}
		resource := portableExportResource{
			Path:     path.Join(string(kind), strconv.FormatInt(id, 10), path.Base(attachment.Filename)),
			Filename: path.Base(attachment.Filename),
		}
		if err := writePortableZipBytes(s.Archive, resource.Path, attachment.Data); err != nil {
			return portableExportResource{}, err
		}
		s.Attachments[id] = resource
		s.Manifest.Attachments = append(s.Manifest.Attachments, portable.ResourceEntry{
			Path: resource.Path, Filename: resource.Filename,
		})
		return resource, nil
	default:
		return portableExportResource{}, fmt.Errorf("unsupported portable resource kind %q", kind)
	}
}

// writePortableZipJSON encodes value as indented JSON with a trailing newline.
func writePortableZipJSON(archive *zip.Writer, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writePortableZipBytes(archive, name, data)
}

// writePortableZipBytes writes one file entry into a portable archive.
func writePortableZipBytes(archive *zip.Writer, name string, data []byte) error {
	entry, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

// nextPortableResourceReference returns the earliest valid stored image or attachment URL in source.
func nextPortableResourceReference(source string) (portableResourceReference, bool) {
	image, imageOK := nextStoredResourceReference(source, "/media/", portableMediaResource)
	attachment, attachmentOK := nextStoredResourceReference(source, "/attachments/", portableAttachmentResource)

	switch {
	case imageOK && attachmentOK:
		if image.Start <= attachment.Start {
			return image, true
		}
		return attachment, true
	case imageOK:
		return image, true
	case attachmentOK:
		return attachment, true
	default:
		return portableResourceReference{}, false
	}
}

// nextStoredResourceReference finds the first valid /kind/ID/filename reference for one resource family.
func nextStoredResourceReference(
	source, prefix string,
	kind portableResourceKind,
) (portableResourceReference, bool) {
	for offset := 0; offset < len(source); {
		index := strings.Index(source[offset:], prefix)
		if index < 0 {
			return portableResourceReference{}, false
		}

		start := offset + index
		idStart := start + len(prefix)
		idEnd := idStart
		for idEnd < len(source) && source[idEnd] >= '0' && source[idEnd] <= '9' {
			idEnd++
		}
		if idEnd == idStart || idEnd >= len(source) || source[idEnd] != '/' {
			offset = idStart
			continue
		}

		end := idEnd + 1
		for end < len(source) && !strings.ContainsRune(" \t\n\r\f)\"'", rune(source[end])) {
			end++
		}
		if end == idEnd+1 {
			offset = end
			continue
		}

		id, err := strconv.ParseInt(source[idStart:idEnd], 10, 64)
		if err != nil || id <= 0 {
			offset = end
			continue
		}

		return portableResourceReference{Start: start, End: end, Kind: kind, ID: id}, true
	}

	return portableResourceReference{}, false
}

// exportAttachmentError retains the origin of an attachment failure in a portable export.
type exportAttachmentError struct {
	// cause is the underlying attachment lookup failure.
	cause error
}

// Error returns the attachment export failure message.
func (e *exportAttachmentError) Error() string { return fmt.Sprintf("export attachment: %v", e.cause) }

// Unwrap returns the underlying attachment lookup failure.
func (e *exportAttachmentError) Unwrap() error { return e.cause }

// writePortableExportProblem translates expected portable resource failures into HTTP problems.
func writePortableExportProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	var mediaError *exportMediaError
	if errors.As(err, &mediaError) {
		if errors.Is(err, domain.ErrNotFound) {
			httpresponse.Problem(w, http.StatusNotFound, "An image referenced by this export was not found.")
			return
		}
		httpresponse.InternalServerError(logger, w, err)
		return
	}

	var attachmentError *exportAttachmentError
	if errors.As(err, &attachmentError) {
		if errors.Is(err, domain.ErrNotFound) {
			httpresponse.Problem(w, http.StatusNotFound, "An attachment referenced by this export was not found.")
			return
		}
		httpresponse.InternalServerError(logger, w, err)
		return
	}

	writePageProblem(logger, w, err)
}
