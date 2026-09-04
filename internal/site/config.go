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
	SiteName  string `toml:"site_name"`
	SiteURL   string `toml:"site_url"`
	SourceDir string `toml:"source_dir"`
	OutputDir string `toml:"output_dir"`
	Theme     string `toml:"theme"`
	Language  string `toml:"language"`
	Mermaid   bool   `toml:"mermaid"`
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
	return config, config.validate()
}

func (c Config) validate() error {
	if strings.TrimSpace(c.SiteName) == "" {
		return errors.New("site_name is required")
	}
	if strings.TrimSpace(c.SourceDir) == "" {
		return errors.New("source_dir is required")
	}
	if strings.TrimSpace(c.OutputDir) == "" {
		return errors.New("output_dir is required")
	}
	if strings.TrimSpace(c.Theme) == "" {
		return errors.New("theme is required")
	}
	if strings.TrimSpace(c.Language) == "" {
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
	if source == output || directoryContains(source, output) || directoryContains(output, source) {
		return errors.New("source_dir and output_dir must be separate directories")
	}
	return nil
}

func directoryContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil || relative == "." {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
