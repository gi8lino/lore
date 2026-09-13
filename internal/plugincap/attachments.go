package plugincap

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/pluginapi"
)

// AttachmentReader must enforce request authorization before reading a range.
// It is supplied explicitly by the composition root; no global media store is
// exposed through the rendering pipeline.
type AttachmentReader interface {
	ReadAttachment(context.Context, pluginapi.AttachmentRead) (pluginapi.Attachment, error)
}

func Attachments(reader AttachmentReader) plugin.Capability {
	return func(ctx context.Context, data json.RawMessage) (any, error) {
		var request pluginapi.AttachmentRead
		if err := json.Unmarshal(data, &request); err != nil || request.ID <= 0 || request.Offset < 0 || request.Length < 1 || request.Length > 1<<20 {
			return nil, errors.New("invalid attachment range")
		}
		attachment, err := reader.ReadAttachment(ctx, request)
		if err != nil {
			return nil, err
		}
		if len(attachment.Data) > request.Length {
			return nil, errors.New("attachment response exceeds range")
		}
		return attachment, nil
	}
}
