package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/pluginbrowser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserPluginLifecycleAndAssetBoundary(t *testing.T) {
	ctx := context.Background()
	renderer, err := md.New(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, renderer.Close(ctx)) }()
	manager := renderer.PluginManager()
	recorder := httptest.NewRecorder()
	PluginModules(manager)(recorder, httptest.NewRequest("GET", "http://lore.test/plugins/modules.json", nil))
	var modules []pluginbrowser.Module
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &modules))
	require.Len(t, modules, 2)
	module := modules[0]
	assert.Equal(t, "io.lore.mermaid", module.PluginID)
	asset := func(name, digest string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://lore.test/asset", nil)
		r.SetPathValue("pluginID", module.PluginID)
		r.SetPathValue("digest", digest)
		r.SetPathValue("asset", name)
		w := httptest.NewRecorder()
		PluginAssets(manager)(w, r)
		return w
	}
	w := asset("plugin.js", module.Digest)
	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "lorePlugin")
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	for _, name := range []string{"../plugin.wasm", "/plugin.js", "..\\plugin.js", "missing.js", "a/../plugin.js"} {
		assert.Equal(t, 404, asset(name, module.Digest).Code, name)
	}
	assert.Equal(t, 404, asset("plugin.js", "stale").Code)
	r := httptest.NewRequest("GET", "http://lore.test/", nil)
	r.SetPathValue("pluginID", module.PluginID)
	r.SetPathValue("digest", module.Digest)
	r.SetPathValue("frame", module.ModuleID+".html")
	w = httptest.NewRecorder()
	PluginFrame(manager)(w, r)
	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox allow-scripts")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "connect-src 'none'")
	assert.NotContains(t, w.Header().Get("Content-Security-Policy"), "allow-same-origin")
	assert.Contains(t, w.Body.String(), "/plugins/runtime.js")
	require.NoError(t, manager.Disable(ctx, module.PluginID))
	assert.Len(t, manager.BrowserModules(), 1)
	assert.Equal(t, 404, asset("plugin.js", module.Digest).Code)
	w = httptest.NewRecorder()
	PluginFrame(manager)(w, r)
	assert.Equal(t, 404, w.Code)
	require.NoError(t, manager.Enable(ctx, module.PluginID))
	assert.Equal(t, 200, asset("plugin.js", module.Digest).Code)
}
