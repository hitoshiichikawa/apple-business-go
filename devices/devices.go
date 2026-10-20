// Package devices covers the device-management pillar of the Apple Business API:
// organization devices, MDM servers (device management services), AppleCare
// coverage, Activation Lock status, and device-assignment activities.
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

// AppleCareCoverages returns all AppleCare coverages for a device
// (pagination handled automatically).
//
// 実 Apple Business の GET /v1/orgDevices/{id}/appleCareCoverage は data が
// 配列の一覧型で返る（links.self と meta.paging.limit 100 付き）。1 台に複数の
// 契約が付くことがあり、契約が無ければ空配列になる。要素の ID は契約番号か
// シリアル番号で、agreementNumber / contractCancelDateTime は null（＝ゼロ値の
// 空文字列）になりうる。
func (s *Service) AppleCareCoverages(ctx context.Context, deviceID string) ([]Coverage, error) {
	return applebusiness.List[AppleCareCoverageAttributes](ctx, s.c, "/v1/orgDevices/"+url.PathEscape(deviceID)+"/appleCareCoverage", nil)
}

// AppleCareCoverage returns the first AppleCare coverage for a device, or
// (nil, nil) when the device has none. It is kept for compatibility.
//
// 応答は data が配列で、全件は AppleCareCoverages で取得する。複数ある端末で
// どれを使うかは呼び出し側が決めること（例: endDateTime が最も遅いもの）。
// この関数は API が返した順の 1 件目をそのまま返すだけで、並べ替えや選別はしない。
func (s *Service) AppleCareCoverage(ctx context.Context, deviceID string) (*Coverage, error) {
	all, err := s.AppleCareCoverages(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	return &all[0], nil
}

// ActivationLockStatus returns the Activation Lock status of a device
// (GET /v1/orgDevices/{id}/activationLockStatus, Apple Business API 2.6;
// added 2026-09-16). deviceID is the device's serial number; a 404
// (applebusiness.IsNotFound) means the organization does not manage a device
// with that serial number.
//
// Apple notes that the request can also fail with an unexpected server error
// if the device reports an internal-only lock state that isn't valid for a
// managed device.
func (s *Service) ActivationLockStatus(ctx context.Context, deviceID string) (*ActivationLockStatus, error) {
	return applebusiness.Get[ActivationLockStatusAttributes](ctx, s.c, "/v1/orgDevices/"+url.PathEscape(deviceID)+"/activationLockStatus")
}

// ListMdmServers returns all MDM servers (device management services).
// Each server's attributes are included in the list elements
// (MdmServer.Attributes). Since Apple Business API 2.1 a single server can
// also be fetched with GetMdmServer, and CRUD is available via
// CreateMdmServer / UpdateMdmServer / DeleteMdmServer (see mdmservers.go).
func (s *Service) ListMdmServers(ctx context.Context, q url.Values) ([]MdmServer, error) {
	return applebusiness.List[MdmServerAttributes](ctx, s.c, "/v1/mdmServers", q)
}

// MdmServerDevices returns the linkage (assigned device IDs, type "orgDevices") for an MDM server (all pages).
// This is the only way to obtain membership (GET_RELATED on the related link is
// not allowed). Use MdmServerDeviceList for the full attributes.
func (s *Service) MdmServerDevices(ctx context.Context, serverID string) ([]applebusiness.Data, error) {
	return applebusiness.Relationship(ctx, s.c, "/v1/mdmServers/"+url.PathEscape(serverID)+"/relationships/devices")
}

// MdmServerDeviceList returns the full organization-device objects assigned to an MDM server.
// Note: the related endpoint /v1/mdmServers/{id}/devices does not allow GET_RELATED
// (403, allowed: GET_RELATIONSHIP). It therefore fetches /v1/orgDevices/{id}
// individually for each ID (type=orgDevices) obtained from the relationship (N+1 requests).
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

// ListMdmDevices returns all devices enrolled in Apple's built-in device
// management service (GET /v1/mdmDevices, pagination handled automatically).
func (s *Service) ListMdmDevices(ctx context.Context, q url.Values) ([]MdmDevice, error) {
	return applebusiness.List[MdmDeviceAttributes](ctx, s.c, "/v1/mdmDevices", q)
}

// MdmDeviceDetails returns the details for a device enrolled in Apple's
// built-in device management service (GET /v1/mdmDevices/{id}/details).
func (s *Service) MdmDeviceDetails(ctx context.Context, id string) (*MdmDeviceDetail, error) {
	return applebusiness.Get[MdmDeviceDetailAttributes](ctx, s.c, "/v1/mdmDevices/"+url.PathEscape(id)+"/details")
}
