package site

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/icons"
	"github.com/gi8lino/lore/internal/logging"
	"github.com/gi8lino/lore/themes"
	"github.com/pelletier/go-toml/v2"
)

const defaultConfigPath = "lore-site.toml"

// Config contains filesystem-backed static site build settings.
type Config struct {
	Logo              string                `toml:"logo"`
	Favicon           string                `toml:"favicon"`
	FaviconICO        string                `toml:"favicon_ico"`
	AssetsDir         string                `toml:"assets_dir"`
	SiteName          string                `toml:"site_name"`
	SiteURL           string                `toml:"site_url"`
	SourceDir         string                `toml:"source_dir"`
	OutputDir         string                `toml:"output_dir"`
	Theme             string                `toml:"theme"`
	Language          string                `toml:"language"`
	NavigationStyle   string                `toml:"navigation_style"`
	NavigationDensity string                `toml:"navigation_density"`
	SidebarWidth      int                   `toml:"sidebar_width"`
	Mermaid           bool                  `toml:"mermaid"`
	RobotsPolicy      string                `toml:"robots"`
	ExternalLinks     []domain.ExternalLink `toml:"external_links"`
	logFormat         logging.LogFormat
}

// defaultConfig returns generic zero-infrastructure static site defaults.
func defaultConfig() Config {
	preferences := domain.DefaultUserPreferences()

	return Config{
		SiteName:          "Documentation",
		SourceDir:         "docs",
		OutputDir:         "site",
		Theme:             themes.DefaultTheme,
		Language:          "en",
		NavigationStyle:   preferences.NavigationStyle,
		NavigationDensity: preferences.NavigationDensity,
		SidebarWidth:      preferences.SidebarWidth,
		Mermaid:           true,
		RobotsPolicy:      domain.RobotsPolicyAllow,
		logFormat:         logging.LogFormatJSON,
	}
}

// loadConfig reads an optional TOML site configuration and resolves config-relative asset paths.
func loadConfig(filename string, required bool) (Config, error) {
	config := defaultConfig()
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

	config.resolveAssetPaths(filepath.Dir(filename))
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// resolveAssetPaths resolves user-supplied branding and asset paths relative to the config file.
func (c *Config) resolveAssetPaths(configDir string) {
	resolveRelativePath(configDir, &c.Logo)
	resolveRelativePath(configDir, &c.Favicon)
	resolveRelativePath(configDir, &c.FaviconICO)
	resolveRelativePath(configDir, &c.AssetsDir)
}

// resolveRelativePath resolves one non-empty relative path against the supplied directory.
func resolveRelativePath(baseDir string, filename *string) {
	if *filename == "" || filepath.IsAbs(*filename) {
		return
	}
	*filename = filepath.Clean(filepath.Join(baseDir, *filename))
}

// validate checks all static site configuration invariants before output is modified.
func (c *Config) validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}
	if err := c.validateBuildDirectories(); err != nil {
		return err
	}
	if err := c.validateAssetPaths(); err != nil {
		return err
	}
	if err := c.validateBrandingFormats(); err != nil {
		return err
	}
	if !domain.ValidNavigationStyle(c.NavigationStyle) {
		return errors.New("navigation_style must be sidebar, topbar, or tree")
	}
	if !domain.ValidNavigationDensity(c.NavigationDensity) {
		return errors.New("navigation_density must be comfortable or compact")
	}
	if !domain.ValidSidebarWidth(c.SidebarWidth) {
		return fmt.Errorf("sidebar_width must be between %d and %d pixels", domain.MinSidebarWidth, domain.MaxSidebarWidth)
	}
	if !domain.ValidRobotsPolicy(c.RobotsPolicy) {
		return errors.New("robots must be allow, disallow, or none")
	}
	return c.validateExternalLinks()
}

// validateExternalLinks normalizes and validates configurable top-bar links.
func (c *Config) validateExternalLinks() error {
	for index := range c.ExternalLinks {
		link := &c.ExternalLinks[index]
		link.Label = strings.TrimSpace(link.Label)
		link.URL = strings.TrimSpace(link.URL)
		link.Icon = strings.TrimSpace(link.Icon)
		link.Description = strings.TrimSpace(link.Description)
		link.HoverEffect = strings.TrimSpace(link.HoverEffect)
		link.HoverText = strings.TrimSpace(link.HoverText)

		if link.Label == "" {
			return fmt.Errorf("external_links[%d].label is required", index)
		}
		if !validExternalLinkURL(link.URL) {
			return fmt.Errorf("external_links[%d].url must be an absolute HTTP or HTTPS URL", index)
		}
		if link.Icon != "" && !icons.IsIcon(link.Icon) {
			return fmt.Errorf("external_links[%d].icon must be an available icon", index)
		}
		if !domain.ValidExternalLinkHoverEffect(link.HoverEffect) {
			return fmt.Errorf("external_links[%d].hover_effect must be highlight, lift, or none", index)
		}
		link.HoverEffect = domain.EffectiveExternalLinkHoverEffect(link.HoverEffect)
	}

	return nil
}

