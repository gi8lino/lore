// Package plugin defines Lore's in-process module contracts. Only trusted,
// compiled modules may implement native callbacks; installed code executes through
// the sandboxed runtime adapter. This package grants no persistence or I/O access.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yuin/goldmark"
)

// Descriptor identifies a plugin independently of how it is distributed.
type Descriptor struct {
	ID             string
	Name           string
	Description    string
	DefaultEnabled bool
	Requires       []string
}

// Context contains only render-local capabilities. RenderMarkdown returns
// intermediate HTML: the host must sanitize the complete document afterwards.
// Callbacks must not retain this context or mutate its maps.
type Context struct {
	Context        context.Context
	Features       map[string]bool
	Macros         map[string]MacroRenderer
	RenderMarkdown func(string) (string, error)
}

// Invocation is a serializable macro argument value suitable for a future ABI.
type Invocation = json.RawMessage

// MacroRenderer binds request-local data to a registered macro.
type MacroRenderer func(Invocation) (string, error)

// ConditionalMacro optionally controls whether a macro can expand in a
// particular request. An unavailable invocation stays ordinary Markdown.
type ConditionalMacro interface{ Available(Context) bool }

// Macro recognizes a standalone invocation and produces untrusted HTML.
type Macro interface {
	Name() string
	Parse(string) (Invocation, bool)
	Render(Context, Invocation) (string, error)
}

// Preprocessor transforms Markdown before parsing, including nested blocks.
type Preprocessor interface {
	Preprocess(Context, string) (string, error)
}

// MarkdownExtension creates fresh Goldmark components for each conversion.
// This is a host-side adapter, not an API for loading native community code.
// Extenders can contribute parsers, AST transformers, and node renderers.
type MarkdownExtension interface {
	Extension(Context) goldmark.Extender
}

// Postprocessor transforms the complete HTML, including macro output, before
// the trusted central sanitizer. It runs once per document, not per nested block.
type Postprocessor interface {
	Postprocess(Context, string) (string, error)
}

// BrowserModule describes assets; serving and loading them is a later phase.
type BrowserModule struct{ ID, JavaScript, CSS string }

// EditorExtension reserves editor contribution metadata without loading assets.
type EditorExtension struct{ ID, BrowserModuleID string }

// SettingsModule describes a settings contribution without exposing core storage.
type SettingsModule struct{ ID, Name string }

// Contributions is registered and removed atomically under its owner's ID.
// Order within a stage is registration order, then slice order.
type Contributions struct {
	Preprocessors      []Preprocessor
	MarkdownExtensions []MarkdownExtension
	Postprocessors     []Postprocessor
	Macros             []Macro
	BrowserModules     []BrowserModule
	EditorExtensions   []EditorExtension
	SettingsModules    []SettingsModule
}

// BindMacro adapts a typed, request-scoped renderer to serialized arguments.
func BindMacro[T any](render func(T) (string, error)) MacroRenderer {
	if render == nil {
		return nil
	}
	return func(value Invocation) (string, error) {
		var options T
		if err := json.Unmarshal(value, &options); err != nil {
			return "", err
		}
		return render(options)
	}
}

// BoundMacro adapts an existing parser to a request-local renderer. When no
// binding exists it leaves the invocation literal, unless EmptyWhenUnbound is
// set to preserve a module's established empty-context behavior.
type BoundMacro[T any] struct {
	MacroName        string
	ParseOptions     func(string) (T, bool)
	EmptyWhenUnbound bool
}

func (m BoundMacro[T]) Name() string { return m.MacroName }

func (m BoundMacro[T]) Available(ctx Context) bool {
	return m.EmptyWhenUnbound || ctx.Macros[m.Name()] != nil
}

func (m BoundMacro[T]) Parse(line string) (Invocation, bool) {
	options, ok := m.ParseOptions(line)
	if !ok {
		return nil, false
	}
	encoded, err := json.Marshal(options)
	return encoded, err == nil
}

func (m BoundMacro[T]) Render(ctx Context, value Invocation) (string, error) {
	if render := ctx.Macros[m.Name()]; render != nil {
		return render(value)
	}
	if m.EmptyWhenUnbound {
		return "", nil
	}
	return "", fmt.Errorf("macro %s has no request binding", m.Name())
}

// Guard converts synchronous native module panics into render errors. Native
// modules remain trusted: goroutines, CPU, and memory need a WASM sandbox later.
func Guard[T any](owner string, run func() (T, error)) (result T, err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("plugin %s panicked", owner)
		}
	}()
	result, err = run()
	if err != nil {
		err = fmt.Errorf("plugin %s: %w", owner, err)
	}
	return result, err
}
