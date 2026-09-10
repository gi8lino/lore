package handler

import (
	"context"

	"github.com/gi8lino/lore/internal/domain"
)

// accessiblePageCatalog limits page-report and include reads to one user's access.
type accessiblePageCatalog struct {
	catalog pageReportCatalogService
	access  pageAccessReader
	user    domain.User
}

// GetPage returns the requested page only when the current user may view it.
func (c accessiblePageCatalog) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	allowed, err := c.access.CanView(ctx, c.user, slug)
	if err != nil {
		return domain.Page{}, err
	}
	if !allowed {
		return domain.Page{}, domain.ErrNotFound
	}
	return c.catalog.GetPage(ctx, slug)
}

// Search returns only report pages visible to the current user.
func (c accessiblePageCatalog) Search(ctx context.Context, query string, limit int) ([]domain.Page, error) {
	pages, err := c.catalog.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return c.access.FilterPages(ctx, c.user, pages)
}

// visibleRecentEdits filters recent edits to pages the user may view.
func visibleRecentEdits(ctx context.Context, access pageAccessReader, user domain.User, edits []domain.RecentEdit) ([]domain.RecentEdit, error) {
	result := make([]domain.RecentEdit, 0, len(edits))

	for _, edit := range edits {
		allowed, err := access.CanView(ctx, user, edit.Slug)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		result = append(result, edit)
	}

	return result, nil
}

// visibleKnowledgeGraph removes graph nodes and edges hidden from the user.
func visibleKnowledgeGraph(ctx context.Context, access pageAccessReader, user domain.User, graph domain.KnowledgeGraph) (domain.KnowledgeGraph, error) {
	visible := make(map[string]bool, len(graph.Nodes))
	nodes := make([]domain.GraphNode, 0, len(graph.Nodes))

	for _, node := range graph.Nodes {
		allowed, err := access.CanView(ctx, user, node.Slug)
		if err != nil {
			return domain.KnowledgeGraph{}, err
		}
		if !allowed {
			continue
		}

		visible[node.Slug] = true
		nodes = append(nodes, node)
	}

	edges := make([]domain.GraphEdge, 0, len(graph.Edges))

	for _, edge := range graph.Edges {
		if !visible[edge.Source] || !visible[edge.Target] {
			continue
		}

		edges = append(edges, edge)
	}

	graph.Nodes = nodes
	graph.Edges = edges

	return graph, nil
}
