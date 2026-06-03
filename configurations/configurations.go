// Package configurations covers Configuration management in the built-in
// (embedded) device management of Apple Business: list/get and CRUD for
// custom (.mobileconfig) configuration profiles.
package configurations

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Configuration はConfigurationリソース。
type Configuration = applebusiness.ResourceObject[Attributes]

// Attributes はConfigurationの属性。
// Type は構成種別（= configurationType）。外側 data.type の "configurations" とは別物。
type Attributes struct {
	Type                   string                `json:"type,omitempty"` // AIR_DROP / AUTHENTICATION_SCREEN_LOCK / CUSTOM_SETTING ...
	Name                   string                `json:"name,omitempty"`
	ConfiguredForPlatforms []string              `json:"configuredForPlatforms,omitempty"`
	CustomSettingsValues   *CustomSettingsValues `json:"customSettingsValues,omitempty"`
	CreatedDateTime        string                `json:"createdDateTime,omitempty"`
	UpdatedDateTime        string                `json:"updatedDateTime,omitempty"`
}

// CustomSettingsValues は CUSTOM_SETTING のときのプロファイル本体。
type CustomSettingsValues struct {
	ConfigurationProfile string `json:"configurationProfile,omitempty"` // byte=Base64（.mobileconfig をBase64エンコードした文字列）
	Filename             string `json:"filename,omitempty"`
}

const (
	// TypeCustomSetting はカスタム構成プロファイルの種別。
	// ConfigurationType は全23種（AIR_DROP / AUTHENTICATION_SCREEN_LOCK / CERTIFICATE / FILE_VAULT /
	// SOFTWARE_UPDATE / VPN / WIFI / WEB_CLIP / ... ）。詳細は docs/apple-business-api-datatypes.md §1。
	TypeCustomSetting = "CUSTOM_SETTING"

	// ConfigurationPlatform の全値。
	PlatformMacOS   = "PLATFORM_MACOS"
	PlatformIOS     = "PLATFORM_IOS"
	PlatformTVOS    = "PLATFORM_TVOS"
	PlatformVisionOS = "PLATFORM_VISIONOS"
)

const resourceType = "configurations"

// Service はConfiguration関連エンドポイントを提供する。New で生成。
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

// CreateInput はConfiguration作成パラメータ。
// CUSTOM_SETTING を作る場合は ConfigurationProfile と Filename が必須。
type CreateInput struct {
	Type                   string // 空なら CUSTOM_SETTING
	Name                   string
	ConfiguredForPlatforms []string
	ConfigurationProfile   string // .mobileconfig XML
	Filename               string
}

// Create は新しいConfigurationを作成する（POST /v1/configurations）。
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

// UpdateInput は変更する属性のみを指定する（nil/空は据え置き）。
type UpdateInput struct {
	Name                   *string
	ConfiguredForPlatforms []string
	ConfigurationProfile   *string
	Filename               *string
}

// Update はConfigurationを更新する（PATCH /v1/configurations/{id}）。
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

// Delete はConfigurationを削除する（DELETE /v1/configurations/{id}）。
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
