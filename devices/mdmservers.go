// MDM server (device management service) CRUD, added in Apple Business API 2.1.
// Read helpers that predate 2.1 (ListMdmServers / MdmServerDevices /
// MdmServerDeviceList) live in devices.go.

package devices

import (
	"context"
	"net/url"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

const mdmServerType = "mdmServers"

// GetMdmServer returns a single MDM server (device management service) by ID
// (GET /v1/mdmServers/{id}, Apple Business API 2.1+).
func (s *Service) GetMdmServer(ctx context.Context, id string) (*MdmServer, error) {
	return applebusiness.Get[MdmServerAttributes](ctx, s.c, "/v1/mdmServers/"+url.PathEscape(id))
}

// CreateMdmServerInput holds the parameters for creating an MDM server.
// ServerName and ServerCertificate are required by the API;
// EnableMdmDisownFlag is optional (nil omits the field).
type CreateMdmServerInput struct {
	ServerName          string
	ServerCertificate   MdmServerCertificate
	EnableMdmDisownFlag *bool
}

// CreateMdmServer creates a new MDM server (POST /v1/mdmServers,
// Apple Business API 2.1+).
func (s *Service) CreateMdmServer(ctx context.Context, in CreateMdmServerInput) (*MdmServer, error) {
	var body mdmServerWriteBody
	body.Data.Type = mdmServerType
	body.Data.Attributes = mdmServerCreateAttrs{
		ServerName:          in.ServerName,
		ServerCertificate:   in.ServerCertificate,
		EnableMdmDisownFlag: in.EnableMdmDisownFlag,
	}
	return applebusiness.Create[MdmServerAttributes](ctx, s.c, "/v1/mdmServers", body)
}

// UpdateMdmServerInput specifies only the attributes to change (nil / empty
// leaves them unchanged). DefaultProductFamilies takes the
// MdmProductFamily* constants.
type UpdateMdmServerInput struct {
	ServerName             *string
	ServerCertificate      *MdmServerCertificate
	EnableMdmDisownFlag    *bool
	DefaultProductFamilies []string
}

// UpdateMdmServer updates the attributes of an MDM server
// (PATCH /v1/mdmServers/{id}, Apple Business API 2.1+).
func (s *Service) UpdateMdmServer(ctx context.Context, id string, in UpdateMdmServerInput) (*MdmServer, error) {
	var body mdmServerWriteBody
	body.Data.Type = mdmServerType
	body.Data.ID = id
	body.Data.Attributes = mdmServerUpdateAttrs(in)
	return applebusiness.Update[MdmServerAttributes](ctx, s.c, "/v1/mdmServers/"+url.PathEscape(id), body)
}

// DeleteMdmServer deletes an MDM server (DELETE /v1/mdmServers/{id},
// Apple Business API 2.1+). A server that still has devices assigned cannot
// be deleted; unassign all devices first.
func (s *Service) DeleteMdmServer(ctx context.Context, id string) error {
	return applebusiness.Delete(ctx, s.c, "/v1/mdmServers/"+url.PathEscape(id))
}

// --- リクエストボディ -------------------------------------------------------

type mdmServerWriteBody struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id,omitempty"`
		Attributes any    `json:"attributes"`
	} `json:"data"`
}

type mdmServerCreateAttrs struct {
	ServerName          string               `json:"serverName"`
	ServerCertificate   MdmServerCertificate `json:"serverCertificate"`
	EnableMdmDisownFlag *bool                `json:"enableMdmDisownFlag,omitempty"`
}

type mdmServerUpdateAttrs struct {
	ServerName             *string               `json:"serverName,omitempty"`
	ServerCertificate      *MdmServerCertificate `json:"serverCertificate,omitempty"`
	EnableMdmDisownFlag    *bool                 `json:"enableMdmDisownFlag,omitempty"`
	DefaultProductFamilies []string              `json:"defaultProductFamilies,omitempty"`
}
