package handler

import (
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/pluginbrowser"
)

// PluginModules returns browser-module metadata for currently enabled plugins.
func PluginModules(manager *plugin.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		result := make([]pluginbrowser.Module, 0)
		if manager != nil {
			for _, module := range manager.BrowserModules() {
				result = append(result, pluginbrowser.View("/plugins", module))
			}
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}

// PluginAssets serves one validated browser asset from an enabled plugin.
func PluginAssets(manager *plugin.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if manager == nil {
			http.NotFound(w, r)
			return
		}
		name := r.PathValue("asset")
		// Asset() also validates the complete path, including backslashes and '..'.
		data, err := manager.BrowserAsset(r.PathValue("pluginID"), r.PathValue("digest"), name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		_, _ = w.Write(data)
	}
}

var browserHost = regexp.MustCompile(`^[a-zA-Z0-9.\[\]:_-]+$`)

// PluginFrame serves the isolated frame used to execute one plugin browser module.
func PluginFrame(manager *plugin.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if manager == nil || !browserHost.MatchString(r.Host) {
			http.NotFound(w, r)
			return
		}
		for _, module := range manager.BrowserModules() {
			if module.PluginID != r.PathValue("pluginID") || module.Digest != r.PathValue("digest") || module.ModuleID+".html" != r.PathValue("frame") {
				continue
			}
			data, policy, err := pluginbrowser.Frame("/plugins", "/plugins/runtime.js", []string{"http://" + r.Host, "https://" + r.Host}, module)
			if err != nil {
				http.Error(w, "Plugin frame unavailable", 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", policy+"; frame-ancestors 'self'")
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Referrer-Policy", "no-referrer")
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	}
}

// PluginBrowserRuntime serves the shared browser bootstrap used by plugin frames.
func PluginBrowserRuntime(appFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(appFS, "js/plugins/frame.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(data)
	}
}

// PluginPresentationStyles serves core-filtered plugin presentation styles for rendered content.
func PluginPresentationStyles(manager *plugin.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte(pluginbrowser.PresentationStyles(manager)))
	}
}
