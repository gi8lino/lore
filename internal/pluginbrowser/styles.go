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

// PresentationStyles publishes only scoped color declarations from active
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
