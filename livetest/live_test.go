//go:build livetest

// Command: go test -tags livetest ./livetest -v
//
// これは実テナントに対する「応答期待の網羅」検証で、通常の `go test ./...` からは
// build タグ `livetest` で除外される。AXM_* 認証情報が無ければ Skip する。
// 読み取りに加えて Configuration / Blueprint / MDM サーバを作成・削除するため、
// 実組織に書き込む（後始末は t.Cleanup で必ず行う）。
package livetest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/apps"
	"github.com/hitoshiichikawa/apple-business-go/auditevents"
	"github.com/hitoshiichikawa/apple-business-go/blueprints"
	"github.com/hitoshiichikawa/apple-business-go/configurations"
	"github.com/hitoshiichikawa/apple-business-go/devices"
	"github.com/hitoshiichikawa/apple-business-go/orgunits"
	"github.com/hitoshiichikawa/apple-business-go/people"
	"github.com/hitoshiichikawa/apple-business-go/webhooks"
)

var (
	clientOnce sync.Once
	sharedC    *applebusiness.Client
	clientErr  error
)

// mustClient builds one shared client from AXM_* env, or skips when creds are absent.
func mustClient(t *testing.T) *applebusiness.Client {
	t.Helper()
	if os.Getenv("AXM_CLIENT_ID") == "" {
		t.Skip("AXM_CLIENT_ID not set; skipping live verification (set AXM_* to run against a real tenant)")
	}
	clientOnce.Do(func() { sharedC, clientErr = buildClient() })
	if clientErr != nil {
		t.Fatalf("build client: %v", clientErr)
	}
	return sharedC
}

