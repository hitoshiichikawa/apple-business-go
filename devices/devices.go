// Package devices covers the device-management pillar of the Apple Business API:
// organization devices, MDM servers (device management services), AppleCare
// coverage, and device-assignment activities.
package devices

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Service exposes device-management endpoints. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// List returns all organization devices (pagination handled automatically).
func (s *Service) List(ctx context.Context, q url.Values) ([]Device, error) {
	return applebusiness.List[DeviceAttributes](ctx, s.c, "/v1/orgDevices", q)
}

// Get returns a single device by ID.
func (s *Service) Get(ctx context.Context, id string) (*Device, error) {
	return applebusiness.Get[DeviceAttributes](ctx, s.c, "/v1/orgDevices/"+url.PathEscape(id))
}

// AssignedServer returns the MDM server a device is assigned to (status=ASSIGNED).
func (s *Service) AssignedServer(ctx context.Context, deviceID string) (*MdmServer, error) {
	return applebusiness.Get[MdmServerAttributes](ctx, s.c, "/v1/orgDevices/"+url.PathEscape(deviceID)+"/assignedServer")
}

// AppleCareCoverage returns AppleCare coverage for a device.
func (s *Service) AppleCareCoverage(ctx context.Context, deviceID string) (*Coverage, error) {
	return applebusiness.Get[AppleCareCoverageAttributes](ctx, s.c, "/v1/orgDevices/"+url.PathEscape(deviceID)+"/appleCareCoverage")
}

// ListMdmServers returns all MDM servers (device management services).
// 注意: mdmServers は GET_COLLECTION のみ許可。単体取得 (GET /v1/mdmServers/{id}) は 403 になるため提供しない。
// 各サーバの属性は一覧の各要素 (MdmServer.Attributes) に含まれる。
func (s *Service) ListMdmServers(ctx context.Context, q url.Values) ([]MdmServer, error) {
	return applebusiness.List[MdmServerAttributes](ctx, s.c, "/v1/mdmServers", q)
}

// MdmServerDevices returns the linkage (assigned device IDs, type "orgDevices") for an MDM server (all pages).
// これがメンバーシップ取得の唯一の手段（related の GET_RELATED は不可）。フル属性は MdmServerDeviceList を使う。
func (s *Service) MdmServerDevices(ctx context.Context, serverID string) ([]applebusiness.Data, error) {
	return applebusiness.Relationship(ctx, s.c, "/v1/mdmServers/"+url.PathEscape(serverID)+"/relationships/devices")
}

// MdmServerDeviceList returns the full organization-device objects assigned to an MDM server.
// 注意: related エンドポイント /v1/mdmServers/{id}/devices は GET_RELATED 不可（403, allowed: GET_RELATIONSHIP）。
// そのため relationships で得たID（type=orgDevices）ごとに /v1/orgDevices/{id} を個別取得する（N+1 リクエスト）。
func (s *Service) MdmServerDeviceList(ctx context.Context, serverID string) ([]Device, error) {
	ids, err := s.MdmServerDevices(ctx, serverID)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(ids))
	for _, d := range ids {
		dev, err := s.Get(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *dev)
	}
	return out, nil
}
