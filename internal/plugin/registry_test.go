package plugin

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryRegistrationIsAtomicAndReversible(t *testing.T) {
	r := &Registry{}
	descriptor := Descriptor{ID: "base", Name: "Base"}
	macro := BoundMacro[string]{MacroName: "example", ParseOptions: func(s string) (string, bool) { return s, true }}
	modules := Contributions{Macros: []Macro{macro}, BrowserModules: []BrowserModule{{ID: "browser", JavaScript: "assets/plugin.js"}}}
	require.NoError(t, r.Register(descriptor, modules))
	modules.Macros[0] = nil
	modules.BrowserModules[0].JavaScript = "mutated"
	snapshot := r.Snapshot()
	require.Equal(t, "example", snapshot.Entries[0].Contributions.Macros[0].Name())
	require.Equal(t, "assets/plugin.js", snapshot.Entries[0].Contributions.BrowserModules[0].JavaScript)
	require.Error(t, r.Register(descriptor, Contributions{}))
	require.Error(t, r.Register(Descriptor{ID: "collision", Name: "Collision"}, Contributions{Macros: []Macro{macro}}))
	require.Error(t, r.Register(Descriptor{ID: "missing", Name: "Missing", Requires: []string{"absent"}}, Contributions{}))
	require.Len(t, r.Snapshot().Entries, 1)
	dependency := Descriptor{ID: "dependent", Name: "Dependent", Requires: []string{"base"}}
	require.NoError(t, r.Register(dependency, Contributions{}))
	dependency.Requires[0] = "mutated"
	require.Error(t, r.Unregister("base"))
	require.NoError(t, r.Unregister("dependent"))
	require.NoError(t, r.Unregister("base"))
	require.Empty(t, r.Snapshot().Entries)
	require.Len(t, snapshot.Entries, 1)
	require.NoError(t, r.Register(descriptor, Contributions{Macros: []Macro{macro}}))
	snapshot.Entries[0].Descriptor.Name = "mutated"
	require.Equal(t, "Base", r.Snapshot().Entries[0].Descriptor.Name)
}

func TestRegistryRejectsInvalidContributions(t *testing.T) {
	for _, modules := range []Contributions{
		{Macros: []Macro{nil}},
		{Preprocessors: []Preprocessor{nil}},
		{MarkdownExtensions: []MarkdownExtension{nil}},
		{Postprocessors: []Postprocessor{nil}},
		{BrowserModules: []BrowserModule{{ID: "same"}, {ID: "same"}}},
		{EditorExtensions: []EditorExtension{{ID: "../bad"}}},
	} {
		r := &Registry{}
		require.Error(t, r.Register(Descriptor{ID: "test", Name: "Test"}, modules))
		require.Empty(t, r.Snapshot().Entries)
	}
}

func TestRegistryConcurrentSnapshotsAndRemoval(t *testing.T) {
	r := &Registry{}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				_ = r.Snapshot()
			}
		})
	}
	for range 100 {
		require.NoError(t, r.Register(Descriptor{ID: "test", Name: "Test"}, Contributions{}))
		require.NoError(t, r.Unregister("test"))
	}
	wg.Wait()
}
