package pluginapi

// ValidRenderPolicy identifies rendering policies that plugins may request from
// the host without receiving access to Lore internals.
func ValidRenderPolicy(name string) bool {
	switch name {
	case "coding-ligatures", "syntax-highlighting":
		return true
	default:
		return false
	}
}
