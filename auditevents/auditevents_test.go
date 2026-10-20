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

// ---------------------------------------------------------------------------
// Apple Business API 2.6（2026-09-23）で追加された監査イベント
// ---------------------------------------------------------------------------

// listOne は body を返すサーバから 1 件取得する。
func listOne(t *testing.T, body string) AuditEvent {
	t.Helper()
	c := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	events, err := New(c).ListRange(context.Background(), time.Now().Add(-time.Hour), time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len = %d", len(events))
	}
	return events[0]
}

// eventJSON は共通エンベロープ + eventData<Event> の 1 件応答を組み立てる。
func eventJSON(typ, category, subjectType, key, data string) string {
	return `{"data":[{"type":"auditEvents","id":"E","attributes":{` +
		`"eventDateTime":"2026-09-23T01:02:03.000Z","type":"` + typ + `",` +
		`"category":"` + category + `","actorType":"USER","subjectType":"` + subjectType + `","outcome":"SUCCESS",` +
		`"eventDataPropertyKey":"` + key + `","` + key + `":` + data +
		`}}],"links":{},"meta":{}}`
}

func TestPayloadRoleUpdated(t *testing.T) {
	ev := listOne(t, eventJSON(TypeRoleUpdated, CategoryAccessManagement, SubjectRole, "eventDataRoleUpdated",
		`{"privilegesAdded":["DEVICE_READ","DEVICE_WRITE"],"privilegesRemoved":["USER_READ"]}`))
	a := ev.Attributes
	if a.Type != TypeRoleUpdated || a.Category != CategoryAccessManagement || a.SubjectType != SubjectRole ||
		a.ActorType != ActorUser || a.Outcome != OutcomeSuccess {
		t.Fatalf("envelope: %+v", a)
	}
	var p RoleUpdated
	if err := a.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.PrivilegesAdded) != 2 || p.PrivilegesAdded[1] != "DEVICE_WRITE" ||
		len(p.PrivilegesRemoved) != 1 || p.PrivilegesRemoved[0] != "USER_READ" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadRoleCreated(t *testing.T) {
	ev := listOne(t, eventJSON(TypeRoleCreated, CategoryAccessManagement, SubjectRole, "eventDataRoleCreated",
		`{"privileges":["DEVICE_READ"]}`))
	var p RoleCreated
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.Privileges) != 1 || p.Privileges[0] != "DEVICE_READ" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadIDPCreated(t *testing.T) {
	ev := listOne(t, eventJSON(TypeIDPCreated, CategoryAccessManagement, SubjectIDP, "eventDataIdpCreated",
		`{"vendorName":"MICROSOFT_OIDC","protocol":"OIDC"}`))
	if ev.Attributes.SubjectType != SubjectIDP {
		t.Fatalf("subjectType = %q", ev.Attributes.SubjectType)
	}
	var p IDPCreated
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if p.VendorName != IDPVendorMicrosoftOIDC || p.Protocol != IDPProtocolOIDC {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadIDPUpdatedAndDeletedShareShape(t *testing.T) {
	for _, tc := range []struct{ typ, key string }{
		{TypeIDPUpdated, "eventDataIdpUpdated"},
		{TypeIDPDeleted, "eventDataIdpDeleted"},
	} {
		ev := listOne(t, eventJSON(tc.typ, CategoryAccessManagement, SubjectIDP, tc.key, `{"vendorName":"GENERIC_IDP"}`))
		var p IDP
		if err := ev.Attributes.Payload(&p); err != nil {
			t.Fatalf("%s: %v", tc.typ, err)
		}
		if p.VendorName != IDPVendorGenericIDP {
			t.Fatalf("%s: payload %+v", tc.typ, p)
		}
	}
}

func TestPayloadIDPAccountSync(t *testing.T) {
	ev := listOne(t, eventJSON(TypeIDPAccountSyncEnabled, CategoryAccessManagement, SubjectIDP, "eventDataIdpAccountSyncEnabled",
		`{"vendorName":"GOOGLE","activityId":"A-100"}`))
	var p IDPAccountSync
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if p.VendorName != IDPVendorGoogle || p.ActivityID != "A-100" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadDomainFederationEnabled(t *testing.T) {
	ev := listOne(t, eventJSON(TypeDomainFederationEnabled, CategoryOrganization, SubjectDomain, "eventDataDomainFederationEnabled",
		`{"activityId":"A-1","scimActivityId":"S-1"}`))
	var p DomainFederationEnabled
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if p.ActivityID != "A-1" || p.ScimActivityID != "S-1" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadActivityIDOnlyEvents(t *testing.T) {
	// activityId だけを持つ 3 種（DOMAIN_FEDERATION_DISABLED / BATCH_PASSWORD_RESET_INITIATED / ACCOUNT_PASSWORD_CHANGED）。
	ev := listOne(t, eventJSON(TypeDomainFederationDisabled, CategoryOrganization, SubjectDomain, "eventDataDomainFederationDisabled", `{"activityId":"A-2"}`))
	var d DomainFederationDisabled
	if err := ev.Attributes.Payload(&d); err != nil || d.ActivityID != "A-2" {
		t.Fatalf("DomainFederationDisabled: %+v %v", d, err)
	}
	ev = listOne(t, eventJSON(TypeBatchPasswordResetInitiated, CategoryAccountActivity, SubjectOrganization, "eventDataBatchPasswordResetInitiated", `{"activityId":"A-3"}`))
	var b BatchPasswordResetInitiated
	if err := ev.Attributes.Payload(&b); err != nil || b.ActivityID != "A-3" {
		t.Fatalf("BatchPasswordResetInitiated: %+v %v", b, err)
	}
	ev = listOne(t, eventJSON(TypeAccountPasswordChanged, CategoryAccountActivity, SubjectUser, "eventDataAccountPasswordChanged", `{"activityId":"A-4"}`))
	var c AccountPasswordChanged
	if err := ev.Attributes.Payload(&c); err != nil || c.ActivityID != "A-4" {
		t.Fatalf("AccountPasswordChanged: %+v %v", c, err)
	}
}

func TestPayloadTermsAccepted(t *testing.T) {
	ev := listOne(t, eventJSON(TypeTermsAccepted, CategoryOrganization, SubjectOrganization, "eventDataTermsAccepted", `{"termsName":"Apple Business Manager Agreement"}`))
	var p TermsAccepted
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if p.TermsName != "Apple Business Manager Agreement" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadAppleServicesSettingsUpdated(t *testing.T) {
	ev := listOne(t, eventJSON(TypeAppleServicesSettingsUpdated, CategoryOrganization, SubjectOrganization, "eventDataAppleServicesSettingsUpdated",
		`{"settingsEnabled":["ICLOUD_DRIVE"],"settingsDisabled":["MESSAGES","FACETIME"]}`))
	var p AppleServicesSettingsUpdated
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.SettingsEnabled) != 1 || p.SettingsEnabled[0] != "ICLOUD_DRIVE" || len(p.SettingsDisabled) != 2 {
		t.Fatalf("payload: %+v", p)
	}
}

func TestPayloadEmptyEventData(t *testing.T) {
	// フィールドを持たないイベント（EXTERNAL_ACCOUNT_INVITED / ACCOUNT_IMAGE_*）は eventData が空オブジェクト。
	ev := listOne(t, eventJSON(TypeAccountImageSet, CategoryAccountActivity, SubjectUser, "eventDataAccountImageSet", `{}`))
	if _, ok := ev.Attributes.EventData["eventDataAccountImageSet"]; !ok {
		t.Fatalf("eventData key should be collected: %v", ev.Attributes.EventData)
	}
	var p struct{}
	if err := ev.Attributes.Payload(&p); err != nil {
		t.Fatalf("Payload on empty object: %v", err)
	}
	ev = listOne(t, eventJSON(TypeExternalAccountInvited, CategoryAccountActivity, SubjectInvitation, "eventDataExternalAccountInvited", `{}`))
	if ev.Attributes.Type != TypeExternalAccountInvited || ev.Attributes.SubjectType != SubjectInvitation {
		t.Fatalf("envelope: %+v", ev.Attributes)
	}
}

func TestListRangePassesTypeFilter(t *testing.T) {
	var gotType string
	c := testutil.NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("filter[type]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	q := url.Values{"filter[type]": {TypeIDPCreated}}
	if _, err := New(c).ListRange(context.Background(), time.Now().Add(-time.Hour), time.Now(), q); err != nil {
		t.Fatal(err)
	}
	if gotType != TypeIDPCreated {
		t.Fatalf("filter[type] = %q", gotType)
	}
}

// 50 種の Type* 定数が重複なく揃っていること（DocC の AuditEventType と同数）。
func TestTypeConstantsAreUniqueAndComplete(t *testing.T) {
	all := []string{
		TypeDeviceAddedToOrg, TypeDeviceRemovedFromOrg, TypeDeviceAssignedToServer, TypeDeviceUnassignedFromServer,
		TypeSubjectHasICloudStoragePurchaseAdded, TypeSubjectHasICloudStoragePurchaseRemoved,
		TypeSubjectHasAppleCarePurchaseAdded, TypeSubjectHasAppleCarePurchaseRemoved, TypeDeviceIsErased,
		TypeConfigSettingsCreated, TypeConfigSettingsUpdated, TypeConfigSettingsDeleted,
		TypeCollectionCreated, TypeCollectionUpdated, TypeCollectionDeleted,
		TypeSubscriptionCreated, TypeSubscriptionUpdated, TypeSubscriptionDeleted,
		TypeAccountRoleLocationChanged, TypeAccountAdded, TypeAccountDeleted,
		TypeExternalAccountAssociated, TypeExternalAccountDisassociated,
		TypeDomainAdded, TypeDomainRemoved, TypeDomainVerified,
		TypeAPIAccountCreatedWithKey, TypeAPIAccountCreatedWithoutKey, TypeAPIAccountDeleted,
		TypeAPIAccountKeyRevoked, TypeAPIAccountKeyGenerated, TypeAPIAccountRoleLocationChanged, TypeAPIAccountNameChanged,
		// API 2.6
		TypeRoleCreated, TypeRoleUpdated, TypeTermsAccepted, TypeDomainFederationEnabled, TypeDomainFederationDisabled,
		TypeBatchPasswordResetInitiated, TypeExternalAccountInvited, TypeAccountImageSet, TypeAccountImageReplaced,
		TypeAccountImageRemoved, TypeIDPCreated, TypeIDPUpdated, TypeIDPDeleted, TypeIDPAccountSyncEnabled,
		TypeIDPAccountSyncDisabled, TypeAppleServicesSettingsUpdated, TypeAccountPasswordChanged,
	}
	if len(all) != 50 {
		t.Fatalf("len = %d, want 50", len(all))
	}
	seen := make(map[string]bool, len(all))
	for _, v := range all {
		if v == "" || seen[v] {
			t.Fatalf("empty or duplicate value %q", v)
		}
		seen[v] = true
	}
}
