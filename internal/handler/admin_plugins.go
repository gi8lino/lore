package handler

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/pluginpackage"
)

// AdminPlugins exposes package metadata and lifecycle operations through the
// existing administration layout. Routes apply browser authentication/admin authorization.
type AdminPlugins struct {
	manager *plugin.Manager
	data    viewDataService
	views   *Views
}

func NewAdminPlugins(manager *plugin.Manager, data viewDataService, views *Views) *AdminPlugins {
	return &AdminPlugins{manager: manager, data: data, views: views}
}

// List renders the plugin inventory and optionally opens one plugin detail modal.
func (a *AdminPlugins) List(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, strings.TrimSpace(r.URL.Query().Get("plugin")), http.StatusOK, "")
}

func (a *AdminPlugins) render(w http.ResponseWriter, r *http.Request, id string, status int, message string) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.manager == nil {
		http.Error(w, "Plugin manager unavailable.", http.StatusServiceUnavailable)
		return
	}
	data, err := administrationData(r, a.data, a.views, "Plugins", "plugins")
	if err != nil {
		httpresponse.InternalServerError(a.views.logger, w, err)
		return
	}
	data.AdminPlugins = a.manager.Plugins()
	data.PluginRequiredIDs = make(map[string]bool, len(data.AdminPlugins))
	data.PluginHasSettings = make(map[string]bool, len(data.AdminPlugins))
	data.PluginREADMEs = make(map[string]template.HTML, len(data.AdminPlugins))
	foundOpenPlugin := id == ""
	for _, item := range data.AdminPlugins {
		pluginID := item.Manifest.ID
		data.PluginRequiredIDs[pluginID] = a.manager.IsRequired(pluginID)
		for _, module := range item.Manifest.Modules {
			if module.Type == "settings" {
				data.PluginHasSettings[pluginID] = true
				break
			}
		}
		readme, renderErr := renderPluginREADME(item.README)
		if renderErr != nil {
			httpresponse.InternalServerError(a.views.logger, w, renderErr)
			return
		}
		data.PluginREADMEs[pluginID] = readme
		if pluginID == id {
			foundOpenPlugin = true
		}
	}
	if !foundOpenPlugin {
		http.NotFound(w, r)
		return
	}
	sort.Slice(data.AdminPlugins, func(i, j int) bool { return data.AdminPlugins[i].Manifest.Name < data.AdminPlugins[j].Manifest.Name })
	data.OpenPluginID = id
	data.PluginMessage = message
	renderStatus(a.views, w, status, "admin_plugins", data)
}

func (a *AdminPlugins) Install(w http.ResponseWriter, r *http.Request) {
	if a.manager == nil {
		http.Error(w, "Plugin manager unavailable.", http.StatusServiceUnavailable)
		return
	}
	archive, status, err := readPluginUpload(w, r)
	if err != nil {
		a.render(w, r, "", status, "Upload failed: "+err.Error()+".")
		return
	}
	if _, err = pluginpackage.Read(archive); err != nil {
		a.render(w, r, "", http.StatusUnprocessableEntity, "Invalid plugin package: "+err.Error())
		return
	}
	item, err := a.manager.Install(r.Context(), archive)
	if err != nil {
		a.failure(w, r, "", "install", err)
		return
	}
	a.audit(r, "install", item.Manifest.ID)
	http.Redirect(w, r, "/admin/plugins?plugin="+item.Manifest.ID, http.StatusSeeOther)
}

