package service

import (
	"context"
	"fmt"
)

// knowledgeRepository contains knowledge tools, saved searches, and notification operations.
type knowledgeRepository interface {
	auditRepository
	KnowledgeGraph(context.Context, int) (KnowledgeGraph, error)
	KnowledgeSnippets(context.Context) ([]KnowledgeSnippet, error)
	KnowledgeSnippetByName(context.Context, string, string) (KnowledgeSnippet, error)
	SaveKnowledgeSnippet(context.Context, int64, int64, string, string, string, string) (KnowledgeSnippet, error)
	DeleteKnowledgeSnippet(context.Context, int64) error
	SavedSearches(context.Context, int64) ([]SavedSearch, error)
	SaveSavedSearch(context.Context, int64, int64, string, string, bool) error
	DeleteSavedSearch(context.Context, int64, int64) error
	Notifications(context.Context, int64, int) ([]Notification, int, error)
	MarkNotificationRead(context.Context, int64, int64) error
}

// Knowledge exposes snippets, saved searches, graph, and notification use cases.
type Knowledge struct{ repository knowledgeRepository }

// NewKnowledge constructs the knowledge tools service.
func NewKnowledge(repository knowledgeRepository) *Knowledge {
	return &Knowledge{repository: repository}
}

// KnowledgeGraph returns page nodes and links for graph rendering.
func (s *Knowledge) KnowledgeGraph(ctx context.Context, limit int) (KnowledgeGraph, error) {
	return s.repository.KnowledgeGraph(ctx, limit)
}

// KnowledgeSnippets returns reusable knowledge snippets.
func (s *Knowledge) KnowledgeSnippets(ctx context.Context) ([]KnowledgeSnippet, error) {
	return s.repository.KnowledgeSnippets(ctx)
}

// KnowledgeSnippetByName returns a reusable snippet by kind and name.
func (s *Knowledge) KnowledgeSnippetByName(
	ctx context.Context,
	kind, name string,
) (KnowledgeSnippet, error) {
	return s.repository.KnowledgeSnippetByName(ctx, kind, name)
}

// SaveKnowledgeSnippet persists a snippet and records an audit event.
func (s *Knowledge) SaveKnowledgeSnippet(
	ctx context.Context,
	id, userID int64,
	kind, name, description, content string,
) (KnowledgeSnippet, error) {
	item, err := s.repository.SaveKnowledgeSnippet(ctx, id, userID, kind, name, description, content)
	if err != nil {
		return KnowledgeSnippet{}, err
	}
	_ = audit(s.repository, ctx, userID, "snippet.saved", "snippet", item.Name, item.Kind)
	return item, nil
}

// DeleteKnowledgeSnippet removes a snippet and records an audit event.
func (s *Knowledge) DeleteKnowledgeSnippet(ctx context.Context, id, actorID int64) error {
	if err := s.repository.DeleteKnowledgeSnippet(ctx, id); err != nil {
		return err
	}
	_ = audit(s.repository, ctx, actorID, "snippet.deleted", "snippet", fmt.Sprint(id), "")
	return nil
}

// SavedSearches returns the searches saved by a user.
func (s *Knowledge) SavedSearches(ctx context.Context, userID int64) ([]SavedSearch, error) {
	return s.repository.SavedSearches(ctx, userID)
}

// SaveSavedSearch creates or updates a user's saved search.
func (s *Knowledge) SaveSavedSearch(
	ctx context.Context,
	userID, id int64,
	name, query string,
	pinned bool,
) error {
	return s.repository.SaveSavedSearch(ctx, userID, id, name, query, pinned)
}

// DeleteSavedSearch removes a saved search owned by a user.
func (s *Knowledge) DeleteSavedSearch(ctx context.Context, userID, id int64) error {
	return s.repository.DeleteSavedSearch(ctx, userID, id)
}

// Notifications returns a user's recent notifications and unread count.
func (s *Knowledge) Notifications(
	ctx context.Context,
	userID int64,
	limit int,
) ([]Notification, int, error) {
	return s.repository.Notifications(ctx, userID, limit)
}

// MarkNotificationRead marks an owned notification as read.
func (s *Knowledge) MarkNotificationRead(ctx context.Context, userID, id int64) error {
	return s.repository.MarkNotificationRead(ctx, userID, id)
}
