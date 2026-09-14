package markdown

import "github.com/gi8lino/lore/internal/plugin"

// Options controls optional Markdown rendering features.
type Options struct {
	pipeline *renderPipeline
	depth    int
	// annotations contains request-local opaque plugin substitutions for an annotated render pass.
	annotations []plugin.Replacement
	// variables is temporary core variable provenance kept until Variables migrates to a plugin.
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
