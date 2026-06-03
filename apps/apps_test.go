package apps

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	tok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

func appsHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/apps" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[AppAttributes]{
				Data: []applebusiness.ResourceObject[AppAttributes]{
					{Type: "apps", ID: "A1", Attributes: AppAttributes{Name: "Pages", BundleID: "com.apple.Pages", SupportedOS: []string{OSIOS, OSIPadOS}}},
				},
			})
		case strings.HasPrefix(p, "/v1/apps/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/apps/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[AppAttributes]{
				Data: applebusiness.ResourceObject[AppAttributes]{Type: "apps", ID: id, Attributes: AppAttributes{Name: "Pages", BundleID: "com.apple.Pages"}},
			})
		case p == "/v1/packages" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[PackageAttributes]{
				Data: []applebusiness.ResourceObject[PackageAttributes]{
					{Type: "packages", ID: "P1", Attributes: PackageAttributes{Name: "Bundle A", BundleIDs: []string{"com.x.a", "com.x.b"}}},
				},
			})
		case strings.HasPrefix(p, "/v1/packages/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/packages/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[PackageAttributes]{
				Data: applebusiness.ResourceObject[PackageAttributes]{Type: "packages", ID: id, Attributes: PackageAttributes{Name: "Bundle A"}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestListApps(t *testing.T) {
	c := newClient(t, appsHandler(t))
	got, err := New(c).ListApps(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "A1" || got[0].Attributes.BundleID != "com.apple.Pages" {
		t.Fatalf("unexpected: %+v", got)
	}
	if len(got[0].Attributes.SupportedOS) != 2 || got[0].Attributes.SupportedOS[0] != OSIOS {
		t.Fatalf("supportedOS: %+v", got[0].Attributes.SupportedOS)
	}
}

func TestGetApp(t *testing.T) {
	c := newClient(t, appsHandler(t))
	got, err := New(c).GetApp(context.Background(), "A9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "A9" || got.Attributes.Name != "Pages" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestListPackages(t *testing.T) {
	c := newClient(t, appsHandler(t))
	got, err := New(c).ListPackages(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "P1" || len(got[0].Attributes.BundleIDs) != 2 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetPackage(t *testing.T) {
	c := newClient(t, appsHandler(t))
	got, err := New(c).GetPackage(context.Background(), "P9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "P9" || got.Attributes.Name != "Bundle A" {
		t.Fatalf("unexpected: %+v", got)
	}
}
