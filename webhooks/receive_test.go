package webhooks

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 公式記事（Understanding webhook events）のテストベクタ。
const (
	vectorSecret    = "This is my secret"
	vectorBody      = "Hello, World!"
	vectorSignature = "hmacsha256=7f062172b01cb00b53ca068614674a3d982a34062a0f5d37687d5e3377e54657"
)

func TestVerifySignatureOfficialVector(t *testing.T) {
	if err := VerifySignature(vectorSecret, []byte(vectorBody), vectorSignature); err != nil {
		t.Fatalf("official test vector: %v", err)
	}
}

func TestVerifySignatureAcceptsCaseAndSpaces(t *testing.T) {
	upper := "HMACSHA256=" + strings.ToUpper(strings.TrimPrefix(vectorSignature, "hmacsha256="))
	for _, sig := range []string{upper, "  " + vectorSignature + " "} {
		if err := VerifySignature(vectorSecret, []byte(vectorBody), sig); err != nil {
			t.Errorf("%q: %v", sig, err)
		}
	}
}

func TestVerifySignatureRejects(t *testing.T) {
	hexDigest := strings.TrimPrefix(vectorSignature, "hmacsha256=")
	for _, tc := range []struct {
		name, secret, body, sig string
	}{
		{"tampered body", vectorSecret, "Hello, World?", vectorSignature},
		{"wrong secret", "another secret", vectorBody, vectorSignature},
		{"empty header", vectorSecret, vectorBody, ""},
		{"no prefix", vectorSecret, vectorBody, hexDigest},
		{"other prefix", vectorSecret, vectorBody, "sha256=" + hexDigest},
		{"not hex", vectorSecret, vectorBody, "hmacsha256=zz"},
		{"truncated digest", vectorSecret, vectorBody, vectorSignature[:len(vectorSignature)-2]},
		{"prefix only", vectorSecret, vectorBody, "hmacsha256="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifySignature(tc.secret, []byte(tc.body), tc.sig)
			if !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("want ErrInvalidSignature, got %v", err)
			}
		})
	}
}

func TestVerifyRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(vectorBody))
	r.Header.Set("x-apple-signature", vectorSignature) // 小文字で届いても Header.Get で取れる
	body, err := VerifyRequest(r, vectorSecret, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != vectorBody {
		t.Fatalf("body = %q", body)
	}
	again, _ := io.ReadAll(r.Body)
	if string(again) != vectorBody {
		t.Fatalf("r.Body was not restored: %q", again)
	}
}

func TestVerifyRequestRejects(t *testing.T) {
	t.Run("bad signature", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(vectorBody))
		r.Header.Set(SignatureHeader, "hmacsha256=00")
		if _, err := VerifyRequest(r, vectorSecret, 0); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("want ErrInvalidSignature, got %v", err)
		}
	})
	t.Run("missing header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(vectorBody))
		if _, err := VerifyRequest(r, vectorSecret, 0); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("want ErrInvalidSignature, got %v", err)
		}
	})
	t.Run("too large", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(vectorBody))
		r.Header.Set(SignatureHeader, vectorSignature)
		if _, err := VerifyRequest(r, vectorSecret, int64(len(vectorBody))-1); !errors.Is(err, ErrBodyTooLarge) {
			t.Fatalf("want ErrBodyTooLarge, got %v", err)
		}
	})
	t.Run("exactly at the limit", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(vectorBody))
		r.Header.Set(SignatureHeader, vectorSignature)
		if _, err := VerifyRequest(r, vectorSecret, int64(len(vectorBody))); err != nil {
			t.Fatalf("body of exactly maxBytes should pass: %v", err)
		}
	})
}

// 公式記事の各イベントの例（プレースホルダ値）。
const (
	exampleAttributeChanged = `{"data":{"type":"orgDeviceAttributeChanged","id":"813ef0fa-ce41-4c3f-88dc-b97096670b26","version":1,` +
		`"attributes":{"orgDeviceSerialNumber":"XABC123DEF4"},` +
		`"relationships":{"instance":{"data":{"type":"orgDevices","id":"XABC123DEF4"},"links":{"self":"https://api-business.apple.com/v1/orgDevices/XABC123DEF4"}}}}}`
	exampleMigrationChanged = `{"data":{"type":"orgDeviceAttributeMigrationDetailsChanged","id":"e00313e9-d0ee-41d4-b13f-f23a23495941","version":1,` +
		`"attributes":{"orgDeviceSerialNumber":"XABC123DEF4"},` +
		`"relationships":{"instance":{"data":{"type":"orgDevices","id":"XABC123DEF4"}}}}}`
	exampleAssignedServerChanged = `{"data":{"type":"orgDeviceRelationshipAssignedServerChanged","id":"812ada8f-d313-4148-b9db-2f0b50a7b009","version":1,` +
		`"attributes":{"orgDeviceSerialNumber":"XABC123DEF4","newMdmServerId":"1F97349736CF4614A94F624E705841AD"},` +
		`"relationships":{"instance":{"data":{"type":"orgDevices","id":"XABC123DEF4"}},` +
		`"newAssignedServer":{"data":{"type":"mdmServers","id":"1F97349736CF4614A94F624E705841AD"}}}}}`
	exampleWarrantyChanged = `{"data":{"type":"orgDeviceRelationshipWarrantyChanged","id":"7de7d1dd-5830-4104-b03e-d861faa1fd1a","version":1,` +
		`"attributes":{"orgDeviceSerialNumber":"XABC123DEF4","appleCareCoverageId":"H2WN4ZX7QK"},` +
		`"relationships":{"instance":{"data":{"type":"orgDevices","id":"XABC123DEF4"}}}}}`
	examplePing = `{"data":{"type":"webhookPingCreated","id":"9be761ff-131c-4652-9df8-b779e8adee29","version":1,` +
		`"attributes":{"createdDateTime":"2026-10-14T17:24:54.096092003Z"}}}`
)