func (a *AdminPlugins) Action(w http.ResponseWriter, r *http.Request) {
	if a.manager == nil {
		http.Error(w, "Plugin manager unavailable.", http.StatusServiceUnavailable)
		return
	}
	id, action := r.PathValue("pluginID"), r.PathValue("action")
	var err error
	switch action {
	case "enable":
		err = a.manager.Enable(r.Context(), id)
	case "disable":
		err = a.manager.Disable(r.Context(), id)
	case "settings":
		err = a.updateSettings(r, id)
	case "uninstall":
		err = a.manager.Uninstall(r.Context(), id)
	case "upgrade":
		var archive []byte
		var status int
		archive, status, err = readPluginUpload(w, r)
		if err != nil {
			a.render(w, r, pluginDetailID(r, id), status, "Upload failed: "+err.Error()+".")
			return
		}
		pkg, parseErr := pluginpackage.Read(archive)
		if parseErr != nil {
			a.render(w, r, pluginDetailID(r, id), http.StatusUnprocessableEntity, "Invalid plugin package: "+parseErr.Error())
			return
		}
		if pkg.Manifest().ID != id {
			a.render(w, r, pluginDetailID(r, id), http.StatusUnprocessableEntity, "The uploaded package must have the same plugin ID.")
			return
		}
		_, err = a.manager.Upgrade(r.Context(), id, archive)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.failure(w, r, id, action, err)
		return
	}
	a.audit(r, action, id)
	destination := "/admin/plugins"
	if action != "uninstall" && pluginDetailID(r, id) != "" {
		destination += "?plugin=" + id
	}
	http.Redirect(w, r, destination, http.StatusSeeOther)
}

// updateSettings persists every declared boolean setting for one plugin.
func (a *AdminPlugins) updateSettings(r *http.Request, id string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	var selected *plugin.LoadedPlugin
	for _, item := range a.manager.Plugins() {
		if item.Manifest.ID == id {
			copy := item
			selected = &copy
			break
		}
	}
	if selected == nil {
		return errors.New("plugin is not installed")
	}

	settings := make(map[string]bool)
	for _, module := range selected.Manifest.Modules {
		if module.Type == "settings" {
			settings[module.ID] = r.Form.Has("setting_" + module.ID)
		}
	}

	return a.manager.UpdateSettings(r.Context(), id, settings)
}

// renderPluginREADME renders package documentation without activating plugin macros.
func renderPluginREADME(source string) (template.HTML, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "# ") {
		if _, rest, ok := strings.Cut(source, "\n"); ok {
			source = strings.TrimSpace(rest)
		}
	}

	renderer := md.NewWithRegistry(nil)
	rendered, err := renderer.Render(source)
	if err != nil {
		return "", err
	}
	return template.HTML(rendered), nil
}

func (a *AdminPlugins) failure(w http.ResponseWriter, r *http.Request, id, action string, err error) {
	a.views.logger.Error("plugin administration failed", "action", action, "plugin_id", id, "error", err)
	// Runtime/storage errors can contain implementation details. Keep them in logs.
	message := "Could not " + action + " the plugin. Check its dependencies, requested permissions, and system-plugin restrictions. The existing plugin state was preserved."
	if action == "settings" {
		message = "Could not save plugin settings. Check the setting dependencies and try again."
	}
	a.render(w, r, pluginDetailID(r, id), http.StatusUnprocessableEntity, message)
}

// pluginDetailID returns id when the request originated from a plugin detail modal.
func pluginDetailID(r *http.Request, id string) string {
	if r.URL.Query().Get("return") == "detail" {
		return id
	}
	return ""
}
func (a *AdminPlugins) audit(r *http.Request, action, id string) {
	a.views.logger.Info("plugin lifecycle changed", "event", "plugin."+action, "plugin_id", id, "actor_id", currentUser(r).ID)
}

// readPluginUpload streams one bounded package without temporary files or extraction.
func readPluginUpload(w http.ResponseWriter, r *http.Request) ([]byte, int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, pluginpackage.MaxArchiveBytes+(64<<10))
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("choose a .loreplugin package to upload")
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "package" || part.FileName() == "" {
		return nil, http.StatusBadRequest, errors.New("upload exactly one plugin package")
	}
	content, err := io.ReadAll(io.LimitReader(part, pluginpackage.MaxArchiveBytes+1))
	if len(content) > pluginpackage.MaxArchiveBytes {
		return nil, http.StatusRequestEntityTooLarge, errors.New("plugin packages must be 16 MiB or smaller")
	}
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("could not read the plugin package")
	}
	if _, err := reader.NextPart(); err != io.EOF {
		return nil, http.StatusBadRequest, errors.New("upload exactly one plugin package")
	}
	return content, http.StatusOK, nil
}
