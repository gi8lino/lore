package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/themes"
)

// publicViewData builds shared data for unauthenticated setup and login pages.
func publicViewData(views *Views, title string) (ViewData, error) {
	preferences := domain.DefaultUserPreferences()
	activeTheme := themes.DefaultTheme
	preferences.Theme = activeTheme
	themeData, err := json.Marshal(views.themes)
	if err != nil {
		return ViewData{}, err
	}

	return ViewData{
		Title:        title,
		Preferences:  preferences,
		Version:      views.version,
		AssetVersion: views.assetVersion,
		Commit:       views.commit,
		Runtime:      views.runtime,
		ThemeData:    template.JS(themeData),
		Themes:       views.themes,
		ActiveTheme:  activeTheme,
	}, nil
}

// ViewDataLoader assembles the shared data required by authenticated HTML views.
type ViewDataLoader struct {
	preferenceUseCases   preferenceService
	navigationUseCases   navigationService
	catalogUseCases      sidebarCatalogService
	settingsUseCases     settingsService
	savedSearchUseCases  savedSearchReader
	notificationUseCases notificationReader
	accessUseCases       pageAccessReader
}

// NewViewDataLoader constructs the shared authenticated view-data loader.
func NewViewDataLoader(
	preferences preferenceService,
	navigation navigationService,
	catalog sidebarCatalogService,
	settings settingsService,
	savedSearches savedSearchReader,
	notifications notificationReader,
	access pageAccessReader,
) *ViewDataLoader {
	return &ViewDataLoader{
		preferenceUseCases:   preferences,
		navigationUseCases:   navigation,
		catalogUseCases:      catalog,
		settingsUseCases:     settings,
		savedSearchUseCases:  savedSearches,
		notificationUseCases: notifications,
		accessUseCases:       access,
	}
}

// Load builds the common template data used by every browser page.
func (l *ViewDataLoader) Load(r *http.Request, views *Views, title string) (ViewData, error) {
	user, _ := auth.User(r)

	preferences, err := l.preferenceUseCases.Preferences(r.Context(), user.ID)
	if err != nil {
		return ViewData{}, err
	}

	var pageNavigation []navigation.Node
	var sidebarPinned []domain.Page
	var sidebarRecent []domain.Page

	if !strings.HasPrefix(r.URL.Path, "/admin") {
		pages, err := l.navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			return ViewData{}, err
		}

		pages, err = l.accessUseCases.FilterPages(r.Context(), user, pages)
		if err != nil {
			return ViewData{}, err
		}

		navigationIcons, err := l.navigationUseCases.NavigationIcons(r.Context())
		if err != nil {
			return ViewData{}, err
		}

		expanded := preferences.ExpandedNavigation

		if !preferences.RememberNavigationState {
			expanded = nil
		}

		navigationPages := make([]navigation.Page, 0, len(pages))

		for _, page := range pages {
			navigationPages = append(navigationPages, navigation.Page{
				Slug:  page.Slug,
				Title: page.Title,
				Icon:  page.Icon,
			})
		}

		pageNavigation = navigation.Build(navigationPages, navigation.Options{
			ActiveSlug:     activeNavigationSlug(r.URL.Path),
			Expanded:       expanded,
			ShowPageCounts: preferences.ShowNavigationPageCounts,
			Icons:          navigationIcons,
		})

		if preferences.ShowPinnedPages {
			sidebarPinned, err = l.catalogUseCases.Favorites(r.Context(), user.ID)
			if err != nil {
				return ViewData{}, err
			}
			sidebarPinned, err = l.accessUseCases.FilterPages(r.Context(), user, sidebarPinned)
			if err != nil {
				return ViewData{}, err
			}
		}
		if preferences.ShowRecentlyViewed {
			sidebarRecent, err = l.catalogUseCases.RecentViewed(r.Context(), user.ID, 8)
			if err != nil {
				return ViewData{}, err
			}
			sidebarRecent, err = l.accessUseCases.FilterPages(r.Context(), user, sidebarRecent)
			if err != nil {
				return ViewData{}, err
			}

			sidebarRecent = pagesWithout(sidebarRecent, sidebarPinned, 5)
		}
	}

	applicationSettings, err := l.settingsUseCases.ApplicationSettings(r.Context())
	if err != nil {
		return ViewData{}, err
	}

	typographySize := preferences.TypographySize
	if typographySize == "" {
		typographySize = applicationSettings.Rendering.DefaultTypographySize
	}
	if !domain.ValidTypographySize(typographySize) {
		typographySize = domain.DefaultTypographySize
	}

	activeTheme := themes.DefaultTheme

	if selected, ok := themes.Find(views.themes, preferences.Theme); ok {
		activeTheme = selected.Title
	}

	preferences.Theme = activeTheme

	themeData, err := json.Marshal(views.themes)
	if err != nil {
		return ViewData{}, err
	}

	savedSearches, err := l.savedSearchUseCases.SavedSearches(r.Context(), user.ID)
	if err != nil {
		return ViewData{}, err
	}

	notifications, unreadNotifications, err := l.notificationUseCases.Notifications(r.Context(), user.ID, 8)
	if err != nil {
		return ViewData{}, err
	}

	return ViewData{
		Title:               title,
		User:                user,
		Preferences:         preferences,
		TypographySize:      typographySize,
		Navigation:          pageNavigation,
		NewPageParent:       activeNavigationSlug(r.URL.Path),
		SidebarPinned:       sidebarPinned,
		SidebarRecent:       sidebarRecent,
		SavedSearches:       savedSearches,
		Notifications:       notifications,
		UnreadNotifications: unreadNotifications,
		PageStatuses:        domain.PageStatuses(),
		Version:             views.version,
		AssetVersion:        views.assetVersion,
		Commit:              views.commit,
		Runtime:             views.runtime,
		ThemeData:           template.JS(themeData),
		Themes:              views.themes,
		ActiveTheme:         activeTheme,
		ApplicationSettings: applicationSettings,
		CanEdit:             user.Role == "admin" || user.Role == "editor",
		RenderMermaid:       applicationSettings.Rendering.Mermaid,
		PageContentLanguage: applicationSettings.Rendering.ContentLanguage,
	}, nil
}

// viewData loads shared authenticated view data through the handler's narrow dependency.
func viewData(r *http.Request, loader viewDataService, views *Views, title string) (ViewData, error) {
	return loader.Load(r, views, title)
}

// pagesWithout returns up to limit pages excluding any page present in excluded.
func pagesWithout(pages, excluded []domain.Page, limit int) []domain.Page {
	if limit <= 0 {
		return nil
	}

	excludedIDs := make(map[int64]bool, len(excluded))

	for _, page := range excluded {
		excludedIDs[page.ID] = true
	}

	result := make([]domain.Page, 0, min(limit, len(pages)))

	for _, page := range pages {
		if excludedIDs[page.ID] {
			continue
		}

		result = append(result, page)
		if len(result) == limit {
			break
		}
	}

	return result
}

// activeNavigationSlug extracts the current page slug from browser page and editor routes.
func activeNavigationSlug(requestPath string) string {
	for _, prefix := range []string{"/pages/", "/edit/"} {
		slug, ok := strings.CutPrefix(requestPath, prefix)
		if !ok {
			continue
		}

		slug = strings.Trim(slug, "/")
		if slug == "new" {
			return ""
		}
		return slug
	}
	return ""
}
