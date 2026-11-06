package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

// recorded は 1 リクエスト分の記録。
type recorded struct {
	method, path, rawQuery string
	body                   map[string]any
}

// recorder は受けたリクエストを記録し、respond の結果を返すハンドラを作る。
func recorder(t *testing.T, got *[]recorded, respond func(w http.ResponseWriter, r *http.Request)) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recorded{method: r.Method, path: r.URL.EscapedPath(), rawQuery: r.URL.RawQuery}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			if err := json.Unmarshal(b, &rec.body); err != nil {
				t.Errorf("request body is not JSON: %s", b)
			}
		}
		*got = append(*got, rec)
		respond(w, r)
	})
}

func webhookJSON(id, name string, enabled bool) applebusiness.SingleResponse[Attributes] {
	return applebusiness.SingleResponse[Attributes]{Data: Webhook{Type: "webhooks", ID: id, Attributes: Attributes{
		Name: name, Enabled: enabled, URL: "https://example.com/hook",
		EventTypes: []string{EventTypeOrgDeviceAttributeChanged},
	}}}
}

// mustJSON は v を JSON 経由で map/[]any に正規化する（期待値との比較用）。
func mustJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestList(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[Attributes]{Data: []Webhook{
			{Type: "webhooks", ID: "W1", Attributes: Attributes{Name: "a", Enabled: true, EventTypes: []string{EventTypeOrgDeviceAttributeChanged}}},
			{Type: "webhooks", ID: "W2", Attributes: Attributes{Name: "b"}},
		}})
	}))
	list, err := New(c).List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].method != http.MethodGet || got[0].path != "/v1/webhooks" {
		t.Fatalf("request: %+v", got)
	}
	if len(list) != 2 || list[0].ID != "W1" || !list[0].Attributes.Enabled || list[1].Attributes.Enabled {
		t.Fatalf("decoded: %+v", list)
	}
	if list[0].Attributes.EventTypes[0] != EventTypeOrgDeviceAttributeChanged {
		t.Fatalf("eventTypes: %+v", list[0].Attributes.EventTypes)
	}
}

func TestGetEscapesIDAndReportsNotFound(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusNotFound, map[string]any{"errors": []map[string]string{{"status": "404", "code": "NOT_FOUND"}}})
	}))
	_, err := New(c).Get(context.Background(), "a/b")
	if !applebusiness.IsNotFound(err) {
		t.Fatalf("want IsNotFound, got %v", err)
	}
	if got[0].path != "/v1/webhooks/a%2Fb" {
		t.Fatalf("path = %q", got[0].path)
	}
}

func TestCreateSendsEveryRequiredAttribute(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusCreated, webhookJSON("W1", "n", false))
	}))
	wh, err := New(c).Create(context.Background(), CreateInput{
		Name: "n", URL: "https://example.com/hook", Secret: "s3cret",
		EventTypes: []string{EventTypeOrgDeviceAttributeChanged, EventTypeOrgDeviceRelationshipWarrantyChanged},
		Enabled:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].method != http.MethodPost || got[0].path != "/v1/webhooks" {
		t.Fatalf("request: %+v", got[0])
	}
	want := mustJSON(t, map[string]any{"data": map[string]any{
		"type": "webhooks",
		"attributes": map[string]any{
			"name": "n", "url": "https://example.com/hook", "secret": "s3cret", "enabled": false, // false も省略しない
			"eventTypes": []string{"ORG_DEVICE_ATTRIBUTE_CHANGED", "ORG_DEVICE_RELATIONSHIP_WARRANTY_CHANGED"},
		},
	}})
	if !reflect.DeepEqual(mustJSON(t, got[0].body), want) {
		t.Fatalf("body:\n got %v\nwant %v", got[0].body, want)
	}
	if wh.ID != "W1" || wh.Type != "webhooks" {
		t.Fatalf("decoded: %+v", wh)
	}
}

