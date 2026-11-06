package auditevents

import "testing"

// Apple Business API 2.8（2026-11-04）で追加された Webhook の監査イベント。

func TestPayloadWebhookCreatedAndUpdated(t *testing.T) {
	for _, tc := range []struct{ typ, key string }{
		{TypeWebhookCreated, "eventDataWebhookCreated"},
		{TypeWebhookUpdated, "eventDataWebhookUpdated"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			ev := listOne(t, eventJSON(tc.typ, CategoryWebhookActivity, SubjectWebhook, tc.key,
				`{"name":"device-sync","enabled":"true","eventTypes":["ORG_DEVICE_ATTRIBUTE_CHANGED"],"url":"https://example.com/hook"}`))
			a := ev.Attributes
			if a.Type != tc.typ || a.Category != CategoryWebhookActivity || a.SubjectType != SubjectWebhook {
				t.Fatalf("common attributes: %+v", a)
			}
			var w Webhook
			if err := a.Payload(&w); err != nil {
				t.Fatal(err)
			}
			if w.Name != "device-sync" || w.Enabled != "true" || w.URL != "https://example.com/hook" ||
				len(w.EventTypes) != 1 || w.EventTypes[0] != "ORG_DEVICE_ATTRIBUTE_CHANGED" {
				t.Fatalf("payload: %+v", w)
			}
		})
	}
}

func TestPayloadWebhookDeletedIsEmpty(t *testing.T) {
	ev := listOne(t, eventJSON(TypeWebhookDeleted, CategoryWebhookActivity, SubjectWebhook, "eventDataWebhookDeleted", `{}`))
	var w Webhook
	if err := ev.Attributes.Payload(&w); err != nil {
		t.Fatal(err)
	}
	if w.Name != "" || w.Enabled != "" || w.URL != "" || len(w.EventTypes) != 0 {
		t.Fatalf("payload should be empty: %+v", w)
	}
}
