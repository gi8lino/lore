package handler

import (
	"context"

	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/service"
)

// renderingOptions resolves administrator-controlled Markdown rendering behavior.
func renderingOptions(
	ctx context.Context,
	settingsUseCases settingsService,
) (options md.Options, rendering service.RenderingSettings, err error) {
	settings, err := settingsUseCases.ApplicationSettings(ctx)
	if err != nil {
		return md.Options{}, service.RenderingSettings{}, err
	}

	return renderingOptionsFromSettings(settings.Rendering), settings.Rendering, nil
}

// renderingOptionsFromSettings maps persisted rendering settings to Markdown renderer options.
func renderingOptionsFromSettings(rendering service.RenderingSettings) md.Options {
	return md.Options{
		WikiLinks:          rendering.WikiLinks,
		Callouts:           rendering.Callouts,
		Tabs:               rendering.Tabs,
		Details:            rendering.Details,
		Tables:             rendering.Tables,
		TableStyles:        rendering.TableStyles,
		TableSorting:       rendering.TableSorting,
		TableFiltering:     rendering.TableFiltering,
		Strikethrough:      rendering.Strikethrough,
		TaskLists:          rendering.TaskLists,
		Autolinks:          rendering.Autolinks,
		SyntaxHighlighting: rendering.SyntaxHighlighting,
		Footnotes:          rendering.Footnotes,
		DefinitionLists:    rendering.DefinitionLists,
		Typographer:        rendering.Typographer,
	}
}
