package wasm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/tetratelabs/wazero/api"
)

type callerKey struct{}
type invocationState struct {
	instance  *Instance
	remaining int
}
type capabilitiesKey struct{}

func decode(data []byte, result any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(result); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}

func (r *Runtime) hostCall(ctx context.Context, module api.Module, pointer, length, output, capacity uint32) uint32 {
	// Validate both buffers before executing a side effect. Never let a plugin
	// probe host addresses or cause a write followed by an invalid-buffer retry.
	if length == 0 || uint64(length) > uint64(r.limits.WireBytes) || capacity < 256 || uint64(capacity) > uint64(r.limits.WireBytes) {
		return 0
	}
	input, ok := module.Memory().Read(pointer, length)
	if !ok {
		return 0
	}
	if _, ok := module.Memory().Read(output, capacity); !ok {
		return 0
	}
	response := pluginapi.CapabilityResponse{}
	value, err := plugin.Guard("host capability", func() (any, error) {
		var request pluginapi.CapabilityRequest
		if err := decode(input, &request); err != nil {
			return nil, errors.New("invalid capability request")
		}
		return r.dispatch(ctx, module, request)
	})
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Value, err = json.Marshal(value)
		if err != nil {
			response.Error = "cannot encode capability result"
		}
	}
	encoded, err := json.Marshal(response)
	if err != nil || len(encoded) > int(capacity) {
		encoded = []byte(`{"error":"capability response exceeds size limit"}`)
	}
	if len(encoded) > int(capacity) || !module.Memory().Write(output, encoded) {
		return 0
	}
	return uint32(len(encoded))
}

func (r *Runtime) dispatch(ctx context.Context, module api.Module, request pluginapi.CapabilityRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state, ok := ctx.Value(callerKey{}).(*invocationState)
	if !ok || state.instance.module != module {
		return nil, errors.New("capabilities unavailable outside invocation")
	}
	if state.remaining <= 0 {
		return nil, errors.New("host call quota exceeded")
	}
	state.remaining--
	caller := state.instance
	permission, known := pluginapi.PermissionFor(request.Method)
	if !known || (permission != "" && (!r.permissions[permission] || !slices.Contains(caller.manifest.Permissions, permission))) {
		return nil, errors.New("capability denied")
	}
	if strings.HasPrefix(request.Method, "plugin.") {
		return r.storageCall(ctx, caller.manifest.ID, request)
	}
	if request.Method == "log" {
		var message pluginapi.LogMessage
		if err := decode(request.Params, &message); err != nil || len(message.Message) > 4096 {
			return nil, errors.New("invalid log message")
		}
		slog.InfoContext(ctx, "plugin message", "plugin_id", caller.manifest.ID, "message", message.Message)
		return nil, nil
	}
	capabilities, _ := ctx.Value(capabilitiesKey{}).(map[string]plugin.Capability)
	capability := capabilities[request.Method]
	if capability == nil {
		return nil, errors.New("capability unavailable in this context")
	}
	return capability(ctx, request.Params)
}

func (r *Runtime) storageCall(ctx context.Context, id string, request pluginapi.CapabilityRequest) (any, error) {
	if r.storage == nil {
		return nil, errors.New("plugin storage unavailable")
	}
	var value pluginapi.StorageValue
	if err := decode(request.Params, &value); err != nil || len(value.Key) == 0 || len(value.Key) > 256 || strings.ContainsRune(value.Key, 0) || len(value.Value) > 64<<10 {
		return nil, errors.New("invalid storage value")
	}
	namespace := "data"
	if strings.HasPrefix(request.Method, "plugin.settings.") {
		namespace = "settings"
	}
	if strings.HasSuffix(request.Method, ".read") {
		data, found, err := r.storage.ReadPluginValue(ctx, id, namespace, value.Key)
		if len(data) > 64<<10 {
			return nil, errors.New("stored value exceeds size limit")
		}
		return pluginapi.StoredValue{Value: data, Found: found}, err
	}
	return nil, r.storage.WritePluginValue(ctx, id, namespace, value.Key, value.Value)
}
