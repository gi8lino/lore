package handler

import "net/http"

// Search executes free-text search plus supported field filters.
func Search(
	viewDataUseCases viewDataService,
	catalogUseCases pageSearchService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		pages, err := catalogUseCases.Search(r.Context(), query, 50)
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		data, err := viewData(r, viewDataUseCases, views, "Search")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		data.Query, data.Pages = query, pages
		render(views, w, "search", data)
	}
}
