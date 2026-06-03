// Package blueprints covers Blueprint management in the built-in (embedded)
// device management of Apple Business: CRUD plus relationship assignment of
// apps, configurations, packages, devices, users, and user groups.
//
// Assigning devices to a Blueprint is done by adding them to the orgDevices
// relationship (there is no separate "apply" activity).
package blueprints

import (
	"context"
	"net/http"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Blueprint はBlueprintリソース。
type Blueprint = applebusiness.ResourceObject[Attributes]

// Attributes はBlueprintの属性。
type Attributes struct {
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	Status              string `json:"status,omitempty"`
	AppLicenseDeficient bool   `json:"appLicenseDeficient,omitempty"`
	CreatedDateTime     string `json:"createdDateTime,omitempty"`
	UpdatedDateTime     string `json:"updatedDateTime,omitempty"`
}

// リレーション名（= 各要素の type と同じ）。
const (
	RelApps           = "apps"
	RelConfigurations = "configurations"
	RelPackages       = "packages"
	RelOrgDevices     = "orgDevices"
	RelUsers          = "users"
	RelUserGroups     = "userGroups"
)

// BlueprintStatus の値。
const (
	StatusActive      = "ACTIVE"
	StatusToBeDeleted = "TO_BE_DELETED"
)

const resourceType = "blueprints"

// Service はBlueprint関連エンドポイントを提供する。New で生成。
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// --- 読み取り ---------------------------------------------------------------

func (s *Service) List(ctx context.Context, q url.Values) ([]Blueprint, error) {
	return applebusiness.List[Attributes](ctx, s.c, "/v1/blueprints", q)
}

func (s *Service) Get(ctx context.Context, id string) (*Blueprint, error) {
	return applebusiness.Get[Attributes](ctx, s.c, "/v1/blueprints/"+url.PathEscape(id))
}

// RelationshipIDs は指定リレーションの関連リソースID一覧を返す（全ページ）。
// rel は Rel* 定数を使う。注: 権限/状態により 403 になり得る。
func (s *Service) RelationshipIDs(ctx context.Context, id, rel string) ([]applebusiness.Data, error) {
	return applebusiness.Relationship(ctx, s.c, "/v1/blueprints/"+url.PathEscape(id)+"/relationships/"+rel)
}

// --- 書き込み ---------------------------------------------------------------

// CreateInput はBlueprint作成パラメータ。各リレーションIDは任意（後から付け外し可）。
type CreateInput struct {
	Name           string
	Description    string
	Apps           []string
	Configurations []string
	Packages       []string
	OrgDevices     []string
	Users          []string
	UserGroups     []string
}

// Create は新しいBlueprintを作成する（POST /v1/blueprints）。
func (s *Service) Create(ctx context.Context, in CreateInput) (*Blueprint, error) {
	var body writeBody
	body.Data.Type = resourceType
	body.Data.Attributes = createAttrs{Name: in.Name, Description: in.Description}
	body.Data.Relationships = buildRelationships(in)
	return applebusiness.Create[Attributes](ctx, s.c, "/v1/blueprints", body)
}

// UpdateInput は変更する属性のみを指定する（nil は据え置き）。
type UpdateInput struct {
	Name        *string
	Description *string
}

// Update はBlueprintの属性を更新する（PATCH /v1/blueprints/{id}）。
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (*Blueprint, error) {
	var body writeBody
	body.Data.Type = resourceType
	body.Data.ID = id
	body.Data.Attributes = updateAttrs{Name: in.Name, Description: in.Description}
	return applebusiness.Update[Attributes](ctx, s.c, "/v1/blueprints/"+url.PathEscape(id), body)
}

// Delete はBlueprintを削除する（DELETE /v1/blueprints/{id}）。
func (s *Service) Delete(ctx context.Context, id string) error {
	return applebusiness.Delete(ctx, s.c, "/v1/blueprints/"+url.PathEscape(id))
}

// AddTo はリレーションに関連リソースを追加する（POST）。rel は Rel* 定数。
func (s *Service) AddTo(ctx context.Context, id, rel string, ids []string) error {
	return s.modifyRel(ctx, http.MethodPost, id, rel, ids)
}

// RemoveFrom はリレーションから関連リソースを削除する（DELETE）。
func (s *Service) RemoveFrom(ctx context.Context, id, rel string, ids []string) error {
	return s.modifyRel(ctx, http.MethodDelete, id, rel, ids)
}

// Replace はリレーションの集合を置換する（PATCH）。
func (s *Service) Replace(ctx context.Context, id, rel string, ids []string) error {
	return s.modifyRel(ctx, http.MethodPatch, id, rel, ids)
}

func (s *Service) modifyRel(ctx context.Context, method, id, rel string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	items := make([]applebusiness.Data, len(ids))
	for i, x := range ids {
		items[i] = applebusiness.Data{Type: rel, ID: x} // rel名 == type
	}
	return applebusiness.ModifyRelationship(ctx, s.c, method,
		"/v1/blueprints/"+url.PathEscape(id)+"/relationships/"+rel, items)
}

// --- リクエストボディ -------------------------------------------------------

type writeBody struct {
	Data struct {
		Type          string         `json:"type"`
		ID            string         `json:"id,omitempty"`
		Attributes    any            `json:"attributes"`
		Relationships *relationships `json:"relationships,omitempty"`
	} `json:"data"`
}

type createAttrs struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type updateAttrs struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type relationships struct {
	Apps           *relData `json:"apps,omitempty"`
	Configurations *relData `json:"configurations,omitempty"`
	Packages       *relData `json:"packages,omitempty"`
	OrgDevices     *relData `json:"orgDevices,omitempty"`
	Users          *relData `json:"users,omitempty"`
	UserGroups     *relData `json:"userGroups,omitempty"`
}

type relData struct {
	Data []applebusiness.Data `json:"data"`
}

func toRel(typ string, ids []string) *relData {
	if len(ids) == 0 {
		return nil
	}
	d := make([]applebusiness.Data, len(ids))
	for i, x := range ids {
		d[i] = applebusiness.Data{Type: typ, ID: x}
	}
	return &relData{Data: d}
}

func buildRelationships(in CreateInput) *relationships {
	r := &relationships{
		Apps:           toRel(RelApps, in.Apps),
		Configurations: toRel(RelConfigurations, in.Configurations),
		Packages:       toRel(RelPackages, in.Packages),
		OrgDevices:     toRel(RelOrgDevices, in.OrgDevices),
		Users:          toRel(RelUsers, in.Users),
		UserGroups:     toRel(RelUserGroups, in.UserGroups),
	}
	if r.Apps == nil && r.Configurations == nil && r.Packages == nil &&
		r.OrgDevices == nil && r.Users == nil && r.UserGroups == nil {
		return nil
	}
	return r
}