// validExternalLinkURL reports whether value is an absolute HTTP or HTTPS URL.
func validExternalLinkURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" {
		return false
	}

	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}

// validateRequiredFields rejects empty values required to build a complete site.
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

// validateBuildDirectories rejects source and output directory overlap.
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

// validateAssetPaths checks configured branding files and the optional asset directory.
func (c Config) validateAssetPaths() error {
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

// validateConfiguredPath checks one configured file or directory and prevents output overlap.
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

// pathOverlapsOutput reports whether a configured path would read from generated output.
func pathOverlapsOutput(pathname, output string, directory bool) bool {
	if pathname == output || directoryContains(output, pathname) {
		return true
	}
	return directory && directoryContains(pathname, output)
}

// validateBrandingFormats checks configured branding file extensions.
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

// validateImageFormat accepts browser-compatible image extensions for configurable branding.
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

// directoryContains reports whether child is nested below parent.
func directoryContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil || relative == "." {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// BindFlags registers static site flags and returns a resolver for the effective configuration.
func BindFlags(flags *tinyflags.FlagSet) func() (Config, error) {
	defaults := defaultConfig()

	configPath := flags.String("config", defaultConfigPath, "TOML site configuration file").Placeholder("FILE")
	siteName := flags.String("site-name", defaults.SiteName, "Site title").Placeholder("NAME")
	siteURL := flags.String("site-url", defaults.SiteURL, "Published site URL").Placeholder("URL")
	source := flags.String("source", defaults.SourceDir, "Markdown source directory").Placeholder("DIR")
	output := flags.String("output", defaults.OutputDir, "Generated site directory").Placeholder("DIR")
	theme := flags.String("theme", defaults.Theme, "Theme").Placeholder("THEME")
	language := flags.String("language", defaults.Language, "HTML content language").Placeholder("LANG")
	navigationStyle := flags.String("navigation-style", defaults.NavigationStyle, "Desktop navigation style").
		Choices(domain.NavigationStyleSidebar, domain.NavigationStyleTopbar, domain.NavigationStyleTree).
		Placeholder("STYLE")
	navigationDensity := flags.String("navigation-density", defaults.NavigationDensity, "Navigation density").
		Choices(domain.NavigationDensityComfortable, domain.NavigationDensityCompact).
		Placeholder("DENSITY")
	sidebarWidth := flags.Int("sidebar-width", defaults.SidebarWidth, "Desktop sidebar width in pixels").Placeholder("PIXELS")
	mermaid := flags.Bool("mermaid", defaults.Mermaid, "Enable Mermaid rendering").Strict()
	robots := flags.String("robots", defaults.RobotsPolicy, "robots.txt policy").
		Choices(domain.RobotsPolicyAllow, domain.RobotsPolicyDisallow, domain.RobotsPolicyNone).
		Placeholder("POLICY")
	logFormat := flags.String("log-format", string(defaults.logFormat), "Log output format").
		Choices(string(logging.LogFormatText), string(logging.LogFormatJSON)).
		Short("l").
		Placeholder("FORMAT")

	return func() (Config, error) {
		cfg, err := loadConfig(*configPath.Value(), configPath.Changed())
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
		if navigationStyle.Changed() {
			cfg.NavigationStyle = *navigationStyle.Value()
		}
		if navigationDensity.Changed() {
			cfg.NavigationDensity = *navigationDensity.Value()
		}
		if sidebarWidth.Changed() {
			cfg.SidebarWidth = *sidebarWidth.Value()
		}
		if mermaid.Changed() {
			cfg.Mermaid = *mermaid.Value()
		}
		if robots.Changed() {
			cfg.RobotsPolicy = *robots.Value()
		}
		cfg.logFormat = logging.LogFormat(*logFormat.Value())

		if err := cfg.validate(); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
}
