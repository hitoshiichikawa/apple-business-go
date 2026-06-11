// Package testutil provides shared scaffolding for service-package tests:
// a fresh ES256 test key, an in-process fake token endpoint, and a Client
// wired to a fake API server. It is internal to the module and not part of
// the public SDK surface.
package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// KeyPEM returns a freshly generated EC P-256 private key (for ES256) in
// SEC1 PEM form. Keys are generated per test and never persisted.
func KeyPEM(t testing.TB) []byte {
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

// NewClient starts a fake API server serving h and a fake token endpoint,
// and returns a Client pointing at both. Servers are closed via t.Cleanup.
func NewClient(t testing.TB, h http.Handler, opts ...applebusiness.Option) *applebusiness.Client {
	t.Helper()
	api := httptest.NewServer(h)
	t.Cleanup(api.Close)
	tok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(tok.Close)
	all := append([]applebusiness.Option{
		applebusiness.WithBaseURL(api.URL),
		applebusiness.WithTokenURL(tok.URL),
	}, opts...)
	c, err := applebusiness.NewClient(applebusiness.Config{Credentials: applebusiness.Credentials{
		ClientID:   "BUSINESSAPI.test",
		TeamID:     "BUSINESSAPI.test",
		KeyID:      "kid",
		PrivateKey: KeyPEM(t),
	}}, all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(t testing.TB, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}
