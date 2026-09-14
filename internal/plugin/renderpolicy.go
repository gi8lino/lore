package plugin

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// PolicyExtensions translates public rendering policy declarations into host grammars.
// These policies are available to every package, independently of plugin identity.
func (s Snapshot) PolicyExtensions() []goldmark.Extender {
	return policyExtensions(s.HasRenderPolicy)
}

// PolicyExtensions returns host grammars requested by the immutable render plan.
func (p *RenderPlan) PolicyExtensions() []goldmark.Extender {
	return policyExtensions(p.hasRenderPolicy)
}

func (p *RenderPlan) hasRenderPolicy(policy string) bool {
	if p == nil {
		return false
	}
	for _, contribution := range p.RenderPolicies {
		if contribution.Policy == policy {
			return true
		}
	}
	return false
}

func policyExtensions(hasPolicy func(string) bool) []goldmark.Extender {
	extensions := make([]goldmark.Extender, 0, 8)

	if hasPolicy("typographer") {
		var typographer goldmark.Extender = extension.Typographer
		if hasPolicy("coding-ligatures") {
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
