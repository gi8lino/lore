package markdown

// Options controls optional Markdown rendering features.
type Options struct {
	pipeline *renderPipeline
	depth    int
	// variables is request-local provenance used only for reading-page inspection.
	variables []Variable
	// WikiLinks enables [[Wiki Link]] resolution.
	WikiLinks bool
	// WikiLinkPrefix is prepended to resolved wiki-link targets. Empty uses /pages/.
	WikiLinkPrefix string
	// Callouts enables Lore callout blocks.
	Callouts bool
	// Mermaid enables the diagram plugin when it is registered.
	Mermaid bool
	// Tables enables GitHub-flavored Markdown tables.
	Tables bool
	// TableStyles enables trusted theme-aware table colors.
	TableStyles bool
	// TableSorting enables client-side sorting for opted-in tables.
	TableSorting bool
	// TableFiltering enables client-side filtering for opted-in tables.
	TableFiltering bool
	// SyntaxHighlighting enables server-side fenced-code highlighting.
	SyntaxHighlighting bool
	// DefinitionLists enables Markdown definition lists.
	DefinitionLists bool
	// Typographer enables typographic punctuation substitutions.
	Typographer bool
	// CodingLigatures preserves ASCII operators when typographic punctuation is enabled.
	CodingLigatures bool
}

// DefaultOptions returns the rendering behavior used before administrator customization.
func DefaultOptions() Options {
	return Options{
		Callouts:           true,
		Mermaid:            true,
		DefinitionLists:    true,
		SyntaxHighlighting: true,
		TableFiltering:     true,
		Tables:             true,
		TableSorting:       true,
		TableStyles:        true,
		Typographer:        true,
		WikiLinks:          true,
		WikiLinkPrefix:     "/pages/",
	}
}
