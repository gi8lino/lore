package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/portable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// portableCatalogStub returns pages from an in-memory slug map.
type portableCatalogStub map[string]domain.Page

// GetPage returns one configured page.
func (s portableCatalogStub) GetPage(_ context.Context, slug string) (domain.Page, error) {
	page, ok := s[slug]
	if !ok {
		return domain.Page{}, domain.ErrNotFound
	}
	return page, nil
}

// portableExportMediaStub returns deterministic referenced and orphaned resources.
type portableExportMediaStub struct{}

// ImageContent returns one configured image payload.
func (portableExportMediaStub) ImageContent(_ context.Context, id int64) (domain.ImageData, error) {
	switch id {
	case 12:
		return domain.ImageData{Filename: "diagram.png", ContentType: "image/png", Data: []byte("image")}, nil
	case 99:
		return domain.ImageData{Filename: "orphan.png", ContentType: "image/png", Data: []byte("orphan-image")}, nil
	default:
		return domain.ImageData{}, domain.ErrNotFound
	}
}

// AttachmentContent returns one configured attachment payload.
func (portableExportMediaStub) AttachmentContent(_ context.Context, id int64) (domain.AttachmentData, error) {
	switch id {
	case 7:
		return domain.AttachmentData{
			Attachment: domain.Attachment{ID: 7, Filename: "runbook.pdf", ContentType: "application/pdf"},
			Data:       []byte("attachment"),
		}, nil
	case 88:
		return domain.AttachmentData{
			Attachment: domain.Attachment{ID: 88, Filename: "orphan.txt", ContentType: "text/plain"},
			Data:       []byte("orphan-attachment"),
		}, nil
	default:
		return domain.AttachmentData{}, domain.ErrNotFound
	}
}

// Images lists both the referenced and orphaned image for full exports.
func (portableExportMediaStub) Images(context.Context) ([]domain.Image, error) {
	return []domain.Image{
		{ID: 99, Filename: "orphan.png"},
		{ID: 12, Filename: "diagram.png"},
	}, nil
}

// Attachments lists both the referenced and orphaned attachment for full exports.
func (portableExportMediaStub) Attachments(context.Context) ([]domain.Attachment, error) {
	return []domain.Attachment{
		{ID: 88, Filename: "orphan.txt"},
		{ID: 7, Filename: "runbook.pdf"},
	}, nil
}

func TestWritePortableExportArchiveIncludesMetadataAndReferencedResources(t *testing.T) {
	t.Parallel()

	catalog := portableCatalogStub{
		"platform/runbook": {
			Slug:               "platform/runbook",
			Title:              "Runbook",
			Icon:               "book-open-lucide",
			Language:           "german",
			Markdown:           "![Diagram](/media/12/diagram.png)\n\n[Manual](/attachments/7/runbook.pdf)\n",
			Tags:               []string{"platform", "ops"},
			Groups:             []domain.Group{{ID: 3, Name: "Platform"}},
			Status:             "verified",
			OwnerGroupID:       3,
			OwnerGroup:         "Platform",
			ReviewIntervalDays: 180,
			Properties:         []domain.PageProperty{{Key: "tier", Value: "critical"}},
		},
	}

	var output bytes.Buffer
	err := writePortableExportArchive(
		context.Background(),
		catalog,
		portableExportMediaStub{},
		&output,
		[]string{"platform/runbook"},
		false,
	)
	require.NoError(t, err)

	files := readPortableTestZip(t, output.Bytes())
	assert.Contains(t, files, portable.ManifestPath)
	assert.Equal(t, "image", string(files["media/12/diagram.png"]))
	assert.Equal(t, "attachment", string(files["attachments/7/runbook.pdf"]))
	assert.NotContains(t, files, "media/99/orphan.png")
	assert.NotContains(t, files, "attachments/88/orphan.txt")
	assert.Contains(t, string(files["pages/platform/runbook.md"]), "../../media/12/diagram.png")
	assert.Contains(t, string(files["pages/platform/runbook.md"]), "../../attachments/7/runbook.pdf")

	var manifest portable.Manifest
	require.NoError(t, json.Unmarshal(files[portable.ManifestPath], &manifest))
	assert.Equal(t, portable.Format, manifest.Format)
	assert.Equal(t, portable.Version, manifest.Version)
	require.Len(t, manifest.Pages, 1)
	require.Len(t, manifest.Media, 1)
	require.Len(t, manifest.Attachments, 1)

	var metadata portable.PageMetadata
	require.NoError(t, json.Unmarshal(files["metadata/platform/runbook.json"], &metadata))
	assert.Equal(t, "platform/runbook", metadata.Slug)
	assert.Equal(t, "Runbook", metadata.Title)
	assert.Equal(t, "book-open-lucide", metadata.Icon)
	assert.Equal(t, "german", metadata.Language)
	assert.Equal(t, []string{"ops", "platform"}, metadata.Tags)
	assert.Equal(t, []string{"Platform"}, metadata.Groups)
	assert.Equal(t, "Platform", metadata.OwnerGroup)
	assert.Equal(t, 180, metadata.ReviewIntervalDays)
	assert.Equal(t, map[string]string{"tier": "critical"}, metadata.Properties)
}

func TestWritePortableExportArchiveIncludesOrphanedResourcesForFullExport(t *testing.T) {
	t.Parallel()

	catalog := portableCatalogStub{
		"guide": {
			Slug:     "guide",
			Title:    "Guide",
			Markdown: "# Guide\n",
			Status:   "verified",
		},
	}

	var output bytes.Buffer
	err := writePortableExportArchive(
		context.Background(),
		catalog,
		portableExportMediaStub{},
		&output,
		[]string{"guide"},
		true,
	)
	require.NoError(t, err)

	files := readPortableTestZip(t, output.Bytes())
	assert.Equal(t, "image", string(files["media/12/diagram.png"]))
	assert.Equal(t, "orphan-image", string(files["media/99/orphan.png"]))
	assert.Equal(t, "attachment", string(files["attachments/7/runbook.pdf"]))
	assert.Equal(t, "orphan-attachment", string(files["attachments/88/orphan.txt"]))

	var manifest portable.Manifest
	require.NoError(t, json.Unmarshal(files[portable.ManifestPath], &manifest))
	assert.Len(t, manifest.Media, 2)
	assert.Len(t, manifest.Attachments, 2)
}

func TestNextPortableResourceReferenceSkipsMalformedEarlierReference(t *testing.T) {
	t.Parallel()

	reference, ok := nextPortableResourceReference("/media/no/file then /media/12/diagram.png")

	require.True(t, ok)
	assert.Equal(t, portableMediaResource, reference.Kind)
	assert.EqualValues(t, 12, reference.ID)
}

// readPortableTestZip returns every non-directory entry from a ZIP payload.
func readPortableTestZip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	files := map[string][]byte{}
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}

		file, err := entry.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(file)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		files[entry.Name] = content
	}

	return files
}