func TestCreateSendsEmptyEventTypesInsteadOfNull(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusCreated, webhookJSON("W1", "n", true))
	}))
	if _, err := New(c).Create(context.Background(), CreateInput{Name: "n", URL: "https://example.com/hook", Secret: "s", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	attrs := got[0].body["data"].(map[string]any)["attributes"].(map[string]any)
	if v, ok := attrs["eventTypes"].([]any); !ok || len(v) != 0 {
		t.Fatalf("eventTypes = %#v, want []", attrs["eventTypes"])
	}
}

func TestUpdateSendsOnlyChangedAttributes(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusOK, webhookJSON("W1", "renamed", false))
	}))
	name, enabled := "renamed", false
	wh, err := New(c).Update(context.Background(), "W1", UpdateInput{Name: &name, Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].method != http.MethodPatch || got[0].path != "/v1/webhooks/W1" {
		t.Fatalf("request: %+v", got[0])
	}
	want := mustJSON(t, map[string]any{"data": map[string]any{
		"type": "webhooks", "id": "W1",
		"attributes": map[string]any{"name": "renamed", "enabled": false},
	}})
	if !reflect.DeepEqual(mustJSON(t, got[0].body), want) {
		t.Fatalf("body:\n got %v\nwant %v", got[0].body, want)
	}
	if wh.Attributes.Name != "renamed" || wh.Attributes.Enabled {
		t.Fatalf("decoded: %+v", wh.Attributes)
	}
}

func TestDelete(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := New(c).Delete(context.Background(), "W1"); err != nil {
		t.Fatal(err)
	}
	if got[0].method != http.MethodDelete || got[0].path != "/v1/webhooks/W1" {
		t.Fatalf("request: %+v", got[0])
	}
}

func TestDeliveriesFollowsPagesAndCollectsEvents(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			next := "http://" + r.Host + "/v1/webhooks/W1/deliveries?cursor=p2&include=event"
			_, _ = w.Write([]byte(`{"data":[{"type":"webhookDeliveries","id":"D1","attributes":{` +
				`"deliveryState":"FAILED","errorMessage":"UnknownHostException","redelivery":false,` +
				`"createdDateTime":"2026-11-06T01:00:00Z","request":{"url":"https://example.invalid/h"}},` +
				`"relationships":{"event":{"data":{"type":"webhookEvents","id":"E1"}}}}],` +
				`"included":[{"type":"webhookEvents","id":"E1","attributes":{"isPingEvent":true,` +
				`"payload":"{\"data\":{\"type\":\"webhookPingCreated\"}}","createdDateTime":"2026-11-06T01:00:00Z"}}],` +
				`"links":{"next":"` + next + `"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"type":"webhookDeliveries","id":"D2","attributes":{` +
			`"deliveryState":"SUCCEEDED","redelivery":true,"response":{"httpStatusCode":204}},` +
			`"relationships":{"event":{"data":{"type":"webhookEvents","id":"E2"}}}}],` +
			`"included":[{"type":"webhookEvents","id":"E2","attributes":{"eventType":"ORG_DEVICE_ATTRIBUTE_CHANGED",` +
			`"payload":"{}"}}],"links":{}}`))
	}))
	q := url.Values{"limit": {"100"}}
	list, err := New(c).Deliveries(context.Background(), "W1", DeliveryFilter{
		States:           []string{DeliveryStateFailed, "", DeliveryStatePending},
		CreatedAtOrAfter: time.Date(2026, 11, 6, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		CreatedBefore:    time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC),
		IncludeEvents:    true,
	}, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].path != "/v1/webhooks/W1/deliveries" {
		t.Fatalf("requests: %+v", got)
	}
	first, _ := url.ParseQuery(got[0].rawQuery)
	for k, v := range map[string]string{
		"limit":                            "100",
		"filter[deliveryState]":            "FAILED,PENDING",
		"filter[createdDateTimeAtOrAfter]": "2026-11-06T00:00:00Z",
		"filter[createdDateTimeAtBefore]":  "2026-11-07T00:00:00Z",
		"include":                          "event",
	} {
		if first.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, first.Get(k), v)
		}
	}
	if len(q) != 1 {
		t.Errorf("q was modified: %v", q)
	}
	if len(list.Deliveries) != 2 || list.Deliveries[0].ID != "D1" || list.Deliveries[1].ID != "D2" {
		t.Fatalf("deliveries: %+v", list.Deliveries)
	}
	d1 := list.Deliveries[0]
	if d1.Attributes.DeliveryState != DeliveryStateFailed || d1.Attributes.ErrorMessage == "" ||
		d1.Attributes.Request == nil || d1.Attributes.Request.URL != "https://example.invalid/h" || d1.EventID() != "E1" {
		t.Fatalf("D1: %+v", d1)
	}
	if e, ok := list.Event(d1); !ok || !e.Attributes.IsPingEvent || !strings.Contains(e.Attributes.Payload, "webhookPingCreated") {
		t.Fatalf("event for D1: %+v (found=%v)", e, ok)
	}
	d2 := list.Deliveries[1]
	if !d2.Attributes.Redelivery || d2.Attributes.Response == nil || d2.Attributes.Response.HTTPStatusCode != 204 {
		t.Fatalf("D2: %+v", d2)
	}
	if e, ok := list.Event(d2); !ok || e.Attributes.EventType != EventTypeOrgDeviceAttributeChanged {
		t.Fatalf("event for D2: %+v (found=%v)", e, ok)
	}
}

