// Package plugincap adapts already-authorized application data to public wire
// values. It does not fetch an unrestricted store or expose domain objects.
package plugincap

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/icons"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/pluginapi"
)

// Source is the minimal authorized page catalog required by plugin page capabilities.
type Source interface {
	// Search queries the authorized source and converts results to public plugin values.
	Search(context.Context, string, int) ([]domain.Page, error)
	// GetPage returns one authorized page as a public plugin value.
	GetPage(context.Context, string) (domain.Page, error)
}

// Pages adapts an already-authorized page catalog to the public plugin capability API.
type Pages struct {
	// Source is the authorized page catalog used for plugin lookups.
	Source Source
}

// Search queries the authorized source and converts results to public plugin values.
func (p Pages) Search(ctx context.Context, query string, limit int) ([]pluginapi.Page, error) {
	pages, err := p.Source.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	result := make([]pluginapi.Page, 0, len(pages))
	for _, page := range pages {
		result = append(result, pageValue(page))
	}

	return result, nil
}

// GetPage returns one authorized page as a public plugin value.
func (p Pages) GetPage(ctx context.Context, slug string) (pluginapi.Page, error) {
	page, err := p.Source.GetPage(ctx, slug)
	return pageValue(page), err
}

// pageValue converts an internal page record into its public plugin representation.
func pageValue(page domain.Page) pluginapi.Page {
	result := pluginapi.Page{
		Slug:       page.Slug,
		Title:      page.Title,
		Status:     page.Status,
		OwnerGroup: page.OwnerGroup,
		UpdatedAt:  page.UpdatedAt,
		Author:     page.Author,
		Tags:       page.Tags,
		ViewCount:  page.ViewCount,
	}

	for _, property := range page.Properties {
		result.Properties = append(result.Properties, pluginapi.Property{Key: property.Key, Value: property.Value})
	}

	return result
}

// Navigation exposes prepared navigation nodes through the public plugin capability API.
func Navigation(nodes []navigation.Node, pageURL func(string) string) []pluginapi.NavigationNode {
	result := make([]pluginapi.NavigationNode, 0, len(nodes))

	for _, node := range nodes {
		item := pluginapi.NavigationNode{Title: node.Title, Icon: node.Icon, Page: node.Page, Children: Navigation(node.Children, pageURL)}
		if node.Page {
			item.URL = pageURL(node.Slug)
		}
		result = append(result, item)
	}

	return result
}

// Capabilities builds the render-scoped capability map supplied to plugins.
func Capabilities(source Source, nodes []pluginapi.NavigationNode) map[string]plugin.Capability {
	result := map[string]plugin.Capability{
		"pages.navigation": func(context.Context, json.RawMessage) (any, error) { return nodes, nil },
		"icons.render": func(_ context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.IconRequest
			if err := json.Unmarshal(data, &request); err != nil || !validIconRequest(request) {
				return nil, errors.New("invalid icon request")
			}
			return string(icons.SVG(request.Name, request.Size)), nil
		},
	}

	if source != nil {
		pages := Pages{source}
		result["pages.get"] = func(ctx context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.PageRef
			if err := json.Unmarshal(data, &request); err != nil || !validPageRef(request) {
				return nil, errors.New("invalid page reference")
			}
			return pages.GetPage(ctx, request.Slug)
		}
		result["pages.search"] = func(ctx context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.PageQuery
			if err := json.Unmarshal(data, &request); err != nil || !validPageQuery(request) {
				return nil, errors.New("invalid page query")
			}
			return pages.Search(ctx, request.Query, request.Limit)
		}
	}

	return result
}

// validIconRequest reports whether an icon capability request stays within supported bounds.
func validIconRequest(request pluginapi.IconRequest) bool {
	return len(request.Name) <= 128 && request.Size >= 1 && request.Size <= 256
}

// validPageRef reports whether a page reference contains a bounded non-empty slug.
func validPageRef(request pluginapi.PageRef) bool {
	return len(request.Slug) > 0 && len(request.Slug) <= 4096
}

// validPageQuery reports whether a page search request stays within supported bounds.
func validPageQuery(request pluginapi.PageQuery) bool {
	return len(request.Query) <= 4096 && request.Limit >= 1 && request.Limit <= 100
}

// SharedPages constrains anonymous capability calls to the explicitly shared
// page. Knowing another slug or matching it in search never grants access.
type SharedPages struct {
	// Source is the authorized page catalog used for plugin lookups.
	Source Source
	// Slug is the only page path this constrained source may expose.
	Slug string
}

// GetPage returns one authorized page as a public plugin value.
func (s SharedPages) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	if slug != s.Slug {
		return domain.Page{}, errors.New("page unavailable")
	}

	return s.Source.GetPage(ctx, slug)
}

// Search queries the authorized source and converts results to public plugin values.
func (s SharedPages) Search(ctx context.Context, query string, limit int) ([]domain.Page, error) {
	pages, err := s.Source.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	var result []domain.Page
	for _, page := range pages {
		if page.Slug == s.Slug {
			result = append(result, page)
		}
	}

	return result, nil
}
