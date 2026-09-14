package pluginbrowser

import (
	"regexp"
	"strings"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"github.com/gi8lino/lore/internal/plugin"
)

var presentationSelector = regexp.MustCompile(`^[a-zA-Z0-9 .>:_-]{1,256}$`)
var presentationValue = regexp.MustCompile(`^[a-zA-Z0-9 #(),.%_-]{1,512}$`)
var presentationFunction = regexp.MustCompile(`([a-zA-Z_-]+)\s*\(`)

// PresentationStyles publishes only scoped presentation declarations from active
// package stylesheets. Arbitrary CSS stays in the sandbox: positioning, URLs,
// generated content, imports, escapes and selector functions are not admitted.
func PresentationStyles(manager *plugin.Manager) string {
	if manager == nil {
		return ""
	}
	var output strings.Builder
	for _, module := range manager.BrowserModules() {
		if module.CSS == "" {
			continue
		}
		data, err := manager.BrowserAsset(module.PluginID, module.Digest, module.CSS)
		if err != nil || len(data) > 256<<10 {
			continue
		}
		output.WriteString(scopedColors(module.PluginID, string(data)))
	}
	for _, module := range manager.CodeHighlighters() {
		if module.CSS == "" {
			continue
		}
		data, err := manager.CodeHighlighterAsset(module.PluginID, module.Digest, module.CSS)
		if err != nil || len(data) > 256<<10 {
			continue
		}
		output.WriteString(scopedCodeStyles(module.PluginID, string(data)))
	}
	for _, module := range manager.ContentStyles() {
		if module.CSS == "" {
			continue
		}
		data, err := manager.ContentStyleAsset(module.PluginID, module.Digest, module.CSS)
		if err != nil || len(data) > 256<<10 {
			continue
		}
		output.WriteString(scopedContentStyles(string(data)))
	}
	return output.String()
}

// scopedColors returns only safe, plugin-scoped color declarations from source.
func scopedColors(id, source string) string {
	sheet, err := parser.Parse(source)
	if err != nil {
		return ""
	}
	var output strings.Builder
	for _, rule := range sheet.Rules {
		if rule.Kind != css.QualifiedRule {
			continue
		}
		var selectors []string
		for _, selector := range rule.Selectors {
			selector = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(selector), ".prose "))
			if !presentationSelector.MatchString(selector) {
				continue
			}
			selectors = append(selectors, `[data-lore-plugin="`+id+`"] `+selector)
		}
		if len(selectors) == 0 {
			continue
		}
		var declarations []string
		for _, declaration := range rule.Declarations {
			property := declaration.Property
			if property == "background" {
				property = "background-color"
			}
			if !presentationProperty(property) {
				continue
			}
			if !safeColor(declaration.Value) {
				continue
			}
			declarations = append(declarations, property+":"+declaration.Value+";")
		}
		if len(declarations) > 0 {
			output.WriteString(strings.Join(selectors, ",") + "{" + strings.Join(declarations, "") + "}\n")
		}
	}
	return output.String()
}

// scopedCodeStyles returns safe highlighter presentation rules scoped to one plugin wrapper.
func scopedCodeStyles(id, source string) string {
	sheet, err := parser.Parse(source)
	if err != nil {
		return ""
	}

	var output strings.Builder
	for _, rule := range sheet.Rules {
		if rule.Kind != css.QualifiedRule {
			continue
		}

		selectors := make([]string, 0, len(rule.Selectors))
		for _, selector := range rule.Selectors {
			selector = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(selector), ".prose "))
			if !presentationSelector.MatchString(selector) {
				continue
			}
			selectors = append(selectors, `[data-lore-plugin="`+id+`"] `+selector)
		}
		if len(selectors) == 0 {
			continue
		}

		declarations := make([]string, 0, len(rule.Declarations))
		for _, declaration := range rule.Declarations {
			property, value, ok := safeCodeDeclaration(declaration.Property, declaration.Value)
			if ok {
				declarations = append(declarations, property+":"+value+";")
			}
		}
		if len(declarations) != 0 {
			output.WriteString(strings.Join(selectors, ",") + "{" + strings.Join(declarations, "") + "}\n")
		}
	}

	return output.String()
}

// safeCodeDeclaration validates the small presentation subset exposed to highlighters.
func safeCodeDeclaration(property, value string) (string, string, bool) {
	switch property {
	case "background":
		property = "background-color"
		fallthrough
	case "background-color", "color", "border-color":
		return property, value, safeColor(value)
	case "font-style":
		return property, value, value == "normal" || value == "italic" || value == "oblique"
	case "font-weight":
		switch value {
		case "normal", "bold", "100", "200", "300", "400", "500", "600", "700", "800", "900":
			return property, value, true
		}
	}

	return "", "", false
}

// presentationProperty reports whether property is allowed in parent-document presentation CSS.
func presentationProperty(property string) bool {
	switch property {
	case "background-color", "color", "border-color":
		return true
	default:
		return false
	}
}

// safeColor reports whether value uses only the supported color syntax.
func safeColor(value string) bool {
	if !presentationValue.MatchString(value) {
		return false
	}
	for _, match := range presentationFunction.FindAllStringSubmatch(value, -1) {
		switch match[1] {
		case "var", "color-mix", "rgb", "rgba", "hsl", "hsla":
		default:
			return false
		}
	}
	return true
}

// scopedContentStyles returns safe typography rules rooted in rendered page content.
func scopedContentStyles(source string) string {
	sheet, err := parser.Parse(source)
	if err != nil {
		return ""
	}

	var output strings.Builder
	for _, rule := range sheet.Rules {
		if rule.Kind != css.QualifiedRule {
			continue
		}

		selectors := make([]string, 0, len(rule.Selectors))
		for _, selector := range rule.Selectors {
			selector = strings.TrimSpace(selector)
			if safeContentSelector(selector) {
				selectors = append(selectors, selector)
			}
		}
		if len(selectors) == 0 {
			continue
		}

		declarations := make([]string, 0, len(rule.Declarations))
		for _, declaration := range rule.Declarations {
			if !contentStyleProperty(declaration.Property) || !safeTypographyValue(declaration.Value) {
				continue
			}
			declarations = append(declarations, declaration.Property+":"+declaration.Value+";")
		}
		if len(declarations) != 0 {
			output.WriteString(strings.Join(selectors, ",") + "{" + strings.Join(declarations, "") + "}\n")
		}
	}

	return output.String()
}

// safeContentSelector limits parent-document plugin CSS to rendered prose typography.
func safeContentSelector(selector string) bool {
	switch selector {
	case ".prose", ".prose code", ".prose pre":
		return true
	default:
		return false
	}
}

// contentStyleProperty reports whether a property may affect rendered-content typography.
func contentStyleProperty(property string) bool {
	switch property {
	case "font-family", "font-feature-settings", "font-variant-ligatures":
		return true
	default:
		return false
	}
}

// safeTypographyValue rejects CSS constructs that can load resources or escape a declaration.
func safeTypographyValue(value string) bool {
	if len(value) == 0 || len(value) > 512 {
		return false
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{"url(", "expression(", "@", "{", "}", ";", "<", ">", "\\"} {
		if strings.Contains(lower, forbidden) {
			return false
		}
	}
	return true
}
