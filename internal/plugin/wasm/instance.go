package wasm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Instance serializes calls into a reactor. The gate is released before host
// Markdown rendering, so recursive blocks never re-enter a suspended guest.
type Instance struct {
	runtime  *Runtime
	compiled wazero.CompiledModule
	manifest pluginpackage.Manifest
	gate     chan struct{}
	module   api.Module
	closed   bool
}

func (i *Instance) Contributions() plugin.Contributions {
	var result plugin.Contributions
	for _, module := range i.manifest.Modules {
		if module.Type == "macro" {
			result.Macros = append(result.Macros, macroModule{rendererModule{instance: i, module: module}})
			continue
		}
		adapter := rendererModule{instance: i, module: module}
		switch module.Stage {
		case "preprocess":
			result.Preprocessors = append(result.Preprocessors, adapter)
		case "postprocess":
			result.Postprocessors = append(result.Postprocessors, adapter)
		}
	}
	return result
}

func (i *Instance) Close(ctx context.Context) error {
	// Close marks the instance unavailable after the currently executing call.
	// Guest calls always have a bounded deadline, so draining is bounded too.
	i.gate <- struct{}{}
	defer func() { <-i.gate }()
	if i.closed {
		return nil
	}
	i.closed = true
	var err error
	if i.module != nil {
		err = i.module.Close(ctx)
	}
	return errors.Join(err, i.compiled.Close(ctx))
}

func (i *Instance) invoke(ctx context.Context, request pluginapi.RenderRequest) (pluginapi.RenderResult, error) {
	ctx, cancel := context.WithTimeout(ctx, i.runtime.limits.CallTimeout)
	defer cancel()
	select {
	case i.gate <- struct{}{}:
		defer func() { <-i.gate }()
	case <-ctx.Done():
		return pluginapi.RenderResult{}, ctx.Err()
	}
	if i.closed {
		return pluginapi.RenderResult{}, errors.New("WASM plugin is closed")
	}
	if i.module == nil || i.module.IsClosed() {
		if err := i.instantiate(ctx); err != nil {
			return pluginapi.RenderResult{}, err
		}
	}
	ctx = context.WithValue(ctx, callerKey{}, &invocationState{instance: i, remaining: 512})
	result, err := i.call(ctx, request)
	if err != nil {
		// Discard a trapped or malformed reactor. A later request gets a clean
		// instance; no partial output or poisoned memory reaches another request.
		_ = i.module.Close(context.Background())
		i.module = nil
	}
	return result, err
}

func (i *Instance) call(ctx context.Context, request pluginapi.RenderRequest) (pluginapi.RenderResult, error) {
	var result pluginapi.RenderResult
	input, err := json.Marshal(request)
	if err != nil {
		return result, err
	}
	if len(input) > i.runtime.limits.WireBytes {
		return result, errors.New("plugin request exceeds size limit")
	}
	allocated, err := i.module.ExportedFunction("lore_alloc").Call(ctx, uint64(len(input)))
	if err != nil {
		return result, fmt.Errorf("allocate plugin request: %w", err)
	}
	pointer := uint32(allocated[0])
	if pointer == 0 || !i.module.Memory().Write(pointer, input) {
		return result, errors.New("plugin returned invalid request memory")
	}
	output, err := i.module.ExportedFunction("lore_transform").Call(ctx, uint64(pointer), uint64(len(input)))
	if err != nil {
		return result, fmt.Errorf("call plugin: %w", err)
	}
	pointer, length := uint32(output[0]), uint32(output[0]>>32)
	if length == 0 || uint64(length) > uint64(i.runtime.limits.WireBytes) {
		return result, errors.New("plugin response exceeds size limit or is empty")
	}
	memory, ok := i.module.Memory().Read(pointer, length)
	if !ok {
		return result, errors.New("plugin returned invalid response memory")
	}
	decoder := json.NewDecoder(bytes.NewReader(memory))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, fmt.Errorf("decode plugin response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return result, errors.New("plugin response must contain one JSON value")
	}
	if result.Error != "" {
		return result, fmt.Errorf("plugin returned error: %.1024s", result.Error)
	}
	if len(result.Parts) > i.runtime.limits.Parts {
		return result, errors.New("plugin returned too many fragments")
	}
	for _, part := range result.Parts {
		if part.Markdown != nil && (part.Text != "" || request.Stage != "preprocess") {
			return result, errors.New("invalid plugin render fragment")
		}
	}
	return result, nil
}

type rendererModule struct {
	instance *Instance
	module   pluginpackage.Module
}

func (m rendererModule) Preprocess(ctx plugin.Context, source string) (string, error) {
	return m.render(ctx, source)
}
func (m rendererModule) Postprocess(ctx plugin.Context, source string) (string, error) {
	return m.render(ctx, source)
}

func (m rendererModule) render(ctx plugin.Context, source string) (string, error) {
	if enabled, configured := ctx.Features[m.instance.manifest.ID]; configured && !enabled {
		return source, nil
	}
	execution := ctx.Context
	if execution == nil {
		execution = context.Background()
	}
	execution = context.WithValue(execution, capabilitiesKey{}, ctx.Capabilities)
	result, err := m.instance.invoke(execution, pluginapi.RenderRequest{
		APIVersion: pluginapi.Version, Module: m.module.ID, Stage: m.module.Stage, Source: source, Features: ctx.Features,
	})
	if err != nil {
		return "", err
	}
	var output strings.Builder
	for _, part := range result.Parts {
		if err := execution.Err(); err != nil {
			return "", err
		}
		content := part.Text
		if part.Markdown != nil {
			if ctx.RenderMarkdown == nil {
				return "", errors.New("markdown rendering capability is unavailable")
			}
			content, err = ctx.RenderMarkdown(*part.Markdown)
			if err != nil {
				return "", err
			}
		}
		if len(content) > m.instance.runtime.limits.WireBytes-output.Len() {
			return "", errors.New("rendered plugin output exceeds size limit")
		}
		output.WriteString(content)
	}
	return output.String(), nil
}
