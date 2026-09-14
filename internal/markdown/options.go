package markdown

// Options controls optional Markdown rendering features.
type Options struct {
	pipeline *renderPipeline
	depth    int
	// variables is request-local provenance used only for reading-page inspection.
	variables []Variable
	// typographer reports whether an active plugin requested typographic substitutions.
	typographer bool
	// codingLigatures reports whether an active plugin requested operator-preserving substitutions.
	codingLigatures bool
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
}

// DefaultOptions returns the rendering behavior used before administrator customization.
func DefaultOptions() Options {
	return Options{
		Callouts:       true,
		Mermaid:        true,
		TableFiltering: true,
		Tables:         true,
		TableSorting:   true,
		TableStyles:    true,
		WikiLinks:      true,
		WikiLinkPrefix: "/pages/",
	}
}
