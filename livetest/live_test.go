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
	read("devices.ListMdmServers", func(ctx context.Context) error { _, e := devices.New(c).ListMdmServers(ctx, small); return e })
	read("devices.ListMdmDevices", func(ctx context.Context) error { _, e := devices.New(c).ListMdmDevices(ctx, small); return e })
	read("people.ListUsers", func(ctx context.Context) error { _, e := people.New(c).ListUsers(ctx, small); return e })
	read("people.ListUserGroups", func(ctx context.Context) error { _, e := people.New(c).ListUserGroups(ctx, small); return e })
	read("orgunits.List", func(ctx context.Context) error { _, e := orgunits.New(c).List(ctx, small); return e })
	read("apps.ListApps", func(ctx context.Context) error { _, e := apps.New(c).ListApps(ctx, small); return e })
	read("apps.ListPackages", func(ctx context.Context) error { _, e := apps.New(c).ListPackages(ctx, small); return e })
	read("configurations.List", func(ctx context.Context) error { _, e := configurations.New(c).List(ctx, small); return e })
	read("blueprints.List", func(ctx context.Context) error { _, e := blueprints.New(c).List(ctx, small); return e })
	read("auditevents.ListRange(24h)", func(ctx context.Context) error {
		_, e := auditevents.New(c).ListRange(ctx, time.Now().Add(-24*time.Hour), time.Now(), small)
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

	t.Run("Replace(apps,[]) → 403 FORBIDDEN", func(t *testing.T) {
		assertForbiddenReplace(t, bp.Replace(ctxFor(t), created.ID, blueprints.RelApps, []string{}))
	})
	t.Run("Replace(configurations,[]) → 403 FORBIDDEN", func(t *testing.T) {
		assertForbiddenReplace(t, bp.Replace(ctxFor(t), created.ID, blueprints.RelConfigurations, []string{}))
	})
	// Member relationships are unconfirmed: record, do not assert.
	t.Run("Replace(orgDevices,[]) → record", func(t *testing.T) {
		logResult(t, "Replace(orgDevices, [])", bp.Replace(ctxFor(t), created.ID, blueprints.RelOrgDevices, []string{}))
	})
	t.Run("Replace(packages,[]) → record", func(t *testing.T) {
		logResult(t, "Replace(packages, [])", bp.Replace(ctxFor(t), created.ID, blueprints.RelPackages, []string{}))
	})
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
