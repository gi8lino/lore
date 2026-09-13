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

type Source interface {
	Search(context.Context, string, int) ([]domain.Page, error)
	GetPage(context.Context, string) (domain.Page, error)
}
type Pages struct{ Source Source }

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
func (p Pages) GetPage(ctx context.Context, slug string) (pluginapi.Page, error) {
	page, err := p.Source.GetPage(ctx, slug)
	return pageValue(page), err
}
func pageValue(page domain.Page) pluginapi.Page {
	result := pluginapi.Page{Slug: page.Slug, Title: page.Title, Status: page.Status, OwnerGroup: page.OwnerGroup, UpdatedAt: page.UpdatedAt, Author: page.Author, Tags: page.Tags, ViewCount: page.ViewCount}
	for _, property := range page.Properties {
		result.Properties = append(result.Properties, pluginapi.Property{Key: property.Key, Value: property.Value})
	}
	return result
}
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
func Capabilities(source Source, nodes []pluginapi.NavigationNode) map[string]plugin.Capability {
	result := map[string]plugin.Capability{
		"pages.navigation": func(context.Context, json.RawMessage) (any, error) { return nodes, nil },
		"icons.render": func(_ context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.IconRequest
			if err := json.Unmarshal(data, &request); err != nil || len(request.Name) > 128 || request.Size < 1 || request.Size > 256 {
				return nil, errors.New("invalid icon request")
			}
			return string(icons.SVG(request.Name, request.Size)), nil
		},
	}
	if source != nil {
		pages := Pages{source}
		result["pages.get"] = func(ctx context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.PageRef
			if err := json.Unmarshal(data, &request); err != nil || len(request.Slug) == 0 || len(request.Slug) > 4096 {
				return nil, errors.New("invalid page reference")
			}
			return pages.GetPage(ctx, request.Slug)
		}
		result["pages.search"] = func(ctx context.Context, data json.RawMessage) (any, error) {
			var request pluginapi.PageQuery
			if err := json.Unmarshal(data, &request); err != nil || len(request.Query) > 4096 || request.Limit < 1 || request.Limit > 100 {
				return nil, errors.New("invalid page query")
			}
			return pages.Search(ctx, request.Query, request.Limit)
		}
	}
	return result
}

// SharedPages constrains anonymous capability calls to the explicitly shared
// page. Knowing another slug or matching it in search never grants access.
type SharedPages struct {
	Source Source
	Slug   string
}

func (s SharedPages) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	if slug != s.Slug {
		return domain.Page{}, errors.New("page unavailable")
	}
	return s.Source.GetPage(ctx, slug)
}
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
