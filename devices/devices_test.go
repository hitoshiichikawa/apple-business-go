package devices

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func devicesHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/orgDevices" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[DeviceAttributes]{
				Data: []applebusiness.ResourceObject[DeviceAttributes]{
					{Type: "orgDevices", ID: "D1", Attributes: DeviceAttributes{SerialNumber: "D1", DeviceModel: "iPhone15", Status: "ASSIGNED"}},
					{Type: "orgDevices", ID: "D2", Attributes: DeviceAttributes{SerialNumber: "D2", DeviceModel: "iPadPro", Status: "UNASSIGNED"}},
				},
			})
		case strings.HasSuffix(p, "/assignedServer"):
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[MdmServerAttributes]{
				Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{ServerName: "Server-1", ServerType: "MDM"}},
			})
		case strings.HasSuffix(p, "/relationships/devices"):
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "orgDevices", ID: "D1"}, {Type: "orgDevices", ID: "D2"}},
			})
		case strings.HasPrefix(p, "/v1/orgDevices/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/orgDevices/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[DeviceAttributes]{
				Data: applebusiness.ResourceObject[DeviceAttributes]{Type: "orgDevices", ID: id, Attributes: DeviceAttributes{SerialNumber: id, DeviceModel: "iPhone15", Status: "ASSIGNED"}},
			})
		case p == "/v1/mdmServers" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[MdmServerAttributes]{
				Data: []applebusiness.ResourceObject[MdmServerAttributes]{
					{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{ServerName: "Server-1", ServerType: "MDM"}},
				},
			})
		case p == "/v1/mdmDevices" && r.Method == http.MethodGet:
			if r.URL.Query().Get("cursor") == "p2" {
				testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[MdmDeviceAttributes]{
					Data: []applebusiness.ResourceObject[MdmDeviceAttributes]{
						{Type: "mdmDevices", ID: "M2", Attributes: MdmDeviceAttributes{SerialNumber: "M2", DeviceName: "iPad-2", ProductFamily: "iPad"}},
					},
				})
				return
			}
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[MdmDeviceAttributes]{
				Data: []applebusiness.ResourceObject[MdmDeviceAttributes]{
					{Type: "mdmDevices", ID: "M1", Attributes: MdmDeviceAttributes{SerialNumber: "M1", DeviceName: "iPhone-1", ProductFamily: "iPhone", EnrolledUserID: "U1"}},
				},
				Links: applebusiness.Links{Next: "http://" + r.Host + "/v1/mdmDevices?cursor=p2"},
			})
		case strings.HasPrefix(p, "/v1/mdmDevices/") && strings.HasSuffix(p, "/details") && r.Method == http.MethodGet:
			id := strings.TrimSuffix(strings.TrimPrefix(p, "/v1/mdmDevices/"), "/details")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[MdmDeviceDetailAttributes]{
				Data: applebusiness.ResourceObject[MdmDeviceDetailAttributes]{Type: "mdmDeviceDetails", ID: id, Attributes: MdmDeviceDetailAttributes{
					DeviceName: "iPhone-1", DeviceModel: "iPhone 15 Pro", SerialNumber: "SER1",
					OSVersion: "18.5", Platform: "iOS",
					StorageTotalCapacity: 256, StorageFreeCapacity: 128,
					DeviceEraseStatus: EraseStatusNotErased, DeviceLockStatus: LockStatusUnlocked, LostModeStatus: LostModeDisabled,
				}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestDeviceList(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
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
	c := testutil.NewClient(t, devicesHandler(t))
	got, err := New(c).Get(context.Background(), "KWG32FK99J")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "KWG32FK99J" || got.Attributes.SerialNumber != "KWG32FK99J" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestDeviceGetDecodesMdmMigrationFields(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"orgDevices","id":"D1","attributes":{
			"serialNumber":"D1","status":"ASSIGNED",
			"isMdmMigrationCapable":true,
			"mdmMigrationStatus":"REQUESTED",
			"mdmMigrationDeadlineDateTime":"2026-09-15T17:00:00.000Z"}}}`))
	})
	c := testutil.NewClient(t, h)
	got, err := New(c).Get(context.Background(), "D1")
	if err != nil {
		t.Fatal(err)
	}
	a := got.Attributes
	if !a.IsMdmMigrationCapable || a.MdmMigrationStatus != MdmMigrationStatusRequested ||
		a.MdmMigrationDeadlineDateTime != "2026-09-15T17:00:00.000Z" {
		t.Fatalf("migration fields: %+v", a)
	}
}

func TestAssignedServer(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	srv, err := New(c).AssignedServer(context.Background(), "D1")
	if err != nil {
		t.Fatal(err)
	}
	if srv.ID != "S1" || srv.Attributes.ServerName != "Server-1" {
		t.Fatalf("unexpected: %+v", srv)
	}
}

func TestListMdmServers(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	got, err := New(c).ListMdmServers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "S1" || got[0].Attributes.ServerName != "Server-1" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestMdmServerDevices(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	ids, err := New(c).MdmServerDevices(context.Background(), "S1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Type != "orgDevices" || ids[1].ID != "D2" {
		t.Fatalf("unexpected: %+v", ids)
	}
}

func TestListMdmDevices(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	got, err := New(c).ListMdmDevices(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Two pages joined via links.next.
	if len(got) != 2 || got[0].ID != "M1" || got[1].ID != "M2" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got[0].Type != "mdmDevices" || got[0].Attributes.EnrolledUserID != "U1" || got[1].Attributes.ProductFamily != "iPad" {
		t.Fatalf("attributes: %+v", got)
	}
}

func TestMdmDeviceDetails(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	got, err := New(c).MdmDeviceDetails(context.Background(), "M1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "mdmDeviceDetails" || got.ID != "M1" {
		t.Fatalf("unexpected: %+v", got)
	}
	a := got.Attributes
	if a.SerialNumber != "SER1" || a.OSVersion != "18.5" || a.Platform != "iOS" ||
		a.StorageTotalCapacity != 256 || a.StorageFreeCapacity != 128 ||
		a.DeviceEraseStatus != EraseStatusNotErased || a.DeviceLockStatus != LockStatusUnlocked || a.LostModeStatus != LostModeDisabled {
		t.Fatalf("attributes: %+v", a)
	}
}

func TestMdmServerDeviceList(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
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
