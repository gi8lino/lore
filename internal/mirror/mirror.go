// Package mirror exports the PostgreSQL-backed Lore state as a deterministic,
// Git-friendly directory tree without changing the database source of truth.
package mirror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/logging"
	"github.com/gi8lino/lore/internal/store"
)

const defaultOutputDir = "lore-mirror"

// Config contains database mirror settings.
type Config struct {
	DatabaseURL string
	OutputDir   string
	LogFormat   logging.LogFormat
}

// repository contains the database reads needed by the mirror exporter.
type repository interface {
	PageInventory(context.Context) ([]domain.Page, error)
	GetPage(context.Context, string) (domain.Page, error)
	Images(context.Context) ([]domain.Image, error)
	ImageContent(context.Context, int64) (domain.ImageData, error)
	Attachments(context.Context) ([]domain.Attachment, error)
	AttachmentContent(context.Context, int64) (domain.AttachmentData, error)
}

// pageMetadata is the portable sidecar written next to mirrored page content.
type pageMetadata struct {
	ID                 int64                 `json:"id"`
	Slug               string                `json:"slug"`
	Title              string                `json:"title"`
	Icon               string                `json:"icon,omitempty"`
	Language           string                `json:"language,omitempty"`
	CreatedBy          int64                 `json:"created_by"`
	UpdatedBy          int64                 `json:"updated_by"`
	Author             string                `json:"author,omitempty"`
	CreatedAt          string                `json:"created_at"`
	UpdatedAt          string                `json:"updated_at"`
	Tags               []string              `json:"tags,omitempty"`
	Groups             []domain.Group        `json:"groups,omitempty"`
	ViewCount          int64                 `json:"view_count"`
	Status             string                `json:"status"`
	OwnerGroupID       int64                 `json:"owner_group_id,omitempty"`
	OwnerGroup         string                `json:"owner_group,omitempty"`
	LastReviewedAt     string                `json:"last_reviewed_at,omitempty"`
	ReviewIntervalDays int                   `json:"review_interval_days,omitempty"`
	DeprecatedTarget   string                `json:"deprecated_target,omitempty"`
	Properties         []domain.PageProperty `json:"properties,omitempty"`
}

// manifest describes the stable mirror format and exported object inventory.
type manifest struct {
	Format      int      `json:"format"`
	Pages       []string `json:"pages"`
	Images      []int64  `json:"images,omitempty"`
	Attachments []int64  `json:"attachments,omitempty"`
}

// BindFlags registers lore mirror flags and returns the parsed configuration.
func BindFlags(flags *tinyflags.FlagSet) func() Config {
	cfg := Config{OutputDir: defaultOutputDir, LogFormat: logging.LogFormatText}
	flags.EnvPrefix("LORE_")
	flags.StringVar(&cfg.DatabaseURL, "database-url", "", "PostgreSQL connection URL").
		Required().
		Placeholder("URL").
		OverriddenValueMaskFn(tinyflags.MaskPostgresURL).
		Value()
	flags.StringVar(&cfg.OutputDir, "output", defaultOutputDir, "Directory that receives the Git-friendly mirror").
		Placeholder("DIR").
		Value()
	logFormat := flags.String("log-format", string(cfg.LogFormat), "Log output format").
		Choices(string(logging.LogFormatText), string(logging.LogFormatJSON)).
		Short("l").
		Placeholder("FORMAT")

	return func() Config {
		cfg.LogFormat = logging.LogFormat(*logFormat.Value())
		return cfg
	}
}

// Run opens PostgreSQL and writes a complete mirror into the configured output directory.
func Run(ctx context.Context, cfg Config, stdout io.Writer) error {
	if err := validateOutputDir(cfg.OutputDir); err != nil {
		return err
	}

	logger := logging.Setup(cfg.LogFormat, false, stdout).With("component", "mirror")
	database, err := store.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := Export(ctx, database, cfg.OutputDir); err != nil {
		return err
	}

	logger.Info("mirror complete", "event", "mirror_complete", "output", cfg.OutputDir)
	return nil
}

