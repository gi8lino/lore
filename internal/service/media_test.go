package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mediaRepositoryStub struct {
	image          domain.Image
	imageErr       error
	deletedImageID int64
}

func (s *mediaRepositoryStub) AttachmentInfo(context.Context, int64) (domain.Attachment, error) {
	return domain.Attachment{}, nil
}

func (s *mediaRepositoryStub) AttachmentContent(context.Context, int64) (domain.AttachmentData, error) {
	return domain.AttachmentData{}, nil
}

func (s *mediaRepositoryStub) Attachments(context.Context) ([]domain.Attachment, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) DeleteAttachment(context.Context, int64) error {
	return nil
}

func (s *mediaRepositoryStub) DeleteImage(_ context.Context, id int64) error {
	s.deletedImageID = id
	return nil
}

func (s *mediaRepositoryStub) ImageInfo(context.Context, int64) (domain.Image, error) {
	return s.image, s.imageErr
}

func (s *mediaRepositoryStub) ImageContent(context.Context, int64) (domain.ImageData, error) {
	return domain.ImageData{}, nil
}

func (s *mediaRepositoryStub) Images(context.Context) ([]domain.Image, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) ImagesByUser(context.Context, int64) ([]domain.Image, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) SaveAttachment(
	context.Context,
	string,
	string,
	[]byte,
	int64,
) (domain.Attachment, error) {
	return domain.Attachment{}, nil
}

func (s *mediaRepositoryStub) SaveImage(
	context.Context,
	string,
	string,
	[]byte,
	int64,
) (domain.Image, error) {
	return domain.Image{}, nil
}

func TestDeleteImageEnforcesOwnership(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: domain.Image{ID: 7, UploadedBy: 2}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		domain.User{ID: 1, Role: "editor"},
	)

	require.ErrorIs(t, err, ErrMediaForbidden)
	assert.Zero(t, repository.deletedImageID)
}

func TestDeleteImageRejectsReferencedOwnedMedia(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: domain.Image{
		ID:         7,
		UploadedBy: 1,
		UsageCount: 3,
	}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		domain.User{ID: 1, Role: "editor"},
	)

	inUse, ok := errors.AsType[*MediaInUseError](err)

	require.True(t, ok)
	assert.Equal(t, int64(3), inUse.References)
	assert.Zero(t, repository.deletedImageID)
}

func TestDeleteImageAllowsAdministratorOverride(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: domain.Image{
		ID:         7,
		UploadedBy: 2,
		UsageCount: 3,
	}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		domain.User{ID: 1, Role: "admin"},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(7), repository.deletedImageID)
}

func TestSanitizeFilenameCharacterRuns(t *testing.T) {
	for _, tt := range []struct{ name, want string }{
		{"my   diagram.png", "my-diagram.png"},
		{"aé💡b.png", "a-b.png"},
		{"a--b.png", "a--b.png"},
		{"a - b.png", "a---b.png"},
		{"  ../some/path/image.png  ", "image.png"},
		{"._-hello_world-_.", "hello_world"},
		{"💡...", "fallback"},
		{"", "fallback"},
		{"Mixed.Case_123.txt", "Mixed.Case_123.txt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.name, "fallback"); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
