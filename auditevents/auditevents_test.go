package auditevents

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func newClient(t *testing.T, h http.Handler) *applebusiness.Client {
	t.Helper()
	api := httptest.NewServer(h)
	t.Cleanup(api.Close)
	tok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	c := newClient(t, h)

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
	c := newClient(t, h)
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

func TestPayloadApiAccountKey(t *testing.T) {
	const body = `{"data":[{"type":"auditEvents","id":"E2","attributes":{` +
		`"type":"API_ACCOUNT_CREATED_WITH_KEY",` +
		`"eventDataPropertyKey":"eventDataApiAccountCreatedWithKey",` +
		`"eventDataApiAccountCreatedWithKey":{"keyId":"K-123"}` +
		`}}],"links":{},"meta":{}}`
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	events, err := New(c).ListRange(context.Background(), time.Now().Add(-time.Hour), time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Attributes.Type != TypeApiAccountCreatedWithKey {
		t.Fatalf("unexpected: %+v", events)
	}
	var p ApiAccountKey
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
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
