package devices

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func TestDeviceFilterQueryZeroValue(t *testing.T) {
	q := DeviceFilter{}.Query(nil)
	if q == nil || len(q) != 0 {
		t.Fatalf("zero filter should give empty, non-nil values: %v", q)
	}
}

func TestDeviceFilterQueryAllFields(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	f := DeviceFilter{
		SerialNumbers:       []string{"C02ABC123DEF", "", "DMPABC456GHI"},
		OrderNumbers:        []string{"ORD-1"},
		ProductFamilies:     []string{ProductFamilyIPhone, ProductFamilyIPad},
		PurchaseSourceTypes: []string{PurchaseSourceTypeReseller},
		Statuses:            []string{DeviceStatusAssigned},
		AddedToOrgDateTime: TimeFilter{
			GTE: time.Date(2025, 1, 1, 9, 0, 0, 0, jst), // UTC に変換して送る
			LT:  time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		OrderDateTime: TimeFilter{
			Equal: []time.Time{time.Date(2024, 12, 24, 0, 0, 0, 0, time.UTC), {}, time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)},
			GT:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			LTE:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}
	want := url.Values{
		"filter[serialNumber]":            {"C02ABC123DEF,DMPABC456GHI"},
		"filter[orderNumber]":             {"ORD-1"},
		"filter[productFamily]":           {"iPhone,iPad"},
		"filter[purchaseSourceType]":      {"RESELLER"},
		"filter[status]":                  {"ASSIGNED"},
		"filter[addedToOrgDateTime][GTE]": {"2025-01-01T00:00:00Z"},
		"filter[addedToOrgDateTime][LT]":  {"2025-07-01T00:00:00Z"},
		"filter[orderDateTime]":           {"2024-12-24T00:00:00Z,2024-12-25T00:00:00Z"},
		"filter[orderDateTime][GT]":       {"2024-01-01T00:00:00Z"},
		"filter[orderDateTime][LTE]":      {"2024-12-31T23:59:59Z"},
	}
	if got := f.Query(nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("Query mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestDeviceFilterQueryKeepsBaseUnmodified(t *testing.T) {
	base := url.Values{"limit": {"100"}, "filter[status]": {"UNASSIGNED"}, "filter[orderNumber]": {"KEEP"}}
	snapshot := url.Values{"limit": {"100"}, "filter[status]": {"UNASSIGNED"}, "filter[orderNumber]": {"KEEP"}}

	got := DeviceFilter{Statuses: []string{DeviceStatusAssigned}}.Query(base)

	if !reflect.DeepEqual(base, snapshot) {
		t.Fatalf("base was modified: %v", base)
	}
	// 値のあるフィールドだけが base の同名パラメータを置き換える。
	want := url.Values{"limit": {"100"}, "filter[status]": {"ASSIGNED"}, "filter[orderNumber]": {"KEEP"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Query mismatch\n got: %v\nwant: %v", got, want)
	}
	got["limit"][0] = "1"
	if base.Get("limit") != "100" {
		t.Fatal("result shares backing slices with base")
	}
}

func TestListFilteredSendsFilterQuery(t *testing.T) {
	var gotRaw string
	var gotQuery url.Values
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/orgDevices" || r.Method != http.MethodGet {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
			return
		}
		gotRaw, gotQuery = r.URL.RawQuery, r.URL.Query()
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[DeviceAttributes]{
			Data: []applebusiness.ResourceObject[DeviceAttributes]{
				{Type: "orgDevices", ID: "D1", Attributes: DeviceAttributes{SerialNumber: "D1", ProductFamily: ProductFamilyIPhone, Status: DeviceStatusAssigned}},
			},
		})
	})
	c := testutil.NewClient(t, h)
	q := url.Values{"limit": {"1000"}}

	got, err := New(c).ListFiltered(context.Background(), DeviceFilter{
		ProductFamilies:    []string{ProductFamilyIPhone, ProductFamilyIPad},
		Statuses:           []string{DeviceStatusAssigned},
		AddedToOrgDateTime: TimeFilter{GTE: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "D1" || got[0].Attributes.ProductFamily != ProductFamilyIPhone {
		t.Fatalf("unexpected: %+v", got)
	}
	for k, v := range map[string]string{
		"limit":                           "1000",
		"filter[productFamily]":           "iPhone,iPad",
		"filter[status]":                  "ASSIGNED",
		"filter[addedToOrgDateTime][GTE]": "2025-01-01T00:00:00Z",
	} {
		if gotQuery.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, gotQuery.Get(k), v)
		}
	}
	// 角括弧とカンマはパーセントエンコードされて送られる。
	if !strings.Contains(gotRaw, "filter%5BproductFamily%5D=iPhone%2CiPad") {
		t.Errorf("raw query: %s", gotRaw)
	}
	if len(q) != 1 || q.Get("limit") != "1000" {
		t.Errorf("q was modified: %v", q)
	}
}

func TestListFilteredZeroFilterSendsNoQuery(t *testing.T) {
	gotRaw := "unset"
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw = r.URL.RawQuery
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[DeviceAttributes]{})
	})
	c := testutil.NewClient(t, h)
	if _, err := New(c).ListFiltered(context.Background(), DeviceFilter{}, nil); err != nil {
		t.Fatal(err)
	}
	if gotRaw != "" {
		t.Fatalf("zero filter should send no query, got %q", gotRaw)
	}
}
