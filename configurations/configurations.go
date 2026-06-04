// Package configurations covers Configuration management in the built-in
// (embedded) device management of Apple Business: list/get and CRUD for
// custom (.mobileconfig) configuration profiles.
package configurations

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Configuration is a Configuration resource.
type Configuration = applebusiness.ResourceObject[Attributes]

// Attributes holds the attributes of a Configuration.
// Type is the configuration kind (configurationType). It is distinct from the
// outer data.type value "configurations".
type Attributes struct {
	Type                   string                `json:"type,omitempty"` // AIR_DROP / AUTHENTICATION_SCREEN_LOCK / CUSTOM_SETTING ...
	Name                   string                `json:"name,omitempty"`
	ConfiguredForPlatforms []string              `json:"configuredForPlatforms,omitempty"`
	CustomSettingsValues   *CustomSettingsValues `json:"customSettingsValues,omitempty"`
	CreatedDateTime        string                `json:"createdDateTime,omitempty"`
	UpdatedDateTime        string                `json:"updatedDateTime,omitempty"`
}

// CustomSettingsValues holds the profile payload for a CUSTOM_SETTING configuration.
type CustomSettingsValues struct {
	ConfigurationProfile string `json:"configurationProfile,omitempty"` // byte=Base64 (the .mobileconfig encoded as a Base64 string)
	Filename             string `json:"filename,omitempty"`
}

const (
	// TypeCustomSetting is the kind for a custom configuration profile.
	// ConfigurationType has 23 values in total (AIR_DROP / AUTHENTICATION_SCREEN_LOCK / CERTIFICATE / FILE_VAULT /
	// SOFTWARE_UPDATE / VPN / WIFI / WEB_CLIP / ...). See docs/apple-business-api-datatypes.md §1 for details.
	TypeCustomSetting = "CUSTOM_SETTING"

	// All values of ConfigurationPlatform.
	PlatformMacOS    = "PLATFORM_MACOS"
	PlatformIOS      = "PLATFORM_IOS"
	PlatformTVOS     = "PLATFORM_TVOS"
	PlatformVisionOS = "PLATFORM_VISIONOS"
)

const resourceType = "configurations"

// Service provides the Configuration-related endpoints. Create it with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

func (s *Service) List(ctx context.Context, q url.Values) ([]Configuration, error) {
	return applebusiness.List[Attributes](ctx, s.c, "/v1/configurations", q)
}

func (s *Service) Get(ctx context.Context, id string) (*Configuration, error) {
	return applebusiness.Get[Attributes](ctx, s.c, "/v1/configurations/"+url.PathEscape(id))
}

// CreateInput holds the parameters for creating a Configuration.
// When creating a CUSTOM_SETTING, ConfigurationProfile and Filename are required.
type CreateInput struct {
	Type                   string // defaults to CUSTOM_SETTING when empty
	Name                   string
	ConfiguredForPlatforms []string
	ConfigurationProfile   string // .mobileconfig XML
	Filename               string
}

// Create creates a new Configuration (POST /v1/configurations).
func (s *Service) Create(ctx context.Context, in CreateInput) (*Configuration, error) {
	t := in.Type
	if t == "" {
		t = TypeCustomSetting
	}
	attrs := Attributes{
		Type:                   t,
		Name:                   in.Name,
		ConfiguredForPlatforms: in.ConfiguredForPlatforms,
	}
	if in.ConfigurationProfile != "" || in.Filename != "" {
		attrs.CustomSettingsValues = &CustomSettingsValues{
			ConfigurationProfile: in.ConfigurationProfile,
			Filename:             in.Filename,
		}
	}
	var body writeBody
	body.Data.Type = resourceType
	body.Data.Attributes = attrs
	return applebusiness.Create[Attributes](ctx, s.c, "/v1/configurations", body)
}

// UpdateInput specifies only the attributes to change (nil or empty leaves them unchanged).
type UpdateInput struct {
	Name                   *string
	ConfiguredForPlatforms []string
	ConfigurationProfile   *string
	Filename               *string
}

// Update updates a Configuration (PATCH /v1/configurations/{id}).
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (*Configuration, error) {
	var attrs updateAttrs
	attrs.Name = in.Name
	attrs.ConfiguredForPlatforms = in.ConfiguredForPlatforms
	if in.ConfigurationProfile != nil || in.Filename != nil {
		csv := &CustomSettingsValues{}
		if in.ConfigurationProfile != nil {
			csv.ConfigurationProfile = *in.ConfigurationProfile
		}
		if in.Filename != nil {
			csv.Filename = *in.Filename
		}
		attrs.CustomSettingsValues = csv
	}
	var body writeBody
	body.Data.Type = resourceType
	body.Data.ID = id
	body.Data.Attributes = attrs
	return applebusiness.Update[Attributes](ctx, s.c, "/v1/configurations/"+url.PathEscape(id), body)
}

// Delete deletes a Configuration (DELETE /v1/configurations/{id}).
func (s *Service) Delete(ctx context.Context, id string) error {
	return applebusiness.Delete(ctx, s.c, "/v1/configurations/"+url.PathEscape(id))
}

// --- リクエストボディ -------------------------------------------------------

type writeBody struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id,omitempty"`
		Attributes any    `json:"attributes"`
	} `json:"data"`
}

type updateAttrs struct {
	Name                   *string               `json:"name,omitempty"`
	ConfiguredForPlatforms []string              `json:"configuredForPlatforms,omitempty"`
	CustomSettingsValues   *CustomSettingsValues `json:"customSettingsValues,omitempty"`
}
