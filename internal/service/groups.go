package service

import (
	"context"

	"github.com/gi8lino/lore/internal/model"
)

// groupRepository contains collaboration group operations.
type groupRepository interface {
	Groups(context.Context) ([]model.Group, error)
	AssignableGroups(context.Context, model.User) ([]model.Group, error)
	CreateGroup(context.Context, string) (model.Group, error)
	DeleteGroup(context.Context, int64) error
	GroupMembers(context.Context, int64) ([]model.User, error)
	AddGroupMember(context.Context, int64, int64) error
	RemoveGroupMember(context.Context, int64, int64) error
}

// Groups exposes collaboration group use cases.
type Groups struct{ repository groupRepository }

// NewGroups constructs the collaboration group service.
func NewGroups(repository groupRepository) *Groups { return &Groups{repository: repository} }

// Groups returns all collaboration groups.
func (s *Groups) Groups(ctx context.Context) ([]model.Group, error) {
	return s.repository.Groups(ctx)
}

// AssignableGroups returns the groups an actor may assign to pages.
func (s *Groups) AssignableGroups(ctx context.Context, user model.User) ([]model.Group, error) {
	return s.repository.AssignableGroups(ctx, user)
}

// CreateGroup creates a collaboration group.
func (s *Groups) CreateGroup(ctx context.Context, name string) (model.Group, error) {
	return s.repository.CreateGroup(ctx, name)
}

// DeleteGroup removes a collaboration group.
func (s *Groups) DeleteGroup(ctx context.Context, id int64) error {
	return s.repository.DeleteGroup(ctx, id)
}

// GroupMembers returns the users assigned to a group.
func (s *Groups) GroupMembers(ctx context.Context, groupID int64) ([]model.User, error) {
	return s.repository.GroupMembers(ctx, groupID)
}

// AddGroupMember assigns a user to a collaboration group.
func (s *Groups) AddGroupMember(ctx context.Context, groupID, userID int64) error {
	return s.repository.AddGroupMember(ctx, groupID, userID)
}

// RemoveGroupMember removes a user from a collaboration group.
func (s *Groups) RemoveGroupMember(ctx context.Context, groupID, userID int64) error {
	return s.repository.RemoveGroupMember(ctx, groupID, userID)
}
