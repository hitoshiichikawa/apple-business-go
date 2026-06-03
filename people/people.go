// Package people covers the people-management pillar of the Apple Business API:
// users and user groups.
package people

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Service exposes people-management endpoints. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// ListUsers returns all users (pagination handled automatically).
func (s *Service) ListUsers(ctx context.Context, q url.Values) ([]User, error) {
	return applebusiness.List[UserAttributes](ctx, s.c, "/v1/users", q)
}

// GetUser returns a single user by ID.
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
	return applebusiness.Get[UserAttributes](ctx, s.c, "/v1/users/"+url.PathEscape(id))
}

// ListUserGroups returns all user groups.
func (s *Service) ListUserGroups(ctx context.Context, q url.Values) ([]UserGroup, error) {
	return applebusiness.List[UserGroupAttributes](ctx, s.c, "/v1/userGroups", q)
}

// GetUserGroup returns a single user group by ID.
func (s *Service) GetUserGroup(ctx context.Context, id string) (*UserGroup, error) {
	return applebusiness.Get[UserGroupAttributes](ctx, s.c, "/v1/userGroups/"+url.PathEscape(id))
}

// GroupMembers returns the user IDs that belong to a group (all pages).
func (s *Service) GroupMembers(ctx context.Context, groupID string) ([]applebusiness.Data, error) {
	return applebusiness.Relationship(ctx, s.c, "/v1/userGroups/"+url.PathEscape(groupID)+"/relationships/users")
}
