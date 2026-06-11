package applebusiness

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type testAttrs struct {
	Name string `json:"name"`
}

// testKeyPEM は ES256 用の EC P-256 秘密鍵を SEC1 PEM で生成する。
func testKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

func writeJSONResp(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

// startTokenServer は固定トークンを返す擬似 OAuth 端点。capture!=nil なら受信フォームを格納。
func startTokenServer(t *testing.T, capture *url.Values) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if capture != nil {
			*capture = r.PostForm
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(s.Close)
	return s
}

func newTestClient(t *testing.T, apiURL, tokURL string, opts ...Option) *Client {
	t.Helper()
	all := []Option{WithBaseURL(apiURL), WithTokenURL(tokURL)}
	all = append(all, opts...)
	c, err := NewClient(Config{Credentials: Credentials{
		ClientID:   "BUSINESSAPI.test",
		TeamID:     "BUSINESSAPI.test",
		KeyID:      "test-kid",
		PrivateKey: testKeyPEM(t),
	}}, all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestTokenExchangeAndGet(t *testing.T) {
	var form url.Values
	tok := startTokenServer(t, &form)

	var gotAuth, gotPath string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "42", Attributes: testAttrs{Name: "hello"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	obj, err := Get[testAttrs](context.Background(), c, "/v1/things/42")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if obj.ID != "42" || obj.Attributes.Name != "hello" {
		t.Fatalf("unexpected object: %+v", obj)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want Bearer test-token", gotAuth)
	}
	if gotPath != "/v1/things/42" {
		t.Fatalf("path = %q", gotPath)
	}
	if form.Get("grant_type") != "client_credentials" {
		t.Fatalf("grant_type = %q", form.Get("grant_type"))
	}
	if form.Get("scope") != "business.api" {
		t.Fatalf("scope = %q, want business.api", form.Get("scope"))
	}
	if form.Get("client_assertion") == "" {
		t.Fatal("client_assertion missing")
	}
}

func thingsPagingHandler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/things", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			writeJSONResp(t, w, http.StatusOK, ListResponse[testAttrs]{
				Data:  []ResourceObject[testAttrs]{{Type: "thing", ID: "1", Attributes: testAttrs{Name: "a"}}},
				Links: Links{Next: "http://" + r.Host + "/v1/things?cursor=2"},
			})
			return
		}
		writeJSONResp(t, w, http.StatusOK, ListResponse[testAttrs]{
			Data: []ResourceObject[testAttrs]{
				{Type: "thing", ID: "2", Attributes: testAttrs{Name: "b"}},
				{Type: "thing", ID: "3", Attributes: testAttrs{Name: "c"}},
			},
		})
	})
	return mux
}

func TestListFollowsPagination(t *testing.T) {
	tok := startTokenServer(t, nil)
	api := httptest.NewServer(thingsPagingHandler(t))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	items, err := List[testAttrs](context.Background(), c, "/v1/things", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	if items[0].ID != "1" || items[2].ID != "3" {
		t.Fatalf("unexpected order: %+v", items)
	}
}

func TestListSeqIterates(t *testing.T) {
	tok := startTokenServer(t, nil)
	api := httptest.NewServer(thingsPagingHandler(t))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)

	var ids []string
	for item, err := range ListSeq[testAttrs](context.Background(), c, "/v1/things", nil) {
		if err != nil {
			t.Fatalf("ListSeq yielded error: %v", err)
		}
		ids = append(ids, item.ID)
	}
	if len(ids) != 3 {
		t.Fatalf("got %d ids, want 3 (%v)", len(ids), ids)
	}

	// 早期終了（break）でページングを止められること。
	count := 0
	for range ListSeq[testAttrs](context.Background(), c, "/v1/things", nil) {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("early break count = %d, want 2", count)
	}
}