func buildClient() (*applebusiness.Client, error) {
	pem, err := os.ReadFile(os.Getenv("AXM_PRIVATE_KEY_PATH"))
	if err != nil {
		return nil, fmt.Errorf("read private key (AXM_PRIVATE_KEY_PATH): %w", err)
	}
	if os.Getenv("AXM_KEY_ID") == "" {
		return nil, errors.New("AXM_KEY_ID is required")
	}
	opts := []applebusiness.Option{applebusiness.WithUserAgent("apple-business-go/livetest")}
	if v := os.Getenv("AXM_TOKEN_URL"); v != "" {
		opts = append(opts, applebusiness.WithTokenURL(v))
	}
	return applebusiness.NewClient(applebusiness.Config{
		BaseURL: os.Getenv("AXM_BASE_URL"),
		Credentials: applebusiness.Credentials{
			ClientID:   os.Getenv("AXM_CLIENT_ID"),
			TeamID:     os.Getenv("AXM_TEAM_ID"),
			KeyID:      os.Getenv("AXM_KEY_ID"),
			PrivateKey: pem,
			Scope:      os.Getenv("AXM_SCOPE"),
		},
	}, opts...)
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func TestAuth_AccessTokenSucceeds(t *testing.T) {
	c := mustClient(t)
	tok, exp, err := c.AccessToken()
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok == "" {
		t.Fatal("AccessToken returned an empty token")
	}
	if !exp.After(time.Now()) {
		t.Fatalf("token already expired at %s", exp)
	}
}

// ---------------------------------------------------------------------------
// Reads — expect success (or a permission 403, which is logged, not failed)
// ---------------------------------------------------------------------------

func TestReads_SucceedOrPermission403(t *testing.T) {
	c := mustClient(t)
	small := url.Values{"limit": {"1"}}
	read := func(name string, fn func(context.Context) error) {
		t.Run(name, func(t *testing.T) {
			err := fn(ctxFor(t))
			switch {
			case err == nil:
				// ok
			case applebusiness.IsForbidden(err):
				t.Logf("%s: 403 (permission) — %v", name, err)
			default:
				t.Fatalf("%s: %v", name, err)
			}
		})
	}
	read("devices.List", func(ctx context.Context) error { _, e := devices.New(c).List(ctx, small); return e })
	// Activation Lock ステータス（API 2.5、2026-09-15 追加）。観測値は -v で記録する。
	t.Run("devices.ActivationLockStatus(first device)", func(t *testing.T) {
		ctx := ctxFor(t)
		list, err := devices.New(c).List(ctx, small)
		switch {
		case applebusiness.IsForbidden(err):
			t.Logf("devices.List: 403 (permission) — %v", err)
			return
		case err != nil:
			t.Fatalf("devices.List: %v", err)
		case len(list) == 0:
			t.Skip("no devices in the organization")
		}
		st, err := devices.New(c).ActivationLockStatus(ctx, list[0].ID)
		switch {
		case err == nil:
			t.Logf("activationLockStatus(%s): isLocked=%v lockType=%q", list[0].ID, st.Attributes.IsLocked, st.Attributes.LockType)
		case applebusiness.IsForbidden(err):
			t.Logf("devices.ActivationLockStatus: 403 (permission) — %v", err)
		default:
			t.Fatalf("devices.ActivationLockStatus: %v", err)
		}
	})
	// orgDevices の絞り込み（API 2.7、2026-10-29 追加）。各 filter[...] を渡して 200 になり、
	// 返った端末（1 ページ目、最大 100 件）が条件を満たすことを確かめる。全件は辿らない。
	t.Run("devices.ListFiltered(API 2.7)", func(t *testing.T) {
		ctx := ctxFor(t)
		firstPage := func(f devices.DeviceFilter) ([]devices.Device, error) {
			var out []devices.Device
			for d, err := range applebusiness.ListSeq[devices.DeviceAttributes](ctx, c, "/v1/orgDevices", f.Query(url.Values{"limit": {"100"}})) {
				if err != nil {
					return out, err
				}
				out = append(out, d)
				if len(out) >= 100 {
					break
				}
			}
			return out, nil
		}
		seed, err := firstPage(devices.DeviceFilter{})
		switch {
		case applebusiness.IsForbidden(err):
			t.Logf("devices.List: 403 (permission) — %v", err)
			return
		case err != nil:
			t.Fatalf("devices.List: %v", err)
		case len(seed) == 0:
			t.Skip("no devices in the organization")
		}
		first := seed[0].Attributes

		for _, st := range []string{devices.DeviceStatusAssigned, devices.DeviceStatusUnassigned} {
			got, err := firstPage(devices.DeviceFilter{Statuses: []string{st}})
			if err != nil {
				t.Fatalf("filter[status]=%s: %v", st, err)
			}
			for _, d := range got {
				if d.Attributes.Status != st {
					t.Errorf("filter[status]=%s returned %s with status %q", st, d.ID, d.Attributes.Status)
				}
			}
			t.Logf("filter[status]=%s: %d device(s) on the first page", st, len(got))
		}

		got, err := firstPage(devices.DeviceFilter{SerialNumbers: []string{first.SerialNumber}})
		if err != nil {
			t.Fatalf("filter[serialNumber]: %v", err)
		}
		if len(got) != 1 || got[0].Attributes.SerialNumber != first.SerialNumber {
			ids := make([]string, 0, len(got))
			for _, d := range got {
				ids = append(ids, d.ID)
			}
			t.Errorf("filter[serialNumber]=%s returned %d device(s): %v", first.SerialNumber, len(got), ids)
		}

		// 複数値はカンマ区切り（%2C にエンコードされる）の 1 パラメータで送る。2 台を指定して 2 台返ること。
		if len(seed) >= 2 {
			two := []string{seed[0].Attributes.SerialNumber, seed[1].Attributes.SerialNumber}
			got, err = firstPage(devices.DeviceFilter{SerialNumbers: two})
			if err != nil {
				t.Fatalf("filter[serialNumber] (2 values): %v", err)
			}
			if len(got) != 2 {
				t.Errorf("filter[serialNumber] with 2 comma-separated values returned %d device(s)", len(got))
			} else {
				t.Logf("filter[serialNumber] with 2 comma-separated values: 2 device(s)")
			}
		}

		got, err = firstPage(devices.DeviceFilter{ProductFamilies: []string{first.ProductFamily}})
		if err != nil {
			t.Fatalf("filter[productFamily]=%s: %v", first.ProductFamily, err)
		}
		for _, d := range got {
			if d.Attributes.ProductFamily != first.ProductFamily {
				t.Errorf("filter[productFamily]=%s returned %s with productFamily %q", first.ProductFamily, d.ID, d.Attributes.ProductFamily)
			}
		}
		t.Logf("filter[productFamily]=%s: %d device(s) on the first page", first.ProductFamily, len(got))

		// 1 台目と別のファミリでも、返った端末がすべてそのファミリであること。
		// Apple がフィルタを無視して全件を返していれば、ここで 1 台目のファミリが混ざって失敗する。
		other := devices.ProductFamilyMac
		if first.ProductFamily == devices.ProductFamilyMac {
			other = devices.ProductFamilyIPhone
		}
		got, err = firstPage(devices.DeviceFilter{ProductFamilies: []string{other}})
		if err != nil {
			t.Fatalf("filter[productFamily]=%s: %v", other, err)
		}
		for _, d := range got {
			if d.Attributes.ProductFamily != other {
				t.Errorf("filter[productFamily]=%s returned %s with productFamily %q", other, d.ID, d.Attributes.ProductFamily)
			}
		}
		t.Logf("filter[productFamily]=%s: %d device(s) on the first page", other, len(got))

		// 未来の日時以降に追加された端末は無いはず（範囲指定が無視されていれば全件が返る）。
		future := time.Now().AddDate(10, 0, 0).Truncate(time.Second)
		got, err = firstPage(devices.DeviceFilter{AddedToOrgDateTime: devices.TimeFilter{GTE: future}})
		if err != nil {
			t.Fatalf("filter[addedToOrgDateTime][GTE] (future): %v", err)
		}
		if len(got) != 0 {
			t.Errorf("filter[addedToOrgDateTime][GTE]=%s (future) returned %d device(s)", future.Format(time.RFC3339), len(got))
		}

		added, perr := time.Parse(time.RFC3339, first.AddedToOrgDateTime)
		if perr != nil {
			t.Logf("skip filter[addedToOrgDateTime][GTE]: cannot parse %q: %v", first.AddedToOrgDateTime, perr)
			return
		}
		gte := added.Truncate(time.Second) // フィルタは秒精度で送る
		got, err = firstPage(devices.DeviceFilter{AddedToOrgDateTime: devices.TimeFilter{GTE: gte}})
		if err != nil {
			t.Fatalf("filter[addedToOrgDateTime][GTE]: %v", err)
		}
		for _, d := range got {
			if at, perr := time.Parse(time.RFC3339, d.Attributes.AddedToOrgDateTime); perr == nil && at.Before(gte) {
				t.Errorf("filter[addedToOrgDateTime][GTE]=%s returned %s added at %s", gte.Format(time.RFC3339), d.ID, d.Attributes.AddedToOrgDateTime)
			}
		}
		t.Logf("filter[addedToOrgDateTime][GTE]=%s: %d device(s) on the first page", gte.Format(time.RFC3339), len(got))
	})
	read("devices.ListMdmServers", func(ctx context.Context) error { _, e := devices.New(c).ListMdmServers(ctx, small); return e })
	read("devices.ListMdmDevices", func(ctx context.Context) error { _, e := devices.New(c).ListMdmDevices(ctx, small); return e })
	read("people.ListUsers", func(ctx context.Context) error { _, e := people.New(c).ListUsers(ctx, small); return e })
	read("people.ListUserGroups", func(ctx context.Context) error { _, e := people.New(c).ListUserGroups(ctx, small); return e })
	read("orgunits.List", func(ctx context.Context) error { _, e := orgunits.New(c).List(ctx, small); return e })
	read("apps.ListApps", func(ctx context.Context) error { _, e := apps.New(c).ListApps(ctx, small); return e })
	read("apps.ListPackages", func(ctx context.Context) error { _, e := apps.New(c).ListPackages(ctx, small); return e })
	read("configurations.List", func(ctx context.Context) error { _, e := configurations.New(c).List(ctx, small); return e })
	read("blueprints.List", func(ctx context.Context) error { _, e := blueprints.New(c).List(ctx, small); return e })
	// ListRange は links.next を最後まで辿るため、limit=1 だと 24h 分のイベント件数ぶん
	// リクエストが発生し、429（Retry-After 60s）→ 90s のタイムアウトに至る（demo 2026-10-14 で観測）。
	// 上限の limit=1000 にして通常 1 リクエストで収める。
	auditQ := url.Values{"limit": {"1000"}}
	read("auditevents.ListRange(24h)", func(ctx context.Context) error {
		_, e := auditevents.New(c).ListRange(ctx, time.Now().Add(-24*time.Hour), time.Now(), auditQ)
		return e
	})
	// API 2.6 で追加されたイベント種別を filter[type] に渡しても 400 にならないこと。
	// 終了日時をゼロ値にしても 400 にならないこと（#63。ListRange が現在時刻を送る）。
	read("auditevents.ListRange(24h, zero end)", func(ctx context.Context) error {
		_, e := auditevents.New(c).ListRange(ctx, time.Now().Add(-24*time.Hour), time.Time{}, auditQ)
		return e
	})
	read("auditevents.ListRange(24h, filter[type]=ROLE_UPDATED)", func(ctx context.Context) error {
		q := url.Values{"limit": {"1000"}, "filter[type]": {auditevents.TypeRoleUpdated}}
		_, e := auditevents.New(c).ListRange(ctx, time.Now().Add(-24*time.Hour), time.Now(), q)
		return e
	})
}

// ---------------------------------------------------------------------------
// 404 — a well-formed but nonexistent id
// ---------------------------------------------------------------------------

func TestGet_BogusID_IsNotFound(t *testing.T) {
	c := mustClient(t)
	t.Run("configurations.Get bogus → 404", func(t *testing.T) {
		_, err := configurations.New(c).Get(ctxFor(t), "00000000-0000-4000-8000-000000000000")
		if !applebusiness.IsNotFound(err) {
			t.Fatalf("want 404 IsNotFound, got %v", err)
		}
	})
	t.Run("blueprints.Get bogus → 404", func(t *testing.T) {
		_, err := blueprints.New(c).Get(ctxFor(t), "livetest-nonexistent-0000")
		if !applebusiness.IsNotFound(err) {
			t.Fatalf("want 404 IsNotFound, got %v", err)
		}
	})
	// 📄 DocC: 組織が管理していないシリアルは 404。
	t.Run("devices.ActivationLockStatus bogus → 404", func(t *testing.T) {
		_, err := devices.New(c).ActivationLockStatus(ctxFor(t), "LIVETEST0000")
		if !applebusiness.IsNotFound(err) {
			t.Fatalf("want 404 IsNotFound, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Blueprint create validation — members and resources are both required
// ---------------------------------------------------------------------------

func TestBlueprintCreate_Validation(t *testing.T) {
	c := mustClient(t)
	bp := blueprints.New(c)
	cfgID := createTestConfig(t, c)
	groupID := firstUserGroup(t, c)

	t.Run("no members → 409 MISSING_MEMBERS", func(t *testing.T) {
		expectCreateConflict(t, bp, blueprints.CreateInput{
			Name:           testName(),
			Configurations: []string{cfgID},
		}, "MISSING_MEMBERS")
	})

	t.Run("no resources → 409 MISSING_RESOURCES", func(t *testing.T) {
		if groupID == "" {
			t.Skip("no user group available to supply a member")
		}
		expectCreateConflict(t, bp, blueprints.CreateInput{
			Name:       testName(),
			UserGroups: []string{groupID},
		}, "MISSING_RESOURCES")
	})
}

// ---------------------------------------------------------------------------
// Blueprint relationships — REPLACE(PATCH) is rejected; AddTo/RemoveFrom work
// ---------------------------------------------------------------------------

func TestBlueprintRelationships(t *testing.T) {
	c := mustClient(t)
	bp := blueprints.New(c)
	cfgID := createTestConfig(t, c)
	groupID := firstUserGroup(t, c)
	if groupID == "" {
		t.Skip("no user group available to satisfy the member requirement")
	}

	created, err := bp.Create(ctxFor(t), blueprints.CreateInput{
		Name:           testName(),
		Configurations: []string{cfgID},
		UserGroups:     []string{groupID},
	})
	if err != nil {
		t.Fatalf("create valid blueprint: %v", err)
	}
	t.Cleanup(func() { bestEffortDelete(func(ctx context.Context) error { return bp.Delete(ctx, created.ID) }) })

	appID := firstApp(t, c)

	t.Run("AddTo/Relationship/RemoveFrom apps", func(t *testing.T) {
		if appID == "" {
			t.Skip("no app available in this tenant")
		}
		ctx := ctxFor(t)
		if err := bp.AddTo(ctx, created.ID, blueprints.RelApps, []string{appID}); err != nil {
			t.Fatalf("AddTo(apps): %v", err)
		}
		ids, err := bp.RelationshipIDs(ctx, created.ID, blueprints.RelApps)
		if err != nil {
			t.Fatalf("RelationshipIDs(apps): %v", err)
		}
		if !containsID(ids, appID) {
			t.Errorf("apps relationship %v does not contain %s after AddTo", dataIDs(ids), appID)
		}
		if err := bp.RemoveFrom(ctx, created.ID, blueprints.RelApps, []string{appID}); err != nil {
			t.Errorf("RemoveFrom(apps): %v", err)
		}
	})

	// REPLACE(PATCH) is verified with a low-level raw request: the SDK no longer
	// exposes blueprints.Replace (removed in v0.11.0, #44 — Apple rejects REPLACE
	// on every Blueprint relationship). This keeps the 403 contract under test so
	// a future Apple change is caught.
	t.Run("PATCH apps [] → 403 FORBIDDEN", func(t *testing.T) {
		assertForbiddenReplace(t, rawReplace(t, c, created.ID, blueprints.RelApps))
	})
	t.Run("PATCH configurations [] → 403 FORBIDDEN", func(t *testing.T) {
		assertForbiddenReplace(t, rawReplace(t, c, created.ID, blueprints.RelConfigurations))
	})
	t.Run("PATCH orgDevices [] → record", func(t *testing.T) {
		logResult(t, "raw PATCH orgDevices []", rawReplace(t, c, created.ID, blueprints.RelOrgDevices))
	})
	t.Run("PATCH packages [] → record", func(t *testing.T) {
		logResult(t, "raw PATCH packages []", rawReplace(t, c, created.ID, blueprints.RelPackages))
	})
}

// rawReplace sends a raw PATCH {"data":[]} to a Blueprint relationship via the
// low-level client, since the SDK no longer exposes Replace (#44).
func rawReplace(t *testing.T, c *applebusiness.Client, bpID, rel string) error {
	t.Helper()
	u := c.BaseURL() + "/v1/blueprints/" + url.PathEscape(bpID) + "/relationships/" + rel
	return c.Do(ctxFor(t), http.MethodPatch, u, []byte(`{"data":[]}`), nil)
}

// ---------------------------------------------------------------------------
// MDM server certificate algorithm — EC rejected, RSA accepted
// ---------------------------------------------------------------------------

func TestMdmServer_CertAlgorithm(t *testing.T) {
	c := mustClient(t)
	dev := devices.New(c)

	t.Run("EC cert → 400 only RSA", func(t *testing.T) {
		name := testName() + "-ec"
		srv, err := dev.CreateMdmServer(ctxFor(t), devices.CreateMdmServerInput{
			ServerName:        name,
			ServerCertificate: devices.MdmServerCertificate{Name: name + ".cer", Data: selfSignedCertEC(name)},
		})
		if err == nil {
			bestEffortDelete(func(ctx context.Context) error { return dev.DeleteMdmServer(ctx, srv.ID) })
			t.Fatalf("EC certificate unexpectedly accepted (server %s)", srv.ID)
		}
		var ae *applebusiness.APIError
		if !errors.As(err, &ae) || ae.StatusCode != 400 {
			t.Fatalf("want 400 APIError for EC cert, got %v", err)
		}
		if len(ae.Errors) > 0 && !strings.Contains(ae.Errors[0].Code, "PARAMETER_ERROR") {
			t.Errorf("want PARAMETER_ERROR code, got %q (detail=%q)", ae.Errors[0].Code, ae.Errors[0].Detail)
		}
	})

	t.Run("RSA cert → Create/Get/Update/Delete", func(t *testing.T) {
		name := testName() + "-rsa"
		ctx := ctxFor(t)
		srv, err := dev.CreateMdmServer(ctx, devices.CreateMdmServerInput{
			ServerName:        name,
			ServerCertificate: devices.MdmServerCertificate{Name: name + ".cer", Data: selfSignedCertRSA(name)},
		})
		if err != nil {
			t.Fatalf("create RSA MDM server: %v", err)
		}
		// Safety net if the explicit Delete below is not reached.
		t.Cleanup(func() { bestEffortDelete(func(ctx context.Context) error { return dev.DeleteMdmServer(ctx, srv.ID) }) })

		if _, err := dev.GetMdmServer(ctx, srv.ID); err != nil {
			t.Errorf("GetMdmServer: %v", err)
		}
		newName := name + "-upd"
		if _, err := dev.UpdateMdmServer(ctx, srv.ID, devices.UpdateMdmServerInput{ServerName: &newName}); err != nil {
			t.Errorf("UpdateMdmServer: %v", err)
		}
		if err := dev.DeleteMdmServer(ctx, srv.ID); err != nil {
			t.Errorf("DeleteMdmServer: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Webhooks (API 2.8) — テスト用の Webhook を作成し、最後に必ず削除する
// ---------------------------------------------------------------------------

// TestWebhook_Lifecycle は作成→取得→更新→ping→配信一覧→再配信→削除を確認し、
// 作成・更新・削除の監査イベントを記録する。URL は名前解決できない .invalid ドメインに
// するため通知は外部に届かず、ping の配信は FAILED（errorMessage 付き）になる想定。
// 作成時に URL が検証されるかもここで観測する。
func TestWebhook_Lifecycle(t *testing.T) {
	c := mustClient(t)
	svc := webhooks.New(c)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(cancel)

	name := testName()
	start := time.Now().Add(-time.Minute)
	wh, err := svc.Create(ctx, webhooks.CreateInput{
		Name:       name,
		URL:        "https://apple-business-go-livetest.invalid/webhook",
		Secret:     randHex(32),
		EventTypes: []string{webhooks.EventTypeOrgDeviceAttributeChanged},
		Enabled:    true,
	})
	if applebusiness.IsForbidden(err) {
		t.Skipf("webhooks.Create: 403 (permission) — %v", err)
	}
	if err != nil {
		t.Fatalf("webhooks.Create: %v", err)
	}
	// 下の明示的な Delete に到達しなかった場合の保険。
	t.Cleanup(func() { bestEffortDelete(func(ctx context.Context) error { return svc.Delete(ctx, wh.ID) }) })
	t.Logf("created webhook %s (enabled=%v)", wh.ID, wh.Attributes.Enabled)

	got, err := svc.Get(ctx, wh.ID)
	if err != nil {
		t.Fatalf("webhooks.Get: %v", err)
	}
	if got.Attributes.Name != name || !got.Attributes.Enabled || len(got.Attributes.EventTypes) != 1 ||
		got.Attributes.EventTypes[0] != webhooks.EventTypeOrgDeviceAttributeChanged {
		t.Errorf("Get returned %+v", got.Attributes)
	}

	newName := name + "-upd"
	up, err := svc.Update(ctx, wh.ID, webhooks.UpdateInput{Name: &newName})
	if err != nil {
		t.Fatalf("webhooks.Update: %v", err)
	}
	if up.Attributes.Name != newName || !up.Attributes.Enabled {
		t.Errorf("Update returned %+v", up.Attributes)
	}

	pingID, err := svc.Ping(ctx, wh.ID)
	if err != nil {
		t.Fatalf("webhooks.Ping: %v", err)
	}
	t.Logf("ping %s sent", pingID)

	// 配信が記録され、すべて PENDING 以外になるまで最大 60 秒待つ。作成日時の範囲は API で必須で、
	// 未来の日時は 409 になるため終わりは現在時刻（ゼロ値）にする。時計のずれで直近の配信を
	// 取りこぼしても、繰り返し取得するうちに拾える。
	var list *webhooks.DeliveryList
	for deadline := time.Now().Add(60 * time.Second); ; {
		list, err = svc.Deliveries(ctx, wh.ID, start, time.Time{}, webhooks.DeliveryFilter{IncludeEvents: true}, nil)
		if err != nil {
			t.Fatalf("webhooks.Deliveries: %v", err)
		}
		if deliveriesSettled(list) || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Second)
	}
	for _, d := range list.Deliveries {
		ev, included := list.Event(d)
		status := 0
		if d.Attributes.Response != nil {
			status = d.Attributes.Response.HTTPStatusCode
		}
		t.Logf("delivery %s: state=%s redelivery=%v httpStatus=%d error=%q event=%s (included=%v isPing=%v)",
			d.ID, d.Attributes.DeliveryState, d.Attributes.Redelivery, status, d.Attributes.ErrorMessage,
			d.EventID(), included, ev.Attributes.IsPingEvent)
		if included && ev.Attributes.IsPingEvent {
			// 配信一覧のイベントの payload は通知本文そのもので、ParseNotification で読めるはず。
			n, err := webhooks.ParseNotification([]byte(ev.Attributes.Payload))
			if err != nil {
				t.Errorf("ParseNotification(event %s payload): %v (payload=%s)", ev.ID, err, ev.Attributes.Payload)
			} else {
				t.Logf("ping payload parsed: type=%s id=%q version=%d isPing=%v (raw payload: %s)", n.Type, n.ID, n.Version, n.IsPing(), strings.Join(strings.Fields(ev.Attributes.Payload), " "))
				if !n.IsPing() {
					t.Errorf("ping payload type = %q, want %q", n.Type, webhooks.NotificationWebhookPingCreated)
				}
			}
		}
	}
	if len(list.Deliveries) == 0 {
		t.Errorf("no delivery was recorded for the ping within 60s")
	} else {
		rd, err := svc.Redeliver(ctx, list.Deliveries[0].ID)
		if err != nil {
			t.Errorf("webhooks.Redeliver: %v", err)
		} else {
			t.Logf("redelivery %s: state=%s redelivery=%v", rd.ID, rd.Attributes.DeliveryState, rd.Attributes.Redelivery)
		}
	}

	if err := svc.Delete(ctx, wh.ID); err != nil {
		t.Fatalf("webhooks.Delete: %v", err)
	}
	if _, err := svc.Get(ctx, wh.ID); !applebusiness.IsNotFound(err) {
		t.Errorf("Get after Delete: want 404, got %v", err)
	}

	logWebhookAuditEvents(ctx, t, c, start, wh.ID, name)
}

// deliveriesSettled は配信が 1 件以上あり、すべて PENDING 以外になったかを返す。
func deliveriesSettled(l *webhooks.DeliveryList) bool {
	if len(l.Deliveries) == 0 {
		return false
	}
	for _, d := range l.Deliveries {
		if d.Attributes.DeliveryState == webhooks.DeliveryStatePending {
			return false
		}
	}
	return true
}

// logWebhookAuditEvents は対象 Webhook の WEBHOOK_* 監査イベントを探し、eventData の生 JSON を
// 記録する。DocC では AuditEventWebhookCreated / Updated の enabled が string 型なので、実際の型を
// ここで確かめる（auditevents.Webhook に decode できなければ失敗にする）。監査イベントは反映に
// 時間がかかることがあるため、見つからないこと自体は失敗にしない。
func logWebhookAuditEvents(ctx context.Context, t *testing.T, c *applebusiness.Client, start time.Time, webhookID, namePrefix string) {
	t.Helper()
	found := map[string]bool{
		auditevents.TypeWebhookCreated: false,
		auditevents.TypeWebhookUpdated: false,
		auditevents.TypeWebhookDeleted: false,
	}
	for deadline := time.Now().Add(90 * time.Second); ; {
		// filter[endTimestamp] も実機では必須（#63）。ListRange はゼロ値なら現在時刻を送るが、ここでは明示する。
		events, err := auditevents.New(c).ListRange(ctx, start, time.Now(), url.Values{"limit": {"1000"}})
		if err != nil {
			t.Logf("auditevents.ListRange: %v", err)
			return
		}
		for _, ev := range events {
			a := ev.Attributes
			done, isWebhook := found[a.Type]
			if !isWebhook || done || (a.SubjectID != webhookID && !strings.HasPrefix(a.SubjectName, namePrefix)) {
				continue
			}
			found[a.Type] = true
			for k, raw := range a.EventData {
				t.Logf("audit %s (category=%s subjectType=%s subjectId=%s): %s=%s", a.Type, a.Category, a.SubjectType, a.SubjectID, k, raw)
			}
			if a.Type != auditevents.TypeWebhookDeleted {
				var w auditevents.Webhook
				if err := a.Payload(&w); err != nil {
					t.Errorf("decode %s payload into auditevents.Webhook: %v", a.Type, err)
				}
			}
		}
		if (found[auditevents.TypeWebhookCreated] && found[auditevents.TypeWebhookUpdated] && found[auditevents.TypeWebhookDeleted]) ||
			time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Second)
	}
	t.Logf("WEBHOOK_* audit events found for %s: %v", webhookID, found)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ctxFor(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// bestEffortDelete runs a delete with its own short-lived context, ignoring the
// result. Used from t.Cleanup, where the test's context is already cancelled.
func bestEffortDelete(del func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = del(ctx)
}

// createTestConfig creates a harmless web-clip Configuration and registers its
// deletion. It is the "resource" a Blueprint needs.
func createTestConfig(t *testing.T, c *applebusiness.Client) string {
	t.Helper()
	cfg := configurations.New(c)
	name := testName()
	created, err := cfg.Create(ctxFor(t), configurations.CreateInput{
		Name:                   name,
		ConfiguredForPlatforms: []string{configurations.PlatformIOS},
		ConfigurationProfile:   sampleMobileconfig(name),
		Filename:               name + ".mobileconfig",
	})
	if err != nil {
		t.Fatalf("create test configuration: %v", err)
	}
	t.Cleanup(func() { bestEffortDelete(func(ctx context.Context) error { return cfg.Delete(ctx, created.ID) }) })
	return created.ID
}

func firstUserGroup(t *testing.T, c *applebusiness.Client) string {
	t.Helper()
	gs, err := people.New(c).ListUserGroups(ctxFor(t), url.Values{"limit": {"1"}})
	if err != nil {
		t.Logf("list user groups (for member discovery): %v", err)
		return ""
	}
	if len(gs) == 0 {
		return ""
	}
	return gs[0].ID
}

func firstApp(t *testing.T, c *applebusiness.Client) string {
	t.Helper()
	as, err := apps.New(c).ListApps(ctxFor(t), url.Values{"limit": {"1"}})
	if err != nil {
		t.Logf("list apps (for apps relationship): %v", err)
		return ""
	}
	if len(as) == 0 {
		return ""
	}
	return as[0].ID
}

func expectCreateConflict(t *testing.T, bp *blueprints.Service, in blueprints.CreateInput, codeSubstr string) {
	t.Helper()
	created, err := bp.Create(ctxFor(t), in)
	if err == nil {
		bestEffortDelete(func(ctx context.Context) error { return bp.Delete(ctx, created.ID) })
		t.Fatalf("Create unexpectedly succeeded (id=%s); want 409 %s", created.ID, codeSubstr)
	}
	if !applebusiness.IsConflict(err) {
		t.Fatalf("want 409 conflict (%s), got %v", codeSubstr, err)
	}
	var ae *applebusiness.APIError
	if errors.As(err, &ae) && len(ae.Errors) > 0 {
		if !strings.Contains(ae.Errors[0].Code, codeSubstr) {
			t.Errorf("want error code containing %q, got %q (detail=%q)", codeSubstr, ae.Errors[0].Code, ae.Errors[0].Detail)
		}
		return
	}
	t.Errorf("409 but no JSON:API error code to check for %q: %v", codeSubstr, err)
}

func assertForbiddenReplace(t *testing.T, err error) {
	t.Helper()
	if !applebusiness.IsForbidden(err) {
		t.Fatalf("want 403 IsForbidden, got %v", err)
	}
	var ae *applebusiness.APIError
	if errors.As(err, &ae) && len(ae.Errors) > 0 {
		if !strings.Contains(ae.Errors[0].Code, "FORBIDDEN") {
			t.Errorf("want a FORBIDDEN code, got %q", ae.Errors[0].Code)
		}
		if !strings.Contains(ae.Errors[0].Detail, "REPLACE") {
			t.Errorf("want detail mentioning REPLACE, got %q", ae.Errors[0].Detail)
		}
	}
}

func logResult(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Logf("%s → success (no error)", label)
		return
	}
	var ae *applebusiness.APIError
	if errors.As(err, &ae) {
		code := ""
		if len(ae.Errors) > 0 {
			code = ae.Errors[0].Code
		}
		t.Logf("%s → APIError status=%d code=%s", label, ae.StatusCode, code)
		return
	}
	t.Logf("%s → %v", label, err)
}

func containsID(ds []applebusiness.Data, id string) bool {
	for _, d := range ds {
		if d.ID == id {
			return true
		}
	}
	return false
}

func dataIDs(ds []applebusiness.Data) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.ID
	}
	return out
}

func testName() string {
	return fmt.Sprintf("livetest-%s-%s", time.Now().Format("20060102-150405"), randHex(3))
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// selfSignedCertRSA / selfSignedCertEC build a self-signed cert (base64 DER).
// The key is discarded; these only exercise the CreateMdmServer validation.
func selfSignedCertRSA(cn string) string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return certBase64(cn, &key.PublicKey, key)
}

func selfSignedCertEC(cn string) string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return certBase64(cn, &key.PublicKey, key)
}

func certBase64(cn string, pub, priv any) string {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(der)
}

func sampleMobileconfig(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadIdentifier</key><string>com.example.abmgo.` + name + `</string>
  <key>PayloadUUID</key><string>` + randUUID() + `</string>
  <key>PayloadDisplayName</key><string>` + name + `</string>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.webClip.managed</string>
      <key>PayloadVersion</key><integer>1</integer>
      <key>PayloadIdentifier</key><string>com.example.abmgo.` + name + `.webclip</string>
      <key>PayloadUUID</key><string>` + randUUID() + `</string>
      <key>PayloadDisplayName</key><string>abm-go livetest</string>
      <key>URL</key><string>https://example.com</string>
      <key>Label</key><string>abm-go livetest</string>
    </dict>
  </array>
</dict>
</plist>`
}

func randUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
