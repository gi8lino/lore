package wasm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/pluginapi"
)

type macroModule struct{ rendererModule }

func (m macroModule) Name() string { return m.module.Name }
func (m macroModule) Available(ctx plugin.Context) bool {
	if enabled, configured := ctx.Features[m.instance.manifest.ID]; configured && !enabled {
		return false
	}
	return m.module.Capability == "" || ctx.Capabilities[m.module.Capability] != nil
}
func (m macroModule) Parse(string) (plugin.Invocation, bool) { return nil, false }
func (m macroModule) ParseContext(ctx plugin.Context, line string) (plugin.Invocation, bool, error) {
	if !strings.HasPrefix(strings.TrimSpace(line), "{{"+m.Name()) {
		return nil, false, nil
	}
	result, err := m.invoke(ctx, "parse", line, nil)
	return result.Invocation, result.Matched, err
}
func (m macroModule) Render(ctx plugin.Context, invocation plugin.Invocation) (string, error) {
	result, err := m.invoke(ctx, "macro", "", invocation)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	for _, part := range result.Parts {
		output.WriteString(part.Text)
	}
	return output.String(), nil
}
func (m macroModule) invoke(ctx plugin.Context, stage, source string, invocation json.RawMessage) (pluginapi.RenderResult, error) {
	execution := ctx.Context
	if execution == nil {
		execution = context.Background()
	}
	execution = context.WithValue(execution, capabilitiesKey{}, ctx.Capabilities)
	return m.instance.invoke(execution, pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: m.module.ID, Stage: stage, Source: source, Invocation: invocation})
}
