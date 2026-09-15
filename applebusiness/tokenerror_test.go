package applebusiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// scriptedTokenServer は受信のたびに calls を加算し、fail(n) が true を返した回（n は 1 始まり）は
// fail 側で失敗応答を書く。false の回は正常なトークンを返す。
func scriptedTokenServer(t *testing.T, calls *atomic.Int32, fail func(n int32, w http.ResponseWriter) bool) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if fail(n, w) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(s.Close)
	return s
}

// writeOAuthError は OAuth 2.0 形式（RFC 6749 §5.2）のエラー応答を書く。code が空なら本文なし。
func writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	if code == "" {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","error_description":"` + desc + `"}`))
}

// tokenFailAlways は毎回同じ OAuth エラーを返す。
func tokenFailAlways(status int, code, desc string) func(int32, http.ResponseWriter) bool {
	return func(_ int32, w http.ResponseWriter) bool {
		writeOAuthError(w, status, code, desc)
		return true
	}
}

// tokenFailFirst は 1 回目だけ OAuth エラーを返し、2 回目以降は成功させる。
func tokenFailFirst(status int, code, desc string) func(int32, http.ResponseWriter) bool {
	return func(n int32, w http.ResponseWriter) bool {
		if n > 1 {
			return false
		}
		writeOAuthError(w, status, code, desc)
		return true
	}
}

// thingServer は GET / POST に 1 件のリソースを返す API サーバ（受信回数を calls に数える）。
func thingServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	return countingServer(t, calls, func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResp(t, w, http.StatusOK, SingleResponse[testAttrs]{
			Data: ResourceObject[testAttrs]{Type: "thing", ID: "1", Attributes: testAttrs{Name: "ok"}},
		})
	})
}

func createThing(c *Client) error {
	_, err := Create[testAttrs](context.Background(), c, "/v1/things",
		map[string]any{"data": map[string]any{"type": "thing"}})
	return err
}

func getThing(c *Client) error {
	_, err := Get[testAttrs](context.Background(), c, "/v1/things/1")
	return err
}

func TestTokenRateLimited_RetriesDisabled_IsRateLimitedTrue(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if !IsRateLimited(err) {
		t.Fatalf("IsRateLimited(%v) = false, want true", err)
	}
}

func TestTokenRateLimitedOnce_Get_RetriesAndSucceeds(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailFirst(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))

	// Act
	err := getThing(c)

	// Assert
	if err != nil || tokCalls.Load() != 2 {
		t.Fatalf("err=%v tokenCalls=%d, want nil and 2", err, tokCalls.Load())
	}
}

func TestTokenRateLimitedOnce_Create_RetriesAndSendsAPIRequestOnce(t *testing.T) {
	// Arrange: トークン取得の失敗時点では API へ未送信なので、POST でも二重実行にならない
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailFirst(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))

	// Act
	err := createThing(c)

	// Assert
	if err != nil || apiCalls.Load() != 1 {
		t.Fatalf("err=%v apiCalls=%d, want nil and exactly 1", err, apiCalls.Load())
	}
}

func TestTokenServerErrorOnce_Create_Retries(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailFirst(http.StatusInternalServerError, "", ""))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))

	// Act
	err := createThing(c)

	// Assert
	if err != nil || tokCalls.Load() != 2 {
		t.Fatalf("err=%v tokenCalls=%d, want nil and 2", err, tokCalls.Load())
	}
}

func TestTokenInvalidClient_RetriesEnabled_NotRetried(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusBadRequest, "invalid_client", "bad assertion"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(1))

	// Act
	err := getThing(c)

	// Assert
	if err == nil || tokCalls.Load() != 1 {
		t.Fatalf("err=%v tokenCalls=%d, want an error and exactly 1 token request", err, tokCalls.Load())
	}
}

func TestTokenInvalidClient_IsUnauthorizedTrue(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusBadRequest, "invalid_client", "bad assertion"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if !IsUnauthorized(err) {
		t.Fatalf("IsUnauthorized(%v) = false, want true", err)
	}
}

func TestTokenInvalidClient_IsRateLimitedFalse(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusBadRequest, "invalid_client", "bad assertion"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if IsRateLimited(err) {
		t.Fatalf("IsRateLimited(%v) = true, want false", err)
	}
}

func TestToken401WithoutBody_IsUnauthorizedTrue(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusUnauthorized, "", ""))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if !IsUnauthorized(err) {
		t.Fatalf("IsUnauthorized(%v) = false, want true", err)
	}
}

func TestToken404_IsNotFoundFalse(t *testing.T) {
	// Arrange: トークン端点の 404 を「リソースが存在しない」と誤判定させない
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusNotFound, "", ""))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if err == nil || IsNotFound(err) {
		t.Fatalf("err=%v IsNotFound=%v, want an error that is not IsNotFound", err, IsNotFound(err))
	}
}

func TestTokenNonJSONBody_ErrorContainsSnippet(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, func(_ int32, w http.ResponseWriter) bool {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>upstream gateway error</html>"))
		return true
	})
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "upstream gateway error") {
		t.Fatalf("err = %v, want the body snippet in Error()", err)
	}
}

func TestTokenOAuthErrorBody_ErrorKeepsExistingFormat(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	const want = "applebusiness oauth: token failed (429): invalid_request Too many requests"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %v, want it to contain %q", err, want)
	}
}

func TestTokenRateLimited_ReturnsTokenErrorWithDetails(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, func(_ int32, w http.ResponseWriter) bool {
		w.Header().Set("Retry-After", "120")
		writeOAuthError(w, http.StatusTooManyRequests, "invalid_request", "Too many requests")
		return true
	})
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	var te *TokenError
	if !errors.As(err, &te) {
		t.Fatalf("errors.As(%v, *TokenError) = false", err)
	}
	if te.StatusCode != http.StatusTooManyRequests || te.Code != "invalid_request" ||
		te.Description != "Too many requests" || te.RetryAfter != 120*time.Second ||
		te.Header.Get("Retry-After") != "120" {
		t.Fatalf("unexpected TokenError: %+v", te)
	}
}

func TestTokenNonJSONBody_KeepsRawBodySnippet(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, func(_ int32, w http.ResponseWriter) bool {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>upstream gateway error</html>"))
		return true
	})
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL, WithMaxRetries(0))

	// Act
	err := getThing(c)

	// Assert
	var te *TokenError
	if !errors.As(err, &te) || te.Code != "" || !strings.Contains(te.RawBody, "upstream gateway error") {
		t.Fatalf("err = %v, want *TokenError with the body snippet in RawBody", err)
	}
}

func TestAccessToken_TokenRateLimited_ReturnsTokenError(t *testing.T) {
	// Arrange
	var tokCalls, apiCalls atomic.Int32
	tok := scriptedTokenServer(t, &tokCalls, tokenFailAlways(http.StatusTooManyRequests, "invalid_request", "Too many requests"))
	api := thingServer(t, &apiCalls)
	c := newTestClient(t, api.URL, tok.URL)

	// Act
	_, _, err := c.AccessToken()

	// Assert
	var te *TokenError
	if !errors.As(err, &te) || te.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("err = %v, want *TokenError with status 429", err)
	}
}
