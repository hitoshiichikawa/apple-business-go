package applebusiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// kidTokenServer は受信のたびに calls を加算し、client_assertion の JWT ヘッダー kid を記録して、
// expires_in 秒のトークンを返す擬似 OAuth 端点。delay > 0 なら応答前に待つ（同時更新の検証用）。
type kidTokenServer struct {
	srv   *httptest.Server
	calls atomic.Int32
	mu    sync.Mutex
	kids  []string
}

func startKidTokenServer(t *testing.T, expiresIn int, delay time.Duration) *kidTokenServer {
	t.Helper()
	ks := &kidTokenServer{}
	ks.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ks.calls.Add(1)
		_ = r.ParseForm()
		if tok, _, err := jwt.NewParser().ParseUnverified(r.PostForm.Get("client_assertion"), jwt.MapClaims{}); err == nil {
			kid, _ := tok.Header["kid"].(string)
			ks.mu.Lock()
			ks.kids = append(ks.kids, kid)
			ks.mu.Unlock()
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		writeJSONResp(t, w, http.StatusOK, map[string]any{
			"access_token": "shared-token", "token_type": "Bearer", "expires_in": expiresIn,
		})
	}))
	t.Cleanup(ks.srv.Close)
	return ks
}

func (ks *kidTokenServer) recordedKids() []string {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return slices.Clone(ks.kids)
}

// countingCreds は呼ばれた回数を calls に数えて creds を返す fn を作る。
func countingCreds(calls *atomic.Int32, creds Credentials) func() (Credentials, error) {
	return func() (Credentials, error) {
		calls.Add(1)
		return creds, nil
	}
}

// 期限が十分先のトークンは再利用され、fn もトークン端点も 1 回しか呼ばれない。
func TestNewTokenSource_ValidTokenIsReusedWithoutCallingFnAgain(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	var fnCalls atomic.Int32
	ts := NewTokenSource(countingCreds(&fnCalls, testCredentials(t)), WithTokenURL(ks.srv.URL))

	// Act
	_, err1 := ts.Token()
	_, err2 := ts.Token()

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("Token errors: %v, %v", err1, err2)
	}
	if fnCalls.Load() != 1 || ks.calls.Load() != 1 {
		t.Fatalf("fn calls = %d, token requests = %d; want 1 and 1", fnCalls.Load(), ks.calls.Load())
	}
}

// 期限切れのトークンは更新のたびに fn を呼び直し、トークンを再発行する。
func TestNewTokenSource_ExpiredTokenCallsFnAndRequestsAgain(t *testing.T) {
	// Arrange: expires_in=1 は tokenSkew（5 分）を引くと発行時点で期限切れになる。
	ks := startKidTokenServer(t, 1, 0)
	var fnCalls atomic.Int32
	ts := NewTokenSource(countingCreds(&fnCalls, testCredentials(t)), WithTokenURL(ks.srv.URL))

	// Act
	_, err1 := ts.Token()
	_, err2 := ts.Token()

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("Token errors: %v, %v", err1, err2)
	}
	if fnCalls.Load() != 2 || ks.calls.Load() != 2 {
		t.Fatalf("fn calls = %d, token requests = %d; want 2 and 2", fnCalls.Load(), ks.calls.Load())
	}
}

// 更新時に fn が別の Key ID を返したら、次のアサーションはその Key ID で署名される（鍵の差し替え）。
func TestNewTokenSource_RefreshUsesRotatedKeyID(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 1, 0)
	var n atomic.Int32
	ts := NewTokenSource(func() (Credentials, error) {
		creds := testCredentials(t)
		creds.KeyID = "kid-1"
		if n.Add(1) > 1 {
			creds.KeyID = "kid-2"
		}
		return creds, nil
	}, WithTokenURL(ks.srv.URL))

	// Act
	_, err1 := ts.Token()
	_, err2 := ts.Token()

	// Assert
	if err1 != nil || err2 != nil {
		t.Fatalf("Token errors: %v, %v", err1, err2)
	}
	if got := ks.recordedKids(); !slices.Equal(got, []string{"kid-1", "kid-2"}) {
		t.Fatalf("assertion kids = %v, want [kid-1 kid-2]", got)
	}
}

// fn がエラーを返したら、そのエラーを包んで返し、トークン端点は呼ばない。
func TestNewTokenSource_FnErrorIsReturnedWithoutRequest(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	errDecrypt := errors.New("decrypt failed")
	ts := NewTokenSource(func() (Credentials, error) {
		return Credentials{}, errDecrypt
	}, WithTokenURL(ks.srv.URL))

	// Act
	_, err := ts.Token()

	// Assert
	if !errors.Is(err, errDecrypt) {
		t.Fatalf("err = %v, want it to wrap errDecrypt", err)
	}
	if ks.calls.Load() != 0 {
		t.Fatalf("token requests = %d, want 0", ks.calls.Load())
	}
}

// fn が nil なら Token はエラーを返す（コンストラクタはエラーを返せないため）。
func TestNewTokenSource_NilFnReturnsError(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	ts := NewTokenSource(nil, WithTokenURL(ks.srv.URL))

	// Act
	_, err := ts.Token()

	// Assert
	if err == nil {
		t.Fatal("Token with nil fn: expected error, got nil")
	}
}

