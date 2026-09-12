package markdown

import (
	"github.com/gi8lino/lore/internal/pagereport"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugins/callouts"
	"github.com/gi8lino/lore/internal/subpages"
)

// DefaultRegistry is the composition root shared by server, preview, exports,
// and static builds. Each call creates an independent registry. Bundling is
// distribution only; these entries use the ordinary registration path.
func DefaultRegistry() *plugin.Registry {
	registry := &plugin.Registry{}
	register := func(d plugin.Descriptor, c plugin.Contributions) {
		if err := registry.Register(d, c); err != nil {
			panic(err)
		}
	}
	register(callouts.Descriptor(), plugin.Contributions{Preprocessors: []plugin.Preprocessor{callouts.Module{}}})
	register(plugin.Descriptor{ID: "io.lore.subpages", Name: "Subpages", DefaultEnabled: true}, plugin.Contributions{
		Macros: []plugin.Macro{plugin.BoundMacro[subpages.Options]{MacroName: "subpages", ParseOptions: subpages.Parse, EmptyWhenUnbound: true}},
	})
	register(plugin.Descriptor{ID: "io.lore.page-report", Name: "Page Report", DefaultEnabled: true}, plugin.Contributions{
		Macros: []plugin.Macro{plugin.BoundMacro[pagereport.Options]{MacroName: "pages", ParseOptions: pagereport.Parse}},
	})
	return registry
}

// moduleFeatures translates legacy settings at the composition boundary. It
// does not activate modules absent from the registry.
func moduleFeatures(options Options) map[string]bool {
	return map[string]bool{callouts.ID: options.Callouts}
}
