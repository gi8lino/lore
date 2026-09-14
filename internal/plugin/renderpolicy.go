package plugin

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// PolicyExtensions translates public rendering policy declarations into host grammars.
// These policies are available to every package, independently of plugin identity.
func (s Snapshot) PolicyExtensions() []goldmark.Extender {
	extensions := make([]goldmark.Extender, 0, 8)

	if s.HasRenderPolicy("typographer") {
		var typographer goldmark.Extender = extension.Typographer
		if s.HasRenderPolicy("coding-ligatures") {
			typographer = extension.NewTypographer(
				extension.WithTypographicSubstitutions(
					extension.TypographicSubstitutions{
						extension.EnDash:          nil,
						extension.EmDash:          nil,
						extension.LeftAngleQuote:  nil,
						extension.RightAngleQuote: nil,
					},
				),
			)
		}
		extensions = append(extensions, typographer)
	}
	return extensions
}