// fn が必須項目の欠けた Credentials を返したら、トークン端点を呼ばずにエラーを返す。
func TestNewTokenSource_MissingRequiredFieldReturnsErrorWithoutRequest(t *testing.T) {
	cases := map[string]func(*Credentials){
		"client_id":   func(c *Credentials) { c.ClientID = "" },
		"key_id":      func(c *Credentials) { c.KeyID = "" },
		"private_key": func(c *Credentials) { c.PrivateKey = nil },
	}
	for name, strip := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			ks := startKidTokenServer(t, 3600, 0)
			creds := testCredentials(t)
			strip(&creds)
			ts := NewTokenSource(func() (Credentials, error) { return creds, nil }, WithTokenURL(ks.srv.URL))

			// Act
			_, err := ts.Token()

			// Assert
			if err == nil || ks.calls.Load() != 0 {
				t.Fatalf("err = %v, token requests = %d; want error and 0 requests", err, ks.calls.Load())
			}
		})
	}
}

// トークン端点が 429 を返したら *TokenError を返す。
func TestNewTokenSource_TokenEndpoint429ReturnsTokenError(t *testing.T) {
	// Arrange
	var calls atomic.Int32
	tok := scriptedTokenServer(t, &calls, tokenFailAlways(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	ts := NewTokenSource(func() (Credentials, error) { return testCredentials(t), nil }, WithTokenURL(tok.URL))

	// Act
	_, err := ts.Token()

	// Assert
	var tokErr *TokenError
	if !errors.As(err, &tokErr) || tokErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("err = %v, want *TokenError with 429", err)
	}
}

// countingTransport は通過したリクエスト数を数えて base に委ねる。
type countingTransport struct {
	base  http.RoundTripper
	calls atomic.Int32
}

func (ct *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ct.calls.Add(1)
	return ct.base.RoundTrip(r)
}

// WithHTTPClient を渡すと、トークン取得のリクエストはその http.Client で送られる。
func TestNewTokenSource_HonorsWithHTTPClient(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	ct := &countingTransport{base: http.DefaultTransport}
	ts := NewTokenSource(func() (Credentials, error) { return testCredentials(t), nil },
		WithTokenURL(ks.srv.URL), WithHTTPClient(&http.Client{Transport: ct}))

	// Act
	_, err := ts.Token()

	// Assert
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if ct.calls.Load() != 1 {
		t.Fatalf("requests through custom transport = %d, want 1", ct.calls.Load())
	}
}

// トークン取得に関係しないオプションは無視される（渡してもトークンは取得できる）。
func TestNewTokenSource_IgnoresUnrelatedOptions(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	ts := NewTokenSource(func() (Credentials, error) { return testCredentials(t), nil },
		WithTokenURL(ks.srv.URL), WithBaseURL("not a url"), WithMaxRetries(0), WithUserAgent("ua/1.0"))

	// Act
	tok, err := ts.Token()

	// Assert
	if err != nil || tok.AccessToken != "shared-token" {
		t.Fatalf("Token = %v, %v; want shared-token", tok, err)
	}
}

// 1 つのトークンソースを 2 つの Client で共有すると、トークンの発行は 1 回で済む。
func TestNewTokenSource_SharedByTwoClientsMintsOnce(t *testing.T) {
	// Arrange
	ks := startKidTokenServer(t, 3600, 0)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "x"}},
		})
	}))
	t.Cleanup(api.Close)
	ts := NewTokenSource(func() (Credentials, error) { return testCredentials(t), nil }, WithTokenURL(ks.srv.URL))
	c1, err1 := NewClient(Config{BaseURL: api.URL}, WithTokenSource(ts))
	c2, err2 := NewClient(Config{BaseURL: api.URL}, WithTokenSource(ts))
	if err1 != nil || err2 != nil {
		t.Fatalf("NewClient errors: %v, %v", err1, err2)
	}

	// Act
	_, getErr1 := Get[testAttrs](context.Background(), c1, "/v1/things/1")
	_, getErr2 := Get[testAttrs](context.Background(), c2, "/v1/things/1")

	// Assert
	if getErr1 != nil || getErr2 != nil {
		t.Fatalf("Get errors: %v, %v", getErr1, getErr2)
	}
	if ks.calls.Load() != 1 {
		t.Fatalf("token requests = %d, want 1", ks.calls.Load())
	}
}

// 新しいトークンソースに同時に Token を呼んでも、トークンの発行は 1 回にまとまる。
func TestNewTokenSource_ConcurrentRefreshMintsOnce(t *testing.T) {
	// Arrange: 応答を遅らせて、全 goroutine の更新が重なるようにする。
	ks := startKidTokenServer(t, 3600, 50*time.Millisecond)
	ts := NewTokenSource(func() (Credentials, error) { return testCredentials(t), nil }, WithTokenURL(ks.srv.URL))
	const workers = 16
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	// Act
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ts.Token()
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	// Assert
	for err := range errs {
		if err != nil {
			t.Fatalf("Token: %v", err)
		}
	}
	if ks.calls.Load() != 1 {
		t.Fatalf("token requests = %d, want 1", ks.calls.Load())
	}
}
