package applebusiness

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"golang.org/x/oauth2"
)

// failOnHitTokenServer returns a fake token endpoint that records how many times
// it was called. When an injected token source is used, this count must stay 0.
func failOnHitTokenServer(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"endpoint-token","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(s.Close)
	return s
}

// TestWithTokenSource_UsesInjectedTokenAndSkipsEndpoint verifies that when a
// caller injects an oauth2.TokenSource, API requests carry that source's token
// and the SDK never calls the OAuth token endpoint.
func TestWithTokenSource_UsesInjectedTokenAndSkipsEndpoint(t *testing.T) {
	var tokenHits int32
	tok := failOnHitTokenServer(t, &tokenHits)

	var gotAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "x"}},
		})
	}))
	t.Cleanup(api.Close)

	injected := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "injected-token", TokenType: "Bearer"})
	c, err := NewClient(
		Config{BaseURL: api.URL},
		WithTokenURL(tok.URL), // present but must NOT be used
		WithTokenSource(injected),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := Get[testAttrs](context.Background(), c, "/v1/things/1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer injected-token" {
		t.Fatalf("Authorization = %q, want Bearer injected-token", gotAuth)
	}
	if n := atomic.LoadInt32(&tokenHits); n != 0 {
		t.Fatalf("token endpoint hit %d times, want 0 (injected source must be used)", n)
	}
}

// TestWithTokenSource_NoCredentialsRequired verifies that injecting a token
// source makes Config.Credentials optional, while the credential-less path
// without a token source still errors (backward-compatible requirement).
func TestWithTokenSource_NoCredentialsRequired(t *testing.T) {
	injected := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "t", TokenType: "Bearer"})
	if _, err := NewClient(Config{BaseURL: "https://example.test"}, WithTokenSource(injected)); err != nil {
		t.Fatalf("NewClient with token source and no credentials: unexpected error: %v", err)
	}

	if _, err := NewClient(Config{BaseURL: "https://example.test"}); err == nil {
		t.Fatal("NewClient without credentials and without token source: expected error, got nil")
	}
}

// TestWithTokenSource_AccessTokenReturnsInjected verifies AccessToken delegates
// to the injected source.
func TestWithTokenSource_AccessTokenReturnsInjected(t *testing.T) {
	injected := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "injected-token", TokenType: "Bearer"})
	c, err := NewClient(Config{BaseURL: "https://example.test"}, WithTokenSource(injected))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, _, err := c.AccessToken()
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if got != "injected-token" {
		t.Fatalf("AccessToken = %q, want injected-token", got)
	}
}

// TestWithTokenSource_WinsOverCredentials verifies that when both Credentials
// and an injected token source are supplied, the injected source is used and the
// token endpoint is never called.
func TestWithTokenSource_WinsOverCredentials(t *testing.T) {
	var tokenHits int32
	tok := failOnHitTokenServer(t, &tokenHits)

	var gotAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "x"}},
		})
	}))
	t.Cleanup(api.Close)

	injected := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "injected-token", TokenType: "Bearer"})
	c, err := NewClient(
		Config{
			BaseURL: api.URL,
			Credentials: Credentials{
				ClientID:   "BUSINESSAPI.test",
				TeamID:     "BUSINESSAPI.test",
				KeyID:      "test-kid",
				PrivateKey: testKeyPEM(t),
			},
		},
		WithTokenURL(tok.URL),
		WithTokenSource(injected),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := Get[testAttrs](context.Background(), c, "/v1/things/1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer injected-token" {
		t.Fatalf("Authorization = %q, want Bearer injected-token (injected source must win)", gotAuth)
	}
	if n := atomic.LoadInt32(&tokenHits); n != 0 {
		t.Fatalf("token endpoint hit %d times, want 0 (injected source must win over credentials)", n)
	}
}
