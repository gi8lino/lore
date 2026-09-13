// Package pluginbrowser defines the core-owned, isolated browser module surface.
package pluginbrowser

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/gi8lino/lore/internal/plugin"
)

// Module describes one browser module exposed by an enabled plugin.
type Module struct {
	plugin.BrowserContribution
	// FrameURL is the isolated frame URL used to load this browser module.
	FrameURL string `json:"frame_url"`
}

// Base returns the public base path for a plugin's browser assets.
func Base(prefix string, m plugin.BrowserContribution) string {
	return strings.TrimRight(prefix, "/") + "/" + m.PluginID + "/" + m.Digest + "/"
}

// View builds browser-module metadata from the active plugin registry.
func View(prefix string, m plugin.BrowserContribution) Module {
	return Module{BrowserContribution: m, FrameURL: Base(prefix, m) + "frames/" + m.ModuleID + ".html"}
}

// Policy also applies when the frame is opened directly. Opaque origin and
// resource restrictions keep plugin JS away from Lore's DOM, credentials and APIs.
func Policy(origins []string, assetBase, runtimeURL string) string {
	sources := make([]string, 0, len(origins)*2)
	for _, origin := range origins {
		sources = append(sources, origin+assetBase, origin+runtimeURL)
	}

	return "default-src 'none'; script-src " + strings.Join(sources, " ") + "; style-src 'unsafe-inline' " + strings.Join(sources, " ") + "; img-src data:; font-src data:; connect-src 'none'; object-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'; sandbox allow-scripts"
}

var frameTemplate = template.Must(template.New("plugin-frame").Parse(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="{{.Policy}}"><title>{{.Name}}</title>{{if .CSS}}<link rel="stylesheet" href="{{.CSS}}">{{end}}<script defer src="{{.Runtime}}"></script></head><body data-plugin-javascript="{{.JavaScript}}"><main id="plugin-root"></main></body></html>`))

// Frame renders the isolated HTML frame used to load one plugin browser module.
func Frame(prefix, runtimeURL string, origins []string, m plugin.BrowserContribution) ([]byte, string, error) {
	base := Base(prefix, m) + "assets/"
	css := ""
	if m.CSS != "" {
		css = assetURL(base + m.CSS)
	}

	policy := Policy(origins, base, runtimeURL)
	var output bytes.Buffer
	err := frameTemplate.Execute(&output, struct{ Name, Policy, Runtime, JavaScript, CSS string }{m.Name, policy, runtimeURL, assetURL(base + m.JavaScript), css})
	if err != nil {
		return nil, "", fmt.Errorf("plugin frame: %w", err)
	}

	return output.Bytes(), policy, nil
}

// assetURL preserves package names containing URL-reserved characters.
func assetURL(name string) string { return (&url.URL{Path: name}).EscapedPath() }
