package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/lore/internal/httpresponse"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/service"
	"github.com/gi8lino/lore/themes"
)

// navigationStateRequest contains the expanded folder paths persisted by the sidebar.
type navigationStateRequest struct {
	// Expanded contains navigation folder slugs that are currently open.
	Expanded []string `json:"expanded"`
}

// sidebarWidthRequest contains a desktop sidebar width selected by dragging the resize handle.
type sidebarWidthRequest struct {
	// Width is the requested sidebar width in CSS pixels.
	Width int `json:"width"`
}

// Settings renders account information and user presentation preferences.
func Settings(
	viewDataUseCases viewDataService,
	userUseCases userManagementService,
	tokenUseCases tokenService,
	mediaUseCases userImageService,
	local *auth.Local,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := viewData(r, viewDataUseCases, views, "Settings")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.Groups, err = userUseCases.UserGroups(r.Context(), data.User.ID)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		if localUser, localErr := local.Authenticate(r); localErr == nil && localUser.ID == data.User.ID {
			data.LocalCredentialAuthenticated = true
		}

		data.UserTokens, err = tokenUseCases.UserTokens(r.Context(), data.User.ID)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		if data.CanEdit {
			images, err := mediaUseCases.ImagesByUser(r.Context(), data.User.ID)
			if err != nil {
				writeUnexpectedProblem(views.logger, w, err)
				return
			}

			data.Images = mediaItems(images)
		}

		render(views, w, "settings", data)
	}
}

// ChangeLocalPassword lets a locally authenticated user replace their own password.
func ChangeLocalPassword(local *auth.Local, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		localUser, err := local.Authenticate(r)
		if err != nil || localUser.ID != user.ID {
			httpresponse.Problem(w, http.StatusForbidden, "A local-password session is required to change this password.")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid password form.")
			return
		}

		currentPassword := r.FormValue("current_password")
		if currentPassword == "" {
			httpresponse.Problem(
				w,
				http.StatusUnprocessableEntity,
				"Password validation failed.",
				httpresponse.NewFieldProblem("current_password", "Enter your current password."),
			)
			return
		}

		newPassword := r.FormValue("new_password")
		if problems := localPasswordValidationProblems(
			newPassword,
			r.FormValue("new_password_confirm"),
			"new_password",
			"new_password_confirm",
			true,
		); len(problems) > 0 {
			httpresponse.Problem(w, http.StatusUnprocessableEntity, "Password validation failed.", problems...)
			return
		}

		token, err := local.ChangePassword(r.Context(), user.ID, user.Username, currentPassword, newPassword)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httpresponse.Problem(w, http.StatusUnauthorized, "Password validation failed.", httpresponse.NewFieldProblem("current_password", "The current password is incorrect."))
			return
		}
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		local.WriteSessionCookie(w, token)
		http.Redirect(w, r, "/settings#password", http.StatusSeeOther)
	}
}

// SavePreferences validates and stores all presentation preferences shown on the settings page.
func SavePreferences(preferenceUseCases preferenceService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid form.")
			return
		}

		selectedTheme, ok := themes.Find(views.themes, r.FormValue("theme"))
		if !ok {
			httpresponse.Problem(w, http.StatusBadRequest, "Unknown theme.")
			return
		}

		density := r.FormValue("navigation_density")
		if !service.ValidNavigationDensity(density) {
			httpresponse.Problem(w, http.StatusBadRequest, "Unknown navigation density.")
			return
		}

		sidebarWidth, err := strconv.Atoi(r.FormValue("sidebar_width"))
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Sidebar width is out of range.")
			return
		}
		if !service.ValidSidebarWidth(sidebarWidth) {
			httpresponse.Problem(w, http.StatusBadRequest, "Sidebar width is out of range.")
			return
		}

		current, err := preferenceUseCases.Preferences(r.Context(), user.ID)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		preferences := service.UserPreferences{
			Theme:                    selectedTheme.Title,
			ShowPageContents:         r.FormValue("show_page_contents") == "on",
			NavigationDensity:        density,
			SidebarWidth:             sidebarWidth,
			ShowNavigationGuides:     r.FormValue("show_navigation_guides") == "on",
			RememberNavigationState:  r.FormValue("remember_navigation_state") == "on",
			ShowPinnedPages:          r.FormValue("show_pinned_pages") == "on",
			ShowRecentlyViewed:       r.FormValue("show_recently_viewed") == "on",
			ShowNavigationPageCounts: r.FormValue("show_navigation_page_counts") == "on",
			ExpandedNavigation:       current.ExpandedNavigation,
		}
		if err := preferenceUseCases.SavePreferences(r.Context(), user.ID, preferences); err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		http.Redirect(w, r, "/settings#preferences", http.StatusSeeOther)
	}
}

// SavePageContentsPreference stores the floating page-contents toggle without changing other preferences.
func SavePageContentsPreference(preferenceUseCases preferenceService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid form.")
			return
		}

		show, err := strconv.ParseBool(r.FormValue("show"))
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid page contents preference.")
			return
		}

		if err := preferenceUseCases.SetShowPageContents(r.Context(), user.ID, show); err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SaveNavigationState stores the expanded folder paths for the current user.
func SaveNavigationState(preferenceUseCases preferenceService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w,
				http.StatusUnauthorized,
				"Unauthorized.",
				httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
			)
			return
		}

		request, err := decode[navigationStateRequest](w, r)
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid navigation state.",
				httpresponse.NewFieldProblem("expanded", err.Error()),
			)
			return
		}
		if len(request.Expanded) > 500 {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Navigation state is too large.",
				httpresponse.NewFieldProblem("expanded", "At most 500 expanded folders can be stored."),
			)
			return
		}
		if err := preferenceUseCases.SetExpandedNavigation(r.Context(), user.ID, request.Expanded); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SaveSidebarWidth stores the desktop sidebar width selected by the current user.
func SaveSidebarWidth(preferenceUseCases preferenceService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w,
				http.StatusUnauthorized,
				"Unauthorized.",
				httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
			)
			return
		}

		request, err := decode[sidebarWidthRequest](w, r)
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid sidebar width.",
				httpresponse.NewFieldProblem("width", err.Error()),
			)
			return
		}
		if !service.ValidSidebarWidth(request.Width) {
			httpresponse.Problem(w,
				http.StatusUnprocessableEntity,
				"Sidebar width is out of range.",
				httpresponse.NewFieldProblem("width", "Choose a width between 220 and 420 pixels."),
			)
			return
		}
		if err := preferenceUseCases.SetSidebarWidth(r.Context(), user.ID, request.Width); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