func TestDeliveriesZeroFilterSendsNoQuery(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusOK, map[string]any{"data": []any{}, "links": map[string]any{}})
	}))
	list, err := New(c).Deliveries(context.Background(), "W1", DeliveryFilter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].rawQuery != "" {
		t.Fatalf("query = %q, want none", got[0].rawQuery)
	}
	if len(list.Deliveries) != 0 || list.Events == nil {
		t.Fatalf("list: %+v", list)
	}
}

func TestRedeliverSendsTemplateRelationship(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"type":"webhookDeliveries","id":"D9","attributes":{"deliveryState":"PENDING","redelivery":true},` +
			`"relationships":{"event":{"data":{"type":"webhookEvents","id":"E1"}}}},"links":{}}`))
	}))
	d, err := New(c).Redeliver(context.Background(), "D1")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].method != http.MethodPost || got[0].path != "/v1/webhookDeliveries" {
		t.Fatalf("request: %+v", got[0])
	}
	want := mustJSON(t, map[string]any{"data": map[string]any{
		"type": "webhookDeliveries",
		"relationships": map[string]any{
			"template": map[string]any{"data": map[string]any{"type": "webhookDeliveries", "id": "D1"}},
		},
	}})
	if !reflect.DeepEqual(mustJSON(t, got[0].body), want) {
		t.Fatalf("body:\n got %v\nwant %v", got[0].body, want)
	}
	if d.ID != "D9" || !d.Attributes.Redelivery || d.EventID() != "E1" {
		t.Fatalf("decoded: %+v", d)
	}
}

func TestPingSendsWebhookRelationship(t *testing.T) {
	var got []recorded
	c := testutil.NewClient(t, recorder(t, &got, func(w http.ResponseWriter, _ *http.Request) {
		testutil.WriteJSON(t, w, http.StatusCreated, map[string]any{"data": map[string]string{"type": "webhookPings", "id": "P1"}, "links": map[string]any{}})
	}))
	id, err := New(c).Ping(context.Background(), "W1")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].method != http.MethodPost || got[0].path != "/v1/webhookPings" {
		t.Fatalf("request: %+v", got[0])
	}
	want := mustJSON(t, map[string]any{"data": map[string]any{
		"type": "webhookPings",
		"relationships": map[string]any{
			"webhook": map[string]any{"data": map[string]any{"type": "webhooks", "id": "W1"}},
		},
	}})
	if !reflect.DeepEqual(mustJSON(t, got[0].body), want) {
		t.Fatalf("body:\n got %v\nwant %v", got[0].body, want)
	}
	if id != "P1" {
		t.Fatalf("id = %q", id)
	}
}
