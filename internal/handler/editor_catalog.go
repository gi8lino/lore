package handler

import (
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/plugin"
)

// EditorCatalog returns page, plugin-completion, and legacy snippet metadata used by editor intelligence.
func EditorCatalog(
	navigationUseCases navigationService,
	knowledgeUseCases knowledgeSnippetReader,
	catalogUseCases pageAliasService,
	plugins *plugin.Manager,
	logger *slog.Logger,
) http.HandlerFunc {
	type pageItem struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			httpresponse.InternalServerError(logger, w, err)
			return
		}

		items := make([]pageItem, 0, len(pages))
		for _, page := range pages {
			items = append(items, pageItem{Slug: page.Slug, Title: page.Title})
		}

		snippets, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			httpresponse.InternalServerError(logger, w, err)
			return
		}

		var completions []plugin.EditorCompletionItem
		var inserts []plugin.EditorInsertContribution
		if plugins != nil {
			completions, err = plugins.EditorCompletions(r.Context())
			if err != nil {
				httpresponse.InternalServerError(logger, w, err)
				return
			}
			inserts = plugins.EditorInserts()
		}

		aliases, err := catalogUseCases.PageAliases(r.Context())
		if err != nil {
			httpresponse.InternalServerError(logger, w, err)
			return
		}
		if aliases == nil {
			aliases = map[string]string{}
		}

		httpresponse.Respond(w, http.StatusOK, map[string]any{
			"pages":       items,
			"snippets":    jsonSlice(snippets),
			"completions": jsonSlice(completions),
			"inserts":     jsonSlice(inserts),
			"aliases":     aliases,
		})
	}
}
