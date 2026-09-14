package handler

import (
	"context"

	"github.com/gi8lino/lore/internal/domain"
	md "github.com/gi8lino/lore/internal/markdown"
)

// renderingOptions resolves administrator-controlled Markdown rendering behavior.
func renderingOptions(
	ctx context.Context,
	settingsUseCases settingsService,
) (options md.Options, application domain.ApplicationSettings, err error) {
	settings, err := settingsUseCases.ApplicationSettings(ctx)
	if err != nil {
		return md.Options{}, domain.ApplicationSettings{}, err
	}

	return renderingOptionsFromSettings(settings.Rendering), settings, nil
}

// renderingOptionsFromSettings maps persisted core rendering settings to Markdown renderer options.
// Plugin-owned features keep their default request flags and are controlled by
// plugin lifecycle and plugin-owned settings at the composition boundary.
func renderingOptionsFromSettings(rendering domain.RenderingSettings) md.Options {
	options := md.DefaultOptions()
	options.WikiLinks = rendering.WikiLinks
	options.Tabs = rendering.Tabs
	options.Details = rendering.Details
	options.Strikethrough = rendering.Strikethrough
	options.TaskLists = rendering.TaskLists
	options.Autolinks = rendering.Autolinks
	options.SyntaxHighlighting = rendering.SyntaxHighlighting
	options.Footnotes = rendering.Footnotes
	options.DefinitionLists = rendering.DefinitionLists
	options.Typographer = rendering.Typographer
	return options
}
