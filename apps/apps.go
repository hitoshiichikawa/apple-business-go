// Package apps covers the "Apps and Packages" category (read-only):
// /v1/apps and /v1/packages.
package apps

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

type (
	App     = applebusiness.ResourceObject[AppAttributes]
	Package = applebusiness.ResourceObject[PackageAttributes]
)

// SupportedOS の値（App.SupportedOS の要素）。
const (
	OSUnspecified = "SUPPORTED_OS_UNSPECIFIED"
	OSIPadOS      = "SUPPORTED_OS_IPADOS"
	OSIOS         = "SUPPORTED_OS_IOS"
	OSMacOS       = "SUPPORTED_OS_MACOS"
	OSTVOS        = "SUPPORTED_OS_TVOS"
	OSWatchOS     = "SUPPORTED_OS_WATCHOS"
	OSVisionOS    = "SUPPORTED_OS_VISIONOS"
)

// AppAttributes : /v1/apps
type AppAttributes struct {
	Name        string `json:"name,omitempty"`
	BundleID    string `json:"bundleId,omitempty"`
	Version     string `json:"version,omitempty"`
	SupportedOS []string `json:"supportedOS,omitempty"` // 公式: [SupportedOS]（配列）
	AppStoreURL string   `json:"appStoreUrl,omitempty"`
	WebsiteURL  string   `json:"websiteUrl,omitempty"`
	IsCustomApp bool     `json:"isCustomApp,omitempty"`
}

// PackageAttributes : /v1/packages
type PackageAttributes struct {
	Name            string   `json:"name,omitempty"`
	Version         string   `json:"version,omitempty"`
	Description     string   `json:"description,omitempty"`
	BundleIDs       []string `json:"bundleIds,omitempty"`
	Hash            string   `json:"hash,omitempty"`
	URL             string   `json:"url,omitempty"`
	CreatedDateTime string   `json:"createdDateTime,omitempty"`
	UpdatedDateTime string   `json:"updatedDateTime,omitempty"`
}

// Service exposes apps/packages read endpoints. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

func (s *Service) ListApps(ctx context.Context, q url.Values) ([]App, error) {
	return applebusiness.List[AppAttributes](ctx, s.c, "/v1/apps", q)
}

func (s *Service) GetApp(ctx context.Context, id string) (*App, error) {
	return applebusiness.Get[AppAttributes](ctx, s.c, "/v1/apps/"+url.PathEscape(id))
}

func (s *Service) ListPackages(ctx context.Context, q url.Values) ([]Package, error) {
	return applebusiness.List[PackageAttributes](ctx, s.c, "/v1/packages", q)
}

func (s *Service) GetPackage(ctx context.Context, id string) (*Package, error) {
	return applebusiness.Get[PackageAttributes](ctx, s.c, "/v1/packages/"+url.PathEscape(id))
}
