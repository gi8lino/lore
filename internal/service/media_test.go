package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mediaRepositoryStub struct {
	image          Image
	imageErr       error
	deletedImageID int64
}

func (s *mediaRepositoryStub) AttachmentInfo(context.Context, int64) (Attachment, error) {
	return Attachment{}, nil
}

func (s *mediaRepositoryStub) AttachmentContent(context.Context, int64) (AttachmentData, error) {
	return AttachmentData{}, nil
}

func (s *mediaRepositoryStub) Attachments(context.Context) ([]Attachment, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) DeleteAttachment(context.Context, int64) error {
	return nil
}

func (s *mediaRepositoryStub) DeleteImage(_ context.Context, id int64) error {
	s.deletedImageID = id
	return nil
}

func (s *mediaRepositoryStub) ImageInfo(context.Context, int64) (Image, error) {
	return s.image, s.imageErr
}

func (s *mediaRepositoryStub) ImageContent(context.Context, int64) (ImageData, error) {
	return ImageData{}, nil
}

func (s *mediaRepositoryStub) Images(context.Context) ([]Image, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) ImagesByUser(context.Context, int64) ([]Image, error) {
	return nil, nil
}

func (s *mediaRepositoryStub) SaveAttachment(
	context.Context,
	string,
	string,
	[]byte,
	int64,
) (Attachment, error) {
	return Attachment{}, nil
}

func (s *mediaRepositoryStub) SaveImage(
	context.Context,
	string,
	string,
	[]byte,
	int64,
) (Image, error) {
	return Image{}, nil
}

func TestDeleteImageEnforcesOwnership(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: Image{ID: 7, UploadedBy: 2}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		User{ID: 1, Role: "editor"},
	)

	require.ErrorIs(t, err, ErrMediaForbidden)
	assert.Zero(t, repository.deletedImageID)
}

func TestDeleteImageRejectsReferencedOwnedMedia(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: Image{
		ID:         7,
		UploadedBy: 1,
		UsageCount: 3,
	}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		User{ID: 1, Role: "editor"},
	)

	inUse, ok := errors.AsType[*MediaInUseError](err)

	require.True(t, ok)
	assert.Equal(t, int64(3), inUse.References)
	assert.Zero(t, repository.deletedImageID)
}

func TestDeleteImageAllowsAdministratorOverride(t *testing.T) {
	t.Parallel()

	repository := &mediaRepositoryStub{image: Image{
		ID:         7,
		UploadedBy: 2,
		UsageCount: 3,
	}}
	err := NewMedia(repository).DeleteImage(
		context.Background(),
		7,
		User{ID: 1, Role: "admin"},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(7), repository.deletedImageID)
}