// Export writes repository content atomically into outputDir.
func Export(ctx context.Context, repository repository, outputDir string) error {
	if err := validateOutputDir(outputDir); err != nil {
		return err
	}

	absolute, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	temporary, err := os.MkdirTemp(parent, ".lore-mirror-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary) // nolint:errcheck

	if err := exportInto(ctx, repository, temporary); err != nil {
		return err
	}
	if err := os.RemoveAll(absolute); err != nil {
		return err
	}
	return os.Rename(temporary, absolute)
}

func exportInto(ctx context.Context, repository repository, outputDir string) error {
	for _, directory := range []string{"pages", "metadata", "media", "attachments"} {
		if err := os.MkdirAll(filepath.Join(outputDir, directory), 0o755); err != nil {
			return err
		}
	}

	inventory, err := repository.PageInventory(ctx)
	if err != nil {
		return err
	}
	slices.SortFunc(inventory, func(left, right domain.Page) int { return strings.Compare(left.Slug, right.Slug) })

	result := manifest{Format: 1, Pages: make([]string, 0, len(inventory))}
	for _, summary := range inventory {
		page, err := repository.GetPage(ctx, summary.Slug)
		if err != nil {
			return fmt.Errorf("read page %q: %w", summary.Slug, err)
		}
		if err := writePage(outputDir, page); err != nil {
			return err
		}
		result.Pages = append(result.Pages, page.Slug)
	}

	images, err := repository.Images(ctx)
	if err != nil {
		return err
	}
	slices.SortFunc(images, func(left, right domain.Image) int { return compareID(left.ID, right.ID) })
	for _, image := range images {
		data, err := repository.ImageContent(ctx, image.ID)
		if err != nil {
			return fmt.Errorf("read image %d: %w", image.ID, err)
		}
		if err := writeBinary(outputDir, "media", image.ID, data.Filename, data.Data); err != nil {
			return err
		}
		result.Images = append(result.Images, image.ID)
	}

	attachments, err := repository.Attachments(ctx)
	if err != nil {
		return err
	}
	slices.SortFunc(attachments, func(left, right domain.Attachment) int { return compareID(left.ID, right.ID) })
	for _, attachment := range attachments {
		data, err := repository.AttachmentContent(ctx, attachment.ID)
		if err != nil {
			return fmt.Errorf("read attachment %d: %w", attachment.ID, err)
		}
		if err := writeBinary(outputDir, "attachments", attachment.ID, data.Filename, data.Data); err != nil {
			return err
		}
		result.Attachments = append(result.Attachments, attachment.ID)
	}

	return writeJSON(filepath.Join(outputDir, "manifest.json"), result)
}

func writePage(outputDir string, page domain.Page) error {
	relative := cleanSlug(page.Slug)
	markdownPath := filepath.Join(outputDir, "pages", filepath.FromSlash(relative)+".md")
	if err := writeFile(markdownPath, []byte(page.Markdown)); err != nil {
		return fmt.Errorf("write page %q: %w", page.Slug, err)
	}

	metadata := pageMetadata{
		ID:                 page.ID,
		Slug:               page.Slug,
		Title:              page.Title,
		Icon:               page.Icon,
		Language:           page.Language,
		CreatedBy:          page.CreatedBy,
		UpdatedBy:          page.UpdatedBy,
		Author:             page.Author,
		CreatedAt:          page.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z"),
		UpdatedAt:          page.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z"),
		Tags:               page.Tags,
		Groups:             page.Groups,
		ViewCount:          page.ViewCount,
		Status:             page.Status,
		OwnerGroupID:       page.OwnerGroupID,
		OwnerGroup:         page.OwnerGroup,
		ReviewIntervalDays: page.ReviewIntervalDays,
		DeprecatedTarget:   page.DeprecatedTarget,
		Properties:         page.Properties,
	}
	if page.LastReviewedAt != nil {
		metadata.LastReviewedAt = page.LastReviewedAt.UTC().Format("2006-01-02T15:04:05.000000000Z")
	}
	return writeJSON(filepath.Join(outputDir, "metadata", filepath.FromSlash(relative)+".json"), metadata)
}

func writeBinary(outputDir, kind string, id int64, filename string, data []byte) error {
	name := filepath.Base(filepath.FromSlash(filename))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "file"
	}
	return writeFile(filepath.Join(outputDir, kind, fmt.Sprint(id), name), data)
}

func writeJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(filename, data)
}

func writeFile(filename string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0o644)
}

func cleanSlug(slug string) string {
	cleaned := strings.Trim(strings.TrimSpace(slug), "/")
	if cleaned == "" {
		return "index"
	}
	return cleaned
}

func compareID(left, right int64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func validateOutputDir(outputDir string) error {
	if strings.TrimSpace(outputDir) == "" {
		return errors.New("mirror output directory is required")
	}
	absolute, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	root := filepath.VolumeName(absolute) + string(filepath.Separator)
	if absolute == root {
		return errors.New("mirror output directory cannot be a filesystem root")
	}
	current, err := os.Getwd()
	if err != nil {
		return err
	}
	current, err = filepath.Abs(current)
	if err != nil {
		return err
	}
	if absolute == current {
		return errors.New("mirror output directory cannot be the current working directory")
	}
	return nil
}
