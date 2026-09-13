package pluginapi

import (
	"encoding/json"
	"time"
)

// CapabilityRequest never carries a caller identity. Lore supplies the identity
// and grants from the executing instance, and the viewer from the render scope.
type CapabilityRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}
type CapabilityResponse struct {
	Value json.RawMessage `json:"value,omitempty"`
	Error string          `json:"error,omitempty"`
}
type Page struct {
	Slug       string
	Title      string
	Status     string
	OwnerGroup string
	UpdatedAt  time.Time
	Author     string
	Tags       []string
	ViewCount  int64
	Properties []Property
}
type Property struct{ Key, Value string }
type NavigationNode struct {
	Title, Icon, URL string
	Page             bool
	Children         []NavigationNode
}
type PageQuery struct {
	Query string
	Limit int
}
type PageRef struct{ Slug string }
type StorageValue struct {
	Key   string
	Value []byte
}
type StoredValue struct {
	Value []byte
	Found bool
}
type IconRequest struct {
	Name string
	Size int
}
type LogMessage struct{ Message string }

// PermissionFor is the closed set of host operations supported by API v1.
// An empty permission is an explicitly public, non-sensitive operation.
func PermissionFor(method string) (string, bool) {
	switch method {
	case "pages.get", "pages.search", "pages.navigation":
		return "pages:read", true
	case "attachments.read":
		return "attachments:read", true
	case "plugin.settings.read":
		return "settings:read", true
	case "plugin.settings.write":
		return "settings:write", true
	case "plugin.storage.read":
		return "storage:read", true
	case "plugin.storage.write":
		return "storage:write", true
	case "icons.render", "log":
		return "", true
	default:
		return "", false
	}
}
func ValidPermission(permission string) bool {
	switch permission {
	case "pages:read", "attachments:read", "settings:read", "settings:write", "storage:read", "storage:write":
		return true
	default:
		return false
	}
}

// AttachmentRead selects a bounded byte range; a request scope must explicitly
// supply an authorized attachment reader before this operation is available.
type AttachmentRead struct {
	ID     int64
	Offset int64
	Length int
}
type Attachment struct {
	Filename, ContentType string
	Size                  int64
	Data                  []byte
}
