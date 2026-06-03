package devices

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

func testKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

// newClient は擬似トークン端点と擬似APIサーバを立て、それらを指す Client を返す。
func newClient(t *testing.T, h http.Handler) *applebusiness.Client {
	t.Helper()
	api := httptest.NewServer(h)
	t.Cleanup(api.Close)
	tok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(tok.Close)
	c, err := applebusiness.NewClient(applebusiness.Config{Credentials: applebusiness.Credentials{
		ClientID:   "BUSINESSAPI.test",
		TeamID:     "BUSINESSAPI.test",
		KeyID:      "kid",
		PrivateKey: testKeyPEM(t),
	}}, applebusiness.WithBaseURL(api.URL), applebusiness.WithTokenURL(tok.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

func devicesHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/orgDevices" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[DeviceAttributes]{
				Data: []applebusiness.ResourceObject[DeviceAttributes]{
					{Type: "orgDevices", ID: "D1", Attributes: DeviceAttributes{SerialNumber: "D1", DeviceModel: "iPhone15", Status: "ASSIGNED"}},
					{Type: "orgDevices", ID: "D2", Attributes: DeviceAttributes{SerialNumber: "D2", DeviceModel: "iPadPro", Status: "UNASSIGNED"}},
				},
			})
		case strings.HasSuffix(p, "/assignedServer"):
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[MdmServerAttributes]{
				Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{ServerName: "Server-1", ServerType: "MDM"}},
			})
		case strings.HasSuffix(p, "/relationships/devices"):
			writeJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "orgDevices", ID: "D1"}, {Type: "orgDevices", ID: "D2"}},
			})
		case strings.HasPrefix(p, "/v1/orgDevices/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/orgDevices/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[DeviceAttributes]{
				Data: applebusiness.ResourceObject[DeviceAttributes]{Type: "orgDevices", ID: id, Attributes: DeviceAttributes{SerialNumber: id, DeviceModel: "iPhone15", Status: "ASSIGNED"}},
			})
		case p == "/v1/mdmServers" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[MdmServerAttributes]{
				Data: []applebusiness.ResourceObject[MdmServerAttributes]{
					{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{ServerName: "Server-1", ServerType: "MDM"}},
				},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestDeviceList(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	got, err := New(c).List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "D1" || got[0].Attributes.SerialNumber != "D1" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got[1].Attributes.Status != "UNASSIGNED" {
		t.Fatalf("status: %+v", got[1].Attributes)
	}
}

func TestDeviceGet(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	got, err := New(c).Get(context.Background(), "KWG32FK99J")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "KWG32FK99J" || got.Attributes.SerialNumber != "KWG32FK99J" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestAssignedServer(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	srv, err := New(c).AssignedServer(context.Background(), "D1")
	if err != nil {
		t.Fatal(err)
	}
	if srv.ID != "S1" || srv.Attributes.ServerName != "Server-1" {
		t.Fatalf("unexpected: %+v", srv)
	}
}

func TestListMdmServers(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	got, err := New(c).ListMdmServers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "S1" || got[0].Attributes.ServerName != "Server-1" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestMdmServerDevices(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	ids, err := New(c).MdmServerDevices(context.Background(), "S1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Type != "orgDevices" || ids[1].ID != "D2" {
		t.Fatalf("unexpected: %+v", ids)
	}
}

func TestMdmServerDeviceList(t *testing.T) {
	c := newClient(t, devicesHandler(t))
	devs, err := New(c).MdmServerDeviceList(context.Background(), "S1")
	if err != nil {
		t.Fatal(err)
	}
	if len(devs) != 2 || devs[0].ID != "D1" || devs[1].ID != "D2" {
		t.Fatalf("unexpected: %+v", devs)
	}
	if devs[0].Attributes.SerialNumber != "D1" {
		t.Fatalf("serial: %+v", devs[0].Attributes)
	}
}