func TestWriteMethodsAndBodies(t *testing.T) {
	tok := startTokenServer(t, nil)

	type capture struct {
		method string
		path   string
		body   map[string]any
	}
	var last capture

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last.method = r.Method
		last.path = r.URL.Path
		last.body = nil
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&last.body)
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "x"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	ctx := context.Background()

	if _, err := Create[testAttrs](ctx, c, "/v1/things", map[string]any{"data": map[string]any{"type": "thing"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if last.method != http.MethodPost || last.path != "/v1/things" {
		t.Fatalf("Create sent %s %s", last.method, last.path)
	}
	if last.body["data"] == nil {
		t.Fatalf("Create body missing data: %v", last.body)
	}

	if _, err := Update[testAttrs](ctx, c, "/v1/things/1", map[string]any{"data": map[string]any{"id": "1"}}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if last.method != http.MethodPatch || last.path != "/v1/things/1" {
		t.Fatalf("Update sent %s %s", last.method, last.path)
	}

	if err := Delete(ctx, c, "/v1/things/1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if last.method != http.MethodDelete {
		t.Fatalf("Delete sent %s", last.method)
	}
}

func TestModifyRelationshipBody(t *testing.T) {
	tok := startTokenServer(t, nil)

	var gotMethod string
	var gotData []Data
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		var payload struct {
			Data []Data `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotData = payload.Data
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	err := ModifyRelationship(context.Background(), c, http.MethodPost,
		"/v1/blueprints/bp1/relationships/orgDevices",
		[]Data{{Type: "orgDevices", ID: "d1"}, {Type: "orgDevices", ID: "d2"}})
	if err != nil {
		t.Fatalf("ModifyRelationship: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if len(gotData) != 2 || gotData[0].ID != "d1" || gotData[1].Type != "orgDevices" {
		t.Fatalf("unexpected data: %+v", gotData)
	}
}

func TestAPIErrorDecodeAndClassifier(t *testing.T) {
	tok := startTokenServer(t, nil)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResp(t, w, http.StatusNotFound, map[string]any{
			"errors": []map[string]any{{
				"status": "404", "code": "NOT_FOUND", "title": "Not Found", "detail": "no such thing",
			}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound = false for %v", err)
	}
	if IsRateLimited(err) || IsUnauthorized(err) {
		t.Fatalf("misclassified: %v", err)
	}
}

func TestListRefusesCrossHostNext(t *testing.T) {
	tok := startTokenServer(t, nil)

	var evilCalls int
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		evilCalls++
		writeJSONResp(t, w, http.StatusOK, ListResponse[testAttrs]{})
	}))
	t.Cleanup(evil.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResp(t, w, http.StatusOK, ListResponse[testAttrs]{
			Data:  []ResourceObject[testAttrs]{{Type: "thing", ID: "1", Attributes: testAttrs{Name: "a"}}},
			Links: Links{Next: evil.URL + "/v1/things?cursor=2"},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	_, err := List[testAttrs](context.Background(), c, "/v1/things", nil)
	if err == nil {
		t.Fatal("expected error for cross-host links.next")
	}
	if !errors.Is(err, errCrossHost) {
		t.Fatalf("err = %v, want errCrossHost", err)
	}
	if evilCalls != 0 {
		t.Fatalf("cross-host server received %d requests, want 0", evilCalls)
	}
}

func TestDoRefusesCrossHostRedirect(t *testing.T) {
	tok := startTokenServer(t, nil)

	var evilCalls int
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		evilCalls++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(evil.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/steal", http.StatusFound)
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err == nil {
		t.Fatal("expected error for cross-host redirect")
	}
	if !errors.Is(err, errCrossHost) {
		t.Fatalf("err = %v, want errCrossHost", err)
	}
	if evilCalls != 0 {
		t.Fatalf("cross-host server received %d requests, want 0", evilCalls)
	}
}

func TestAPIErrorNonJSONBody(t *testing.T) {
	tok := startTokenServer(t, nil)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("<html>upstream error</html>"))
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("not an APIError: %v", err)
	}
	if ae.StatusCode != http.StatusBadRequest || len(ae.Errors) != 0 {
		t.Fatalf("unexpected APIError: %+v", ae)
	}
	if !strings.Contains(ae.RawBody, "upstream error") {
		t.Fatalf("RawBody = %q, want body snippet", ae.RawBody)
	}
	if !strings.Contains(err.Error(), "upstream error") {
		t.Fatalf("Error() = %q, want body snippet included", err.Error())
	}
}

func TestRetryThenSuccess(t *testing.T) {
	tok := startTokenServer(t, nil)
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "ok"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(2))
	obj, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err != nil {
		t.Fatalf("Get after retry: %v", err)
	}
	if obj.Attributes.Name != "ok" {
		t.Fatalf("unexpected: %+v", obj)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 calls, got %d", calls)
	}
}

func TestDoRejectsOversizedResponse(t *testing.T) {
	old := maxResponseBytes
	maxResponseBytes = 1024
	t.Cleanup(func() { maxResponseBytes = old })

	tok := startTokenServer(t, nil)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"thing","id":"1","attributes":{"name":"`))
		_, _ = w.Write(bytes.Repeat([]byte("x"), 4096))
		_, _ = w.Write([]byte(`"}}}`))
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL)
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err = %v, want response-size-limit error", err)
	}
}

func TestPostNotRetriedOn5xx(t *testing.T) {
	tok := startTokenServer(t, nil)
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSONResp(t, w, http.StatusInternalServerError, map[string]any{
			"errors": []map[string]any{{"status": "500", "code": "INTERNAL", "title": "boom", "detail": "server broke"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(3))
	_, err := Create[testAttrs](context.Background(), c, "/v1/things",
		map[string]any{"data": map[string]any{"type": "thing"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *APIError
	if !errors.As(err, &ae) || ae.StatusCode != http.StatusInternalServerError {
		t.Fatalf("err = %v, want APIError 500", err)
	}
	if len(ae.Errors) == 0 || ae.Errors[0].Code != "INTERNAL" {
		t.Fatalf("expected decoded error body, got %+v", ae)
	}
	if calls != 1 {
		t.Fatalf("POST attempted %d times, want exactly 1", calls)
	}
}

func TestPostRetriedOn429(t *testing.T) {
	tok := startTokenServer(t, nil)
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "ok"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(2))
	obj, err := Create[testAttrs](context.Background(), c, "/v1/things",
		map[string]any{"data": map[string]any{"type": "thing"}})
	if err != nil {
		t.Fatalf("Create after 429: %v", err)
	}
	if obj.Attributes.Name != "ok" {
		t.Fatalf("unexpected: %+v", obj)
	}
	if calls != 2 {
		t.Fatalf("POST attempted %d times, want 2", calls)
	}
}

func TestGetRetriedOn500(t *testing.T) {
	tok := startTokenServer(t, nil)
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "ok"}},
		})
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(2))
	obj, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err != nil {
		t.Fatalf("Get after 500: %v", err)
	}
	if obj.Attributes.Name != "ok" || calls != 2 {
		t.Fatalf("obj=%+v calls=%d", obj, calls)
	}
}

func TestRateLimitedExhausted(t *testing.T) {
	tok := startTokenServer(t, nil)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(api.Close)

	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !IsRateLimited(err) {
		t.Fatalf("IsRateLimited = false for %v", err)
	}
}
