package handler

import (
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/httpresponse"
)

// EditorCatalog returns page and reusable-snippet metadata used by editor intelligence.
func EditorCatalog(
	navigationUseCases navigationService,
	knowledgeUseCases knowledgeSnippetReader,
	catalogUseCases pageAliasService,
	logger *slog.Logger,
) http.HandlerFunc {
	type pageItem struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			writeInternalServerError(logger, w, err)
			return
		}

		items := make([]pageItem, 0, len(pages))
		for _, page := range pages {
			items = append(items, pageItem{Slug: page.Slug, Title: page.Title})
		}

		snippets, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			writeInternalServerError(logger, w, err)
			return
		}

		aliases, err := catalogUseCases.PageAliases(r.Context())
		if err != nil {
			writeInternalServerError(logger, w, err)
			return
		}
		if aliases == nil {
			aliases = map[string]string{}
		}

		httpresponse.Respond(w, http.StatusOK, map[string]any{
			"pages":    items,
			"snippets": jsonSlice(snippets),
			"aliases":  aliases,
		})
	}
}
