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
		case strings.HasSuffix(p, "/appleCareCoverage") && r.Method == http.MethodGet:
			// 実 AB の応答は data が配列（links.self と meta.paging.limit 100 付き）。
			// テストごとに応答を変えるため device ID で分ける。
			self := "http://" + r.Host + p
			meta := applebusiness.Meta{Paging: applebusiness.Paging{Limit: 100}}
			switch strings.TrimSuffix(strings.TrimPrefix(p, "/v1/orgDevices/"), "/appleCareCoverage") {
			case "AC2":
				testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[AppleCareCoverageAttributes]{
					Data: []applebusiness.ResourceObject[AppleCareCoverageAttributes]{
						{Type: "appleCareCoverage", ID: "325046665035", Attributes: AppleCareCoverageAttributes{
							Status: "INACTIVE", PaymentType: "PAID_UP_FRONT", Description: "AppleCare",
							AgreementNumber: "325046665035", StartDateTime: "2019-05-05T00:00:00Z", EndDateTime: "2020-09-30T00:00:00Z",
							ContractCancelDateTime: "2020-09-16T00:00:00Z", IsCanceled: true,
						}},
						{Type: "appleCareCoverage", ID: "AC2", Attributes: AppleCareCoverageAttributes{
							Status: "ACTIVE", PaymentType: "NONE", Description: "Limited Warranty",
							StartDateTime: "2024-01-01T00:00:00Z", EndDateTime: "2025-01-01T00:00:00Z",
						}},
					},
					Links: applebusiness.Links{Self: self},
					Meta:  meta,
				})
			case "AC0":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[],"links":{"self":"` + self + `"},"meta":{"paging":{"limit":100}}}`))
			case "AC1":
				// agreementNumber / contractCancelDateTime が null の要素（id はシリアル番号）。
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[{"type":"appleCareCoverage","id":"AC1","attributes":{` +
					`"contractCancelDateTime":null,"startDateTime":"2025-04-01T00:00:00Z","isRenewable":false,` +
					`"isCanceled":false,"description":"Limited Warranty","agreementNumber":null,` +
					`"endDateTime":"2026-04-01T00:00:00Z","status":"ACTIVE","paymentType":"NONE"}}],` +
					`"links":{"self":"` + self + `"},"meta":{"paging":{"limit":100}}}`))
			default:
				http.Error(w, "not found: "+p, http.StatusNotFound)
			}
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

func TestAppleCareCoveragesMultiple(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	svc := New(c)
	all, err := svc.AppleCareCoverages(context.Background(), "AC2")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "325046665035" || all[1].ID != "AC2" {
		t.Fatalf("unexpected: %+v", all)
	}
	a := all[0].Attributes
	if all[0].Type != "appleCareCoverage" || a.Status != "INACTIVE" || a.PaymentType != "PAID_UP_FRONT" ||
		a.AgreementNumber != "325046665035" || a.ContractCancelDateTime != "2020-09-16T00:00:00Z" || !a.IsCanceled {
		t.Fatalf("first coverage: %+v", all[0])
	}
	if all[1].Attributes.Status != "ACTIVE" || all[1].Attributes.EndDateTime != "2025-01-01T00:00:00Z" {
		t.Fatalf("second coverage: %+v", all[1])
	}

	first, err := svc.AppleCareCoverage(context.Background(), "AC2")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || first.ID != "325046665035" || first.Attributes.AgreementNumber != "325046665035" {
		t.Fatalf("AppleCareCoverage should return the first element: %+v", first)
	}
}

func TestAppleCareCoveragesEmpty(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	svc := New(c)
	all, err := svc.AppleCareCoverages(context.Background(), "AC0")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("want 0 coverages, got %+v", all)
	}

	first, err := svc.AppleCareCoverage(context.Background(), "AC0")
	if err != nil {
		t.Fatal(err)
	}
	if first != nil {
		t.Fatalf("want nil for no coverage, got %+v", first)
	}
}

func TestAppleCareCoveragesSingleWithNulls(t *testing.T) {
	c := testutil.NewClient(t, devicesHandler(t))
	svc := New(c)
	all, err := svc.AppleCareCoverages(context.Background(), "AC1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "AC1" || all[0].Type != "appleCareCoverage" {
		t.Fatalf("unexpected: %+v", all)
	}
	a := all[0].Attributes
	// null はゼロ値（空文字列）に落ちる。
	if a.AgreementNumber != "" || a.ContractCancelDateTime != "" ||
		a.Status != "ACTIVE" || a.PaymentType != "NONE" || a.EndDateTime != "2026-04-01T00:00:00Z" {
		t.Fatalf("attributes: %+v", a)
	}

	first, err := svc.AppleCareCoverage(context.Background(), "AC1")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || first.ID != "AC1" || first.Attributes != a {
		t.Fatalf("AppleCareCoverage: %+v", first)
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
