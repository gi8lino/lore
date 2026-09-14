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
	// ID identifies the associated object.
	ID string
	// Name is the human-readable name.
	Name string
	// Description summarizes the associated object.
	Description string
	// DefaultEnabled indicates whether the plugin starts enabled by default.
	DefaultEnabled bool
	// Requires lists plugin IDs that must be active before this plugin.
	Requires []string
}

// Context contains only render-local capabilities. RenderMarkdown returns
// intermediate HTML: the host must sanitize the complete document afterwards.
// Callbacks must not retain this context or mutate its maps.
type Context struct {
	// Context carries cancellation and request-scoped values.
	Context context.Context
	// Capabilities exposes request-scoped host capabilities.
	Capabilities map[string]Capability
	// Features contains request-scoped feature flags.
	Features map[string]bool
	// Macros contains request-scoped macro renderers.
	Macros map[string]MacroRenderer
	// ExportParameters contains request-local values keyed by plugin and field.
	ExportParameters map[string]map[string]map[string]string
	// RenderMarkdown renders nested Markdown through the host pipeline.
	RenderMarkdown func(string) (string, error)
}

// Replacement binds an opaque source token to Markdown inserted after all content preprocessors run.
type Replacement struct {
	// Token is the opaque source marker emitted by a content preprocessor.
	Token string
	// Value is the Markdown restored immediately before Goldmark parsing.
	Value string
	// Annotation identifies an optional inspector item that owns this replacement.
	Annotation string
}

// InspectorItem is one value exposed by a plugin-owned reading-page inspector.
type InspectorItem struct {
	// Key identifies the item within the inspector.
	Key string
	// Label is the human-readable item name.
	Label string
	// Value is the saved value shown by the inspector.
	Value string
	// Description is optional explanatory text.
	Description string
	// Occurrences is the number of replacements originating from this item.
	Occurrences int
	// Annotation identifies rendered spans associated with this item.
	Annotation string
}

// Inspector describes one plugin-owned reading-page inspector.
type Inspector struct {
	// ID identifies the inspector within its plugin.
	ID string
	// PluginID identifies the owning plugin.
	PluginID string
	// Name is the human-readable inspector title.
	Name string
	// Items contains values used by the current page.
	Items []InspectorItem
}

// ExportField describes one request-local plugin parameter exposed by page export UI.
type ExportField struct {
	// PluginID identifies the owning plugin.
	PluginID string
	// ModuleID identifies the owning content module.
	ModuleID string
	// Key identifies the exported resource item.
	Key string
	// Label is the human-readable field label.
	Label string
	// Value is the saved value used as the initial export value.
	Value string
	// Description is optional explanatory text.
	Description string
}

// PreparedContent contains Markdown after plugin-owned content preprocessing plus render metadata.
type PreparedContent struct {
	// Markdown is the transformed source passed to the Markdown renderer.
	Markdown string
	// Replacements restore opaque plugin substitutions immediately before parsing.
	Replacements []Replacement
	// Inspectors contains reading-page metadata contributed by enabled plugins.
	Inspectors []Inspector
	// ExportFields contains request-local export controls contributed by enabled plugins.
	ExportFields []ExportField
}

// ContentPreprocessor transforms page Markdown before the normal render pipeline.
// Its output is never recursively reprocessed by another pass; preprocessors run once
// in ascending priority order.
type ContentPreprocessor interface {
	// Priority orders content preprocessing. Lower values run first.
	Priority() int
	// PreprocessContent transforms source and may contribute render metadata.
	PreprocessContent(Context, string) (PreparedContent, error)
}

// AdminResource describes one declarative plugin-owned record collection.
type AdminResource struct {
	// ID identifies the resource within its plugin.
	ID string
	// Name is the human-readable resource title.
	Name string
	// Description explains the resource to administrators.
	Description string
}

// EditorCompletion describes one resource-backed editor completion provider.
type EditorCompletion struct {
	// ID identifies the contribution within its plugin.
	ID string
	// Resource identifies the backing admin-resource module.
	Resource string
	// Trigger opens completion in the Markdown editor.
	Trigger string
	// Replacement formats one selected record into Markdown.
	Replacement string
	// LabelField and DetailField select resource fields shown in completion results.
	LabelField, DetailField string
}

// EditorInsert describes one static insert action owned by a plugin.
type EditorInsert struct {
	// ID identifies the contribution within its plugin.
	ID string
	// Name and Description are shown in editor insertion UI.
	Name, Description string
	// Markdown is inserted when the action is selected.
	Markdown string
	// Inline reports whether insertion should avoid surrounding line breaks.
	Inline bool
}

// Invocation is a serializable macro argument value carried by the WASM ABI.
type Invocation = json.RawMessage

// MacroRenderer binds request-local data to a registered macro.
type MacroRenderer func(Invocation) (string, error)

// ConditionalMacro optionally controls whether a macro can expand in a
// particular request. An unavailable invocation stays ordinary Markdown.
type ConditionalMacro interface {
	// Available reports whether the macro can expand in the current render context.
	Available(Context) bool
}

// ContextualMacro parses invocations using request-local render context.
type ContextualMacro interface {
	// ParseContext parses one invocation with access to request-local capabilities.
	ParseContext(Context, string) (Invocation, bool, error)
}

// Macro recognizes a standalone invocation and produces untrusted HTML.
type Macro interface {
	// Name returns the registered contribution name.
	Name() string
	// Parse recognizes one macro invocation and returns serialized arguments.
	Parse(string) (Invocation, bool)
	// Render expands one parsed macro invocation into untrusted HTML.
	Render(Context, Invocation) (string, error)
}

