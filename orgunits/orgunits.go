// Package orgunits covers organizational units of the Apple Business API
// (added in API 2.2). Organizational units are the org-structure/permission
// layer that users and user groups reference; they are read-only.
package orgunits

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Service exposes the organizational-unit endpoints. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// List returns all organizational units (pagination handled automatically).
func (s *Service) List(ctx context.Context, q url.Values) ([]OrganizationalUnit, error) {
	return applebusiness.List[OrganizationalUnitAttributes](ctx, s.c, "/v1/organizationalUnits", q)
}

// Get returns a single organizational unit by ID.
func (s *Service) Get(ctx context.Context, id string) (*OrganizationalUnit, error) {
	return applebusiness.Get[OrganizationalUnitAttributes](ctx, s.c, "/v1/organizationalUnits/"+url.PathEscape(id))
}

// Members returns the user IDs (type "users") that belong to an organizational
// unit (all pages). Only the users relationship is exposed for organizational
// units; resolve full user objects via people.GetUser.
func (s *Service) Members(ctx context.Context, ouID string) ([]applebusiness.Data, error) {
	return applebusiness.Relationship(ctx, s.c, "/v1/organizationalUnits/"+url.PathEscape(ouID)+"/relationships/users")
}
