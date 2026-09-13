package plugin

import (
	"context"
	"encoding/json"
)

// Capability is a request-scoped, authorization-filtered host operation. It must
// respect cancellation and may return only public Plugin API values.
type Capability func(context.Context, json.RawMessage) (any, error)

// Storage is a trusted adapter. Identity and namespace are supplied by core,
// never decoded from plugin arguments.
type Storage interface {
	ReadPluginValue(context.Context, string, string, string) ([]byte, bool, error)
	WritePluginValue(context.Context, string, string, string, []byte) error
}
