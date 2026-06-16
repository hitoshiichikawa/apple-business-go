package auditevents

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

// 実APIで観測した形（共通エンベロープ + eventData<Event>）。
const auditEventsJSON = `{"data":[{"type":"auditEvents","id":"E1","attributes":{` +
	`"eventDateTime":"2026-06-02T06:18:50.789Z","type":"DEVICE_ASSIGNED_TO_SERVER",` +
	`"category":"DEVICE_ACTIVITY","actorType":"USER","outcome":"SUCCESS",` +
	`"eventDataPropertyKey":"eventDataDeviceAssignedToServer",` +
	`"eventDataDeviceAssignedToServer":{"serialNumber":"KWG32FK99J","targetServerName":"Server-1"}` +
	`}}],"links":{},"meta":{"paging":{"limit":3}}}`

func TestListRange(t *testing.T) {
	var gotStart, gotEnd, gotPath string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q := r.URL.Query()
		gotStart = q.Get("filter[startTimestamp]")
		gotEnd = q.Get("filter[endTimestamp]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(auditEventsJSON))
	})
	c := testutil.NewClient(t, h)

	start := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	events, err := New(c).ListRange(context.Background(), start, end, nil)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/v1/auditEvents" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotStart != start.Format(time.RFC3339) {
		t.Fatalf("filter[startTimestamp] = %q want %q", gotStart, start.Format(time.RFC3339))
	}
	if gotEnd != end.Format(time.RFC3339) {
		t.Fatalf("filter[endTimestamp] = %q want %q", gotEnd, end.Format(time.RFC3339))
	}

	if len(events) != 1 {
		t.Fatalf("len = %d", len(events))
	}
	ev := events[0]
	if ev.Attributes.Type != "DEVICE_ASSIGNED_TO_SERVER" {
		t.Fatalf("type = %q", ev.Attributes.Type)
	}
	if ev.Attributes.EventDataPropertyKey != "eventDataDeviceAssignedToServer" {
		t.Fatalf("eventDataPropertyKey = %q", ev.Attributes.EventDataPropertyKey)
	}

	var p DeviceAssignedToServer
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatalf("Payload: %v", err)
	}
	if p.SerialNumber != "KWG32FK99J" || p.TargetServerName != "Server-1" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestListRangeOmitsEndWhenZero(t *testing.T) {
	var gotStart, gotEnd string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotStart = q.Get("filter[startTimestamp]")
		gotEnd = q.Get("filter[endTimestamp]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"links":{},"meta":{}}`))
	})
	c := testutil.NewClient(t, h)
	if _, err := New(c).ListRange(context.Background(), time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC), time.Time{}, nil); err != nil {
		t.Fatal(err)
	}
	if gotStart == "" {
		t.Fatal("filter[startTimestamp] should be set")
	}
	if gotEnd != "" {
		t.Fatalf("filter[endTimestamp] should be empty, got %q", gotEnd)
	}
}

func TestPayloadAPIAccountKey(t *testing.T) {
	const body = `{"data":[{"type":"auditEvents","id":"E2","attributes":{` +
		`"type":"API_ACCOUNT_CREATED_WITH_KEY",` +
		`"eventDataPropertyKey":"eventDataApiAccountCreatedWithKey",` +
		`"eventDataApiAccountCreatedWithKey":{"keyId":"K-123"}` +
		`}}],"links":{},"meta":{}}`
	c := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	events, err := New(c).ListRange(context.Background(), time.Now().Add(-time.Hour), time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Attributes.Type != TypeAPIAccountCreatedWithKey {
		t.Fatalf("unexpected: %+v", events)
	}
	var p APIAccountKey
	if err := events[0].Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if p.KeyID != "K-123" {
		t.Fatalf("keyId: %q", p.KeyID)
	}
}

func TestPayloadAccountRoleLocationChanged(t *testing.T) {
	const body = `{"data":[{"type":"auditEvents","id":"E3","attributes":{` +
		`"type":"ACCOUNT_ROLE_LOCATION_CHANGED",` +
		`"eventDataPropertyKey":"eventDataAccountRoleLocationChanged",` +
		`"eventDataAccountRoleLocationChanged":{"accountRoleLocationList":[{"roleName":"ADMINISTRATOR","locationUniqueIdentifier":"LOC1"}]}` +
		`}}],"links":{},"meta":{}}`
	c := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	events, err := New(c).ListRange(context.Background(), time.Now().Add(-time.Hour), time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Attributes.Type != TypeAccountRoleLocationChanged {
		t.Fatalf("unexpected: %+v", events)
	}
	var p AccountRoleLocationChanged
	if err := events[0].Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.AccountRoleLocationList) != 1 || p.AccountRoleLocationList[0].RoleName != "ADMINISTRATOR" ||
		p.AccountRoleLocationList[0].LocationUniqueIdentifier != "LOC1" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestListRangeDoesNotMutateQuery(t *testing.T) {
	c := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	s := New(c)

	q := url.Values{"limit": {"5"}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.ListRange(context.Background(), start, time.Time{}, q); err != nil {
		t.Fatalf("ListRange: %v", err)
	}
	if len(q) != 1 || q.Get("limit") != "5" {
		t.Fatalf("caller url.Values was mutated: %v", q)
	}
	// nil の q も従来どおり許容される。
	if _, err := s.ListRange(context.Background(), start, time.Time{}, nil); err != nil {
		t.Fatalf("ListRange(nil q): %v", err)
	}
}
