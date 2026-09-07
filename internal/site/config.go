package site

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gi8lino/lore/themes"
	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigPath = "lore-site.toml"

// Config contains filesystem-backed static site build settings.
type Config struct {
	Logo       string `toml:"logo"`
	Favicon    string `toml:"favicon"`
	FaviconICO string `toml:"favicon_ico"`
	AssetsDir  string `toml:"assets_dir"`
	SiteName   string `toml:"site_name"`
	SiteURL    string `toml:"site_url"`
	SourceDir  string `toml:"source_dir"`
	OutputDir  string `toml:"output_dir"`
	Theme      string `toml:"theme"`
	Language   string `toml:"language"`
	Mermaid    bool   `toml:"mermaid"`
}

// DefaultConfig returns a useful zero-infrastructure documentation setup.
func DefaultConfig() Config {
	return Config{
		SiteName:  "Lore",
		SourceDir: "docs",
		OutputDir: "site",
		Theme:     themes.DefaultTheme,
		Language:  "en",
		Mermaid:   true,
	}
}

// LoadConfig reads an optional TOML site configuration.
func LoadConfig(path string, required bool) (Config, error) {
	config := DefaultConfig()
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) && !required {
		return config, nil
	}
	if err != nil {
		return Config{}, err
	}

	defer file.Close() // nolint:errcheck

	if err := toml.NewDecoder(file).DisallowUnknownFields().Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}

	for _, filename := range []*string{&config.Logo, &config.Favicon, &config.FaviconICO, &config.AssetsDir} {
		if *filename != "" && !filepath.IsAbs(*filename) {
			*filename = filepath.Join(filepath.Dir(path), *filename)
		}
	}
	return config, config.validate()
}

func (c Config) validate() error {
	switch {
	case strings.TrimSpace(c.SiteName) == "":
		return errors.New("site_name is required")
	case strings.TrimSpace(c.SourceDir) == "":
		return errors.New("source_dir is required")
	case strings.TrimSpace(c.OutputDir) == "":
		return errors.New("output_dir is required")
	case strings.TrimSpace(c.Theme) == "":
		return errors.New("theme is required")
	case strings.TrimSpace(c.Language) == "":
		return errors.New("language is required")
	}

	source, err := filepath.Abs(c.SourceDir)
	if err != nil {
		return err
	}

	output, err := filepath.Abs(c.OutputDir)
	if err != nil {
		return err
	}
	if directoriesOverlap(source, output) {
		return errors.New("source_dir and output_dir must be separate directories")
	}

	for _, asset := range []struct {
		name, filename string
		directory      bool
	}{
		{"logo", c.Logo, false}, {"favicon", c.Favicon, false},
		{"favicon_ico", c.FaviconICO, false}, {"assets_dir", c.AssetsDir, true},
	} {
		if asset.filename == "" {
			continue
		}
		absolute, err := filepath.Abs(asset.filename)
		if err != nil {
			return err
		}
		if absolute == output || directoryContains(output, absolute) || (asset.directory && directoryContains(absolute, output)) {
			return fmt.Errorf("%s must be separate from output_dir", asset.name)
		}
		info, err := os.Stat(asset.filename)
		if err != nil {
			return fmt.Errorf("%s: %w", asset.name, err)
		}
		if asset.directory && !info.IsDir() {
			return fmt.Errorf("%s must be a directory", asset.name)
		}
		if !asset.directory && !info.Mode().IsRegular() {
			return fmt.Errorf("%s must be a regular file", asset.name)
		}
	}
	for name, filename := range map[string]string{"logo": c.Logo, "favicon": c.Favicon} {
		if filename == "" {
			continue
		}
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico":
		default:
			return fmt.Errorf("%s must be an SVG, PNG, JPEG, WebP, GIF, or ICO image", name)
		}
	}
	if c.FaviconICO != "" && !strings.EqualFold(filepath.Ext(c.FaviconICO), ".ico") {
		return errors.New("favicon_ico must be an ICO file")
	}
	return nil
}

// directoriesOverlap reports whether either directory is equal to or contains the other.
func directoriesOverlap(left, right string) bool {
	if left == right {
		return true
	}
	if directoryContains(left, right) {
		return true
	}

	return directoryContains(right, left)
}

func directoryContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil || relative == "." {
		return false
	}

	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