func TestParseNotificationDeviceEvents(t *testing.T) {
	for _, tc := range []struct {
		body, typ, eventType, id string
	}{
		{exampleAttributeChanged, NotificationOrgDeviceAttributeChanged, EventTypeOrgDeviceAttributeChanged, "813ef0fa-ce41-4c3f-88dc-b97096670b26"},
		{exampleMigrationChanged, NotificationOrgDeviceAttributeMigrationDetailsChanged, EventTypeOrgDeviceAttributeMigrationDetailsChanged, "e00313e9-d0ee-41d4-b13f-f23a23495941"},
		{exampleAssignedServerChanged, NotificationOrgDeviceRelationshipAssignedServerChanged, EventTypeOrgDeviceRelationshipAssignedServerChanged, "812ada8f-d313-4148-b9db-2f0b50a7b009"},
		{exampleWarrantyChanged, NotificationOrgDeviceRelationshipWarrantyChanged, EventTypeOrgDeviceRelationshipWarrantyChanged, "7de7d1dd-5830-4104-b03e-d861faa1fd1a"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			n, err := ParseNotification([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if n.Type != tc.typ || n.ID != tc.id || n.Version != 1 || n.EventType() != tc.eventType || n.IsPing() {
				t.Fatalf("notification: %+v (eventType=%q)", n, n.EventType())
			}
			inst, ok := n.Instance()
			if !ok || inst.Type != "orgDevices" || inst.ID != "XABC123DEF4" {
				t.Fatalf("instance: %+v (ok=%v)", inst, ok)
			}
			var d DeviceChanged // 4 種とも orgDeviceSerialNumber を持つ
			if err := n.DecodeAttributes(&d); err != nil || d.OrgDeviceSerialNumber != "XABC123DEF4" {
				t.Fatalf("attributes: %+v (err=%v)", d, err)
			}
		})
	}
}

func TestParseNotificationEventSpecificAttributes(t *testing.T) {
	n, err := ParseNotification([]byte(exampleAssignedServerChanged))
	if err != nil {
		t.Fatal(err)
	}
	var a AssignedServerChanged
	if err := n.DecodeAttributes(&a); err != nil || a.NewMdmServerID != "1F97349736CF4614A94F624E705841AD" {
		t.Fatalf("AssignedServerChanged: %+v (err=%v)", a, err)
	}
	if rel := n.Relationships["newAssignedServer"]; rel.Data.Type != "mdmServers" || rel.Data.ID != a.NewMdmServerID {
		t.Fatalf("newAssignedServer: %+v", rel)
	}

	n, err = ParseNotification([]byte(exampleWarrantyChanged))
	if err != nil {
		t.Fatal(err)
	}
	var w WarrantyChanged
	if err := n.DecodeAttributes(&w); err != nil || w.AppleCareCoverageID != "H2WN4ZX7QK" {
		t.Fatalf("WarrantyChanged: %+v (err=%v)", w, err)
	}
}

func TestParseNotificationPing(t *testing.T) {
	n, err := ParseNotification([]byte(examplePing))
	if err != nil {
		t.Fatal(err)
	}
	if !n.IsPing() || n.EventType() != "" {
		t.Fatalf("ping: %+v (eventType=%q)", n, n.EventType())
	}
	if _, ok := n.Instance(); ok {
		t.Fatal("ping has no instance relationship")
	}
	var p PingCreated
	if err := n.DecodeAttributes(&p); err != nil || p.CreatedDateTime != "2026-10-14T17:24:54.096092003Z" {
		t.Fatalf("PingCreated: %+v (err=%v)", p, err)
	}
}

func TestParseNotificationUnknownTypeKeepsRawAttributes(t *testing.T) {
	n, err := ParseNotification([]byte(`{"data":{"type":"somethingNew","id":"X","version":2,"attributes":{"k":"v"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "somethingNew" || n.EventType() != "" || string(n.Attributes) != `{"k":"v"}` {
		t.Fatalf("unknown type: %+v", n)
	}
}

func TestParseNotificationRejects(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":     `nope`,
		"no data":      `{"type":"orgDeviceAttributeChanged"}`,
		"no data.type": `{"data":{"id":"X"}}`,
		"null data":    `{"data":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseNotification([]byte(body)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

// 配信一覧で返るイベントの payload（JSON 文字列）も同じ関数で解析できる。
func TestParseNotificationFromDeliveredEventPayload(t *testing.T) {
	ev := Event{Type: "webhookEvents", ID: "9be761ff-131c-4652-9df8-b779e8adee29", Attributes: EventAttributes{IsPingEvent: true, Payload: examplePing}}
	n, err := ParseNotification([]byte(ev.Attributes.Payload))
	if err != nil {
		t.Fatal(err)
	}
	if !n.IsPing() || n.ID != ev.ID {
		t.Fatalf("notification: %+v", n)
	}
}

// 実機（2026-11-06）の配信一覧で返った ping の payload。公式記事の例と違い id が無い。
func TestParseNotificationLivePingWithoutID(t *testing.T) {
	payload := `{ "data" : { "type" : "webhookPingCreated", "version" : 1, "attributes" : { "createdDateTime" : "2026-11-06T02:18:13.324391366Z" } } }`
	n, err := ParseNotification([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if !n.IsPing() || n.ID != "" || n.Version != 1 {
		t.Fatalf("notification: %+v", n)
	}
	var p PingCreated
	if err := n.DecodeAttributes(&p); err != nil || p.CreatedDateTime != "2026-11-06T02:18:13.324391366Z" {
		t.Fatalf("PingCreated: %+v (err=%v)", p, err)
	}
}
