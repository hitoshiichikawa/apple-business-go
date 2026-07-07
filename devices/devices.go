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
// Note: mdmServers only allows GET_COLLECTION. Fetching a single server
// (GET /v1/mdmServers/{id}) returns 403, so it is not provided. Each server's
// attributes are included in the list elements (MdmServer.Attributes).
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
