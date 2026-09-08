package site

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/logging"
	"github.com/gi8lino/lore/themes"
	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigPath = "site.toml"

// Config contains filesystem-backed static site build settings.
type Config struct {
	Logo       string            `toml:"logo"`
	Favicon    string            `toml:"favicon"`
	FaviconICO string            `toml:"favicon_ico"`
	AssetsDir  string            `toml:"assets_dir"`
	SiteName   string            `toml:"site_name"`
	SiteURL    string            `toml:"site_url"`
	SourceDir  string            `toml:"source_dir"`
	OutputDir  string            `toml:"output_dir"`
	Theme      string            `toml:"theme"`
	Language   string            `toml:"language"`
	Mermaid    bool              `toml:"mermaid"`
	LogFormat  logging.LogFormat `toml:"-"`
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
		LogFormat: logging.LogFormatText,
	}
}

// LoadConfig reads an optional TOML site configuration.
func LoadConfig(filename string, required bool) (Config, error) {
	config := DefaultConfig()
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) && !required {
		return config, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer file.Close() // nolint:errcheck

	if err := toml.NewDecoder(file).DisallowUnknownFields().Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	config.resolveBrandingPaths(filepath.Dir(filename))
	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *Config) resolveBrandingPaths(configDir string) {
	resolveRelativePath(configDir, &c.Logo)
	resolveRelativePath(configDir, &c.Favicon)
	resolveRelativePath(configDir, &c.FaviconICO)
	resolveRelativePath(configDir, &c.AssetsDir)
}

func resolveRelativePath(baseDir string, filename *string) {
	if *filename == "" || filepath.IsAbs(*filename) {
		return
	}

	*filename = filepath.Clean(filepath.Join(baseDir, *filename))
}

func (c Config) validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}
	if err := c.validateBuildDirectories(); err != nil {
		return err
	}
	if err := c.validateBrandingPaths(); err != nil {
		return err
	}

	return c.validateBrandingFormats()
}

func (c Config) validateRequiredFields() error {
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
	default:
		return nil
	}
}

func (c Config) validateBuildDirectories() error {
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

	return nil
}

func (c Config) validateBrandingPaths() error {
	if err := validateConfiguredPath("logo", c.Logo, c.OutputDir, false); err != nil {
		return err
	}
	if err := validateConfiguredPath("favicon", c.Favicon, c.OutputDir, false); err != nil {
		return err
	}
	if err := validateConfiguredPath("favicon_ico", c.FaviconICO, c.OutputDir, false); err != nil {
		return err
	}

	return validateConfiguredPath("assets_dir", c.AssetsDir, c.OutputDir, true)
}

func validateConfiguredPath(name, filename, outputDir string, directory bool) error {
	if filename == "" {
		return nil
	}

	absolute, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	output, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	if pathOverlapsOutput(absolute, output, directory) {
		return fmt.Errorf("%s must be separate from output_dir", name)
	}

	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if directory && !info.IsDir() {
		return fmt.Errorf("%s must be a directory", name)
	}
	if !directory && !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", name)
	}

	return nil
}

func pathOverlapsOutput(pathname, output string, directory bool) bool {
	if pathname == output || directoryContains(output, pathname) {
		return true
	}

	return directory && directoryContains(pathname, output)
}

func (c Config) validateBrandingFormats() error {
	if err := validateImageFormat("logo", c.Logo); err != nil {
		return err
	}
	if err := validateImageFormat("favicon", c.Favicon); err != nil {
		return err
	}
	if c.FaviconICO != "" && !strings.EqualFold(filepath.Ext(c.FaviconICO), ".ico") {
		return errors.New("favicon_ico must be an ICO file")
	}

	return nil
}

func validateImageFormat(name, filename string) error {
	if filename == "" {
		return nil
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico":
		return nil
	default:
		return fmt.Errorf("%s must be an SVG, PNG, JPEG, WebP, GIF, or ICO image", name)
	}
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

// BindFlags registers the lore build flags and returns a resolver for the parsed Config.
func BindFlags(flags *tinyflags.FlagSet) func() (Config, error) {
	defaults := DefaultConfig()

	configPath := flags.String("config", DefaultConfigPath, "TOML site configuration file").
		Placeholder("FILE")
	siteName := flags.String("site-name", defaults.SiteName, "Site title").
		Placeholder("NAME")
	siteURL := flags.String("site-url", defaults.SiteURL, "Published site URL").
		Placeholder("URL")
	source := flags.String("source", defaults.SourceDir, "Markdown source directory").
		Placeholder("DIR")
	output := flags.String("output", defaults.OutputDir, "Generated site directory").
		Placeholder("DIR")
	theme := flags.String("theme", defaults.Theme, "Lore theme").
		Placeholder("THEME")
	language := flags.String("language", defaults.Language, "HTML content language").
		Placeholder("LANG")
	mermaid := flags.Bool("mermaid", defaults.Mermaid, "Enable Mermaid rendering").Strict()
	logFormat := tinyflags.Enum(flags, "log-format", logging.LogFormatText, "Log output format", logging.LogFormatText, logging.LogFormatJSON).
		Short("l").
		Placeholder("FORMAT")

	return func() (Config, error) {
		cfg, err := LoadConfig(*configPath.Value(), configPath.Changed())
		if err != nil {
			return Config{}, err
		}

		if siteName.Changed() {
			cfg.SiteName = *siteName.Value()
		}
		if siteURL.Changed() {
			cfg.SiteURL = *siteURL.Value()
		}
		if source.Changed() {
			cfg.SourceDir = *source.Value()
		}
		if output.Changed() {
			cfg.OutputDir = *output.Value()
		}
		if theme.Changed() {
			cfg.Theme = *theme.Value()
		}
		if language.Changed() {
			cfg.Language = *language.Value()
		}
		if mermaid.Changed() {
			cfg.Mermaid = *mermaid.Value()
		}

		cfg.LogFormat = *logFormat.Value()

		if err := cfg.validate(); err != nil {
			return Config{}, err
		}

		return cfg, nil
	}
}