// Preprocessor transforms Markdown before parsing, including nested blocks.
type Preprocessor interface {
	// Preprocess transforms Markdown before the core parser runs.
	Preprocess(Context, string) (string, error)
}

// MarkdownExtension creates fresh Goldmark components for each conversion.
// This is a host-side adapter, not an API for loading native community code.
// Extenders can contribute parsers, AST transformers, and node renderers.
type MarkdownExtension interface {
	// Extension returns a fresh Goldmark extension for the current render.
	Extension(Context) goldmark.Extender
}

// CodeHighlightResult contains one highlighter response. HTML remains untrusted
// until Lore sanitizes the complete rendered document.
type CodeHighlightResult struct {
	// HTML contains highlighted block markup when Matched is true.
	HTML string
	// Matched reports whether the provider recognized the fenced-code language.
	Matched bool
}

// CodeHighlighter highlights one fenced code block. Only one provider may be
// active at a time so fenced-code ownership is deterministic.
type CodeHighlighter interface {
	// Highlight renders source for language or reports that the language is unsupported.
	Highlight(Context, string, string) (CodeHighlightResult, error)
}

// CodeHighlighterModule describes one exclusive code-highlighting contribution.
type CodeHighlighterModule struct {
	// ID identifies the module within its plugin.
	ID string
	// CSS is an optional package stylesheet filtered and scoped by Lore.
	CSS string
	// Highlighter performs the sandboxed highlighting operation.
	Highlighter CodeHighlighter
}

// Postprocessor transforms the complete HTML, including macro output, before
// the trusted central sanitizer. It runs once per document, not per nested block.
type Postprocessor interface {
	// Postprocess transforms rendered HTML before central sanitization.
	Postprocess(Context, string) (string, error)
}

// BrowserModule declares browser assets contributed by one plugin module.
type BrowserModule struct {
	// ID, JavaScript, and CSS identify the module and its optional asset paths.
	ID, JavaScript, CSS string
}

// EditorExtension reserves editor contribution metadata without loading assets.
type EditorExtension struct {
	// ID and BrowserModuleID identify the editor extension and its browser module.
	ID, BrowserModuleID string
}

// SettingsModule describes a settings contribution without exposing core storage.
type SettingsModule struct {
	// ID and Name identify the settings contribution and its display name.
	ID, Name string
	// Requires lists prerequisite feature keys within the owning package.
	Requires []string
}

// ContentStyle declares a stylesheet that core may expose to rendered page
// content after applying its parent-document CSS safety filter.
type ContentStyle struct {
	// ID identifies the contribution within its plugin.
	ID string
	// CSS is the validated package-relative stylesheet asset path.
	CSS string
}

// RenderPolicy declares one host rendering behavior requested by an active plugin.
type RenderPolicy struct {
	// ID identifies the contribution within its plugin.
	ID string
	// Policy is the versioned host policy name.
	Policy string
}

// Contributions is registered and removed atomically under its owner's ID.
// Order within a stage is registration order, then slice order.
type Contributions struct {
	// ContentPreprocessors transform application Markdown before the normal render pipeline.
	ContentPreprocessors []ContentPreprocessor
	// Preprocessors run before core Markdown parsing in contribution order.
	Preprocessors []Preprocessor
	// MarkdownExtensions contribute fresh Goldmark extensions per render.
	MarkdownExtensions []MarkdownExtension
	// CodeHighlighters contains an exclusive fenced-code highlighting provider.
	CodeHighlighters []CodeHighlighterModule
	// Postprocessors run on rendered HTML before central sanitization.
	Postprocessors []Postprocessor
	// Macros contains request-scoped macro renderers.
	Macros []Macro
	// BrowserModules declares browser assets exposed for enabled plugins.
	BrowserModules []BrowserModule
	// EditorExtensions declares editor integrations owned by the plugin.
	EditorExtensions []EditorExtension
	// AdminResources declares plugin-owned record collections rendered by Lore.
	AdminResources []AdminResource
	// EditorCompletions declares resource-backed editor completion providers.
	EditorCompletions []EditorCompletion
	// EditorInserts declares static editor insertion actions.
	EditorInserts []EditorInsert
	// SettingsModules declares settings integrations owned by the plugin.
	SettingsModules []SettingsModule
	// ContentStyles declares safe parent-document styles owned by the plugin.
	ContentStyles []ContentStyle
	// RenderPolicies declares host rendering behavior owned by the plugin.
	RenderPolicies []RenderPolicy
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
	// MacroName is the registered macro identifier.
	MacroName string
	// ParseOptions recognizes source syntax and returns typed macro options.
	ParseOptions func(string) (T, bool)
	// EmptyWhenUnbound renders an empty result when no request-local binding exists.
	EmptyWhenUnbound bool
}

// Name returns the registered contribution name.
func (m BoundMacro[T]) Name() string { return m.MacroName }

// Available reports whether the contribution is available in the current context.
func (m BoundMacro[T]) Available(ctx Context) bool {
	return m.EmptyWhenUnbound || ctx.Macros[m.Name()] != nil
}

// Parse recognizes one macro invocation and returns serialized arguments.
func (m BoundMacro[T]) Parse(line string) (Invocation, bool) {
	options, ok := m.ParseOptions(line)
	if !ok {
		return nil, false
	}
	encoded, err := json.Marshal(options)
	return encoded, err == nil
}

// Render expands one parsed macro invocation into untrusted HTML.
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
// modules remain trusted; installed contributions execute in the WASM sandbox.
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
