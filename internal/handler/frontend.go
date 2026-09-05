package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

// Assets serves embedded browser assets with content-versioned cache semantics.
func Assets(appFS fs.FS) http.Handler {
	files := http.FileServer(http.FS(appFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, versioned := assetPath(r.URL.Path)

		if versioned {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		request := r.Clone(r.Context())
		url := *r.URL
		url.Path = "/" + path
		request.URL = &url

		files.ServeHTTP(w, request)
	})
}

// assetPath removes the optional content-version prefix from an asset request path.
func assetPath(requestPath string) (string, bool) {
	requestPath = strings.TrimPrefix(requestPath, "/assets/")
	version, remainder, ok := strings.Cut(requestPath, "/")
	if ok && strings.HasPrefix(version, "v-") && len(version) > 2 {
		return remainder, true
	}

	return requestPath, false
}

// ServiceWorker serves the root-scoped progressive-web-app worker without long-lived caching.
func ServiceWorker(appFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(appFS, "sw.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Service-Worker-Allowed", "/")

		_, _ = w.Write(data)
	})
}
