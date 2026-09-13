// Package pluginpackage reads bounded, versioned .loreplugin ZIP archives.
// Archives are kept in memory and never extracted into the host filesystem.
package pluginpackage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
	"go.yaml.in/yaml/v3"
)

const (
	MaxArchiveBytes  = 16 << 20
	MaxExpandedBytes = 32 << 20
	MaxWASMBytes     = 16 << 20
	MaxAssetBytes    = 4 << 20
	MaxManifestBytes = 64 << 10
	MaxFiles         = 256
)

// Manifest is the complete v1 package manifest. Unknown fields and unsupported
// modules or permissions are rejected rather than silently ignored.
type Manifest struct {
	APIVersion     int      `yaml:"api_version"`
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Version        string   `yaml:"version"`
	Description    string   `yaml:"description,omitempty"`
	DefaultEnabled bool     `yaml:"default_enabled,omitempty"`
	Requires       []string `yaml:"requires,omitempty"`
	Modules        []Module `yaml:"modules"`
	Permissions    []string `yaml:"permissions"`
}

type Module struct {
	Type       string `yaml:"type"`
	ID         string `yaml:"id"`
	Stage      string `yaml:"stage,omitempty"`
	Name       string `yaml:"name,omitempty"`
	Capability string `yaml:"capability,omitempty"`
}

// Package exposes copies of validated content so callers cannot mutate the
// package after validation. The digest covers the original ZIP bytes.
type Package struct {
	manifest Manifest
	wasm     []byte
	assets   map[string][]byte
	digest   [32]byte
}

func (p *Package) Manifest() Manifest {
	m := p.manifest
	m.Modules = slices.Clone(m.Modules)
	m.Requires = slices.Clone(m.Requires)
	m.Permissions = slices.Clone(m.Permissions)
	return m
}
func (p *Package) WASM() []byte         { return bytes.Clone(p.wasm) }
func (p *Package) Digest() [32]byte     { return p.digest }
func (p *Package) AssetNames() []string { return slices.Sorted(maps.Keys(p.assets)) }
func (p *Package) Asset(name string) ([]byte, error) {
	if !validPath(name) {
		return nil, fs.ErrInvalid
	}
	content, ok := p.assets[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return bytes.Clone(content), nil
}

// Read validates paths, types, decompression limits, CRCs, and manifest fields
// before returning any package. Asset names are relative to assets/.
func Read(data []byte) (*Package, error) {
	if len(data) == 0 || len(data) > MaxArchiveBytes {
		return nil, errors.New("plugin archive exceeds size limit or is empty")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read plugin archive: %w", err)
	}
	if len(archive.File) > MaxFiles {
		return nil, errors.New("too many plugin archive entries")
	}
	files := make(map[string][]byte)
	seen := make(map[string]bool)
	total := 0
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if !validPath(name) || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate plugin path %q", file.Name)
		}
		seen[name] = true
		if kind := file.Mode().Type(); kind != 0 && kind != fs.ModeDir {
			return nil, fmt.Errorf("plugin entry %q is not a regular file or directory", name)
		}
		if file.FileInfo().IsDir() {
			if !strings.HasSuffix(file.Name, "/") || file.UncompressedSize64 != 0 {
				return nil, fmt.Errorf("invalid plugin directory %q", name)
			}
			if name != "assets" && !strings.HasPrefix(name, "assets/") {
				return nil, fmt.Errorf("unsupported plugin directory %q", name)
			}
			continue
		}
		limit, err := entryLimit(name)
		if err != nil {
			return nil, err
		}
		content, err := readEntry(file, min(limit, MaxExpandedBytes-total))
		if err != nil {
			return nil, err
		}
		total += len(content)
		files[name] = content
	}
	// A file cannot also be an ancestor directory of another entry.
	for name := range seen {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, fmt.Errorf("plugin path %q has a file as parent", name)
			}
		}
	}
	manifest, err := decodeManifest(files["plugin.yaml"])
	if err != nil {
		return nil, err
	}
	wasm := files["plugin.wasm"]
	if len(wasm) < 8 || !bytes.Equal(wasm[:8], []byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0}) {
		return nil, errors.New("missing or invalid plugin.wasm")
	}
	assets := make(map[string][]byte)
	for name, content := range files {
		if asset, ok := strings.CutPrefix(name, "assets/"); ok {
			assets[asset] = content
		}
	}
	return &Package{manifest: manifest, wasm: wasm, assets: assets, digest: sha256.Sum256(data)}, nil
}

func validPath(name string) bool {
	return len(name) <= 512 && name != "." && fs.ValidPath(name) && !strings.ContainsAny(name, "\\:\x00")
}

func entryLimit(name string) (int, error) {
	switch {
	case name == "plugin.yaml":
		return MaxManifestBytes, nil
	case name == "plugin.wasm":
		return MaxWASMBytes, nil
	case strings.HasPrefix(name, "assets/"):
		return MaxAssetBytes, nil
	default:
		return 0, fmt.Errorf("unsupported plugin entry %q", name)
	}
}

func readEntry(file *zip.File, limit int) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("plugin entry %q exceeds size limit", file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("read plugin entry %q: %w", file.Name, err)
	}
	if len(content) > limit {
		return nil, fmt.Errorf("plugin entry %q exceeds size limit", file.Name)
	}
	return content, nil
}

func decodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("read plugin manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return manifest, errors.New("plugin manifest must contain exactly one YAML document")
	}
	return manifest, manifest.Validate()
}

var identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
var version = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func (m Manifest) Validate() error {
	if m.APIVersion != pluginapi.Version {
		return fmt.Errorf("unsupported plugin API version %d", m.APIVersion)
	}
	if !identifier.MatchString(m.ID) || strings.TrimSpace(m.Name) == "" || len(m.Name) > 128 || !version.MatchString(m.Version) {
		return errors.New("invalid plugin identity or version")
	}
	if len(m.Description) > 4096 {
		return errors.New("plugin description is too long")
	}
	permissions := make(map[string]bool)
	for _, permission := range m.Permissions {
		if !pluginapi.ValidPermission(permission) || permissions[permission] {
			return errors.New("invalid or duplicate plugin permission")
		}
		permissions[permission] = true
	}
	if len(m.Modules) == 0 || len(m.Modules) > 32 {
		return errors.New("plugin must declare between 1 and 32 modules")
	}
	names := make(map[string]bool)
	for _, module := range m.Modules {
		if !identifier.MatchString(module.ID) || names[module.ID] {
			return fmt.Errorf("invalid or duplicate module ID %q", module.ID)
		}
		names[module.ID] = true
		if !validModule(module) {
			return fmt.Errorf("unsupported plugin module %q", module.ID)
		}
	}
	dependencies := make(map[string]bool)
	for _, id := range m.Requires {
		if !identifier.MatchString(id) || id == m.ID || dependencies[id] {
			return fmt.Errorf("invalid plugin dependency %q", id)
		}
		dependencies[id] = true
	}
	return nil
}

func validModule(m Module) bool {
	switch m.Type {
	case "renderer-extension":
		return (m.Stage == "preprocess" || m.Stage == "postprocess") && m.Name == "" && m.Capability == ""
	case "macro":
		if m.Capability != "" {
			if _, ok := pluginapi.PermissionFor(m.Capability); !ok {
				return false
			}
		}
		return identifier.MatchString(m.Name) && m.Stage == ""
	default:
		return false
	}
}
