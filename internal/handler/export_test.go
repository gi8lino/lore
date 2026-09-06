package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/gi8lino/lore/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type exportMediaStub struct {
	calls []int64
	err   error
}

func (s *exportMediaStub) ImageContent(_ context.Context, id int64) (service.ImageData, error) {
	s.calls = append(s.calls, id)
	return service.ImageData{Filename: "image.png", ContentType: "image/png", Data: []byte("image")}, s.err
}

func TestExportedMarkdownScansMediaReferences(t *testing.T) {
	media := &exportMediaStub{}
	source := `![a](/media/12/old.png) ![b](/media/12/old.png) /media/no/file /media/4/ /media/999999999999999999999/x /media/7/image.png`
	got, ids, err := exportedMarkdown(context.Background(), media, "pages/start.md", source, map[int64]service.ImageData{})
	require.NoError(t, err)
	assert.Equal(t, `![a](../media/12/image.png) ![b](../media/12/image.png) /media/no/file /media/4/ /media/999999999999999999999/x ../media/7/image.png`, got)
	assert.Equal(t, []int64{12, 7}, ids)
	assert.Equal(t, ids, referencedImageIDs(source))
	assert.Equal(t, ids, media.calls)
}

func TestExportedMarkdownReturnsLookupError(t *testing.T) {
	failure := errors.New("lookup failed")
	media := &exportMediaStub{err: failure}
	_, _, err := exportedMarkdown(context.Background(), media, "start.md", "/media/1/a /media/2/b", map[int64]service.ImageData{})
	require.ErrorIs(t, err, failure)
	assert.Equal(t, []int64{1}, media.calls)
}
