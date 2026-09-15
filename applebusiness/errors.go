package applebusiness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrorObject is a single JSON:API error entry inside an APIError.
type ErrorObject struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// APIError is a non-2xx API response (a JSON:API error document when the
// server sent one). For 429 / 5xx it describes the last response received,
// after retries were exhausted or when retries are disabled.
type APIError struct {
	StatusCode int
	Errors     []ErrorObject `json:"errors"`
	// RawBody holds a truncated snippet of the response body when it was not
	// a parsable JSON:API error document (e.g. an HTML page from a proxy).
	// It is empty when Errors is populated.
	RawBody string `json:"-"`
	// Header holds the response headers (e.g. Retry-After, or request IDs to
	// quote to support). Like RawBody it is excluded from JSON so that logging
	// an APIError as JSON does not dump headers unintentionally.
	Header http.Header `json:"-"`
	// RetryAfter is the server's Retry-After header, parsed from either
	// delay-seconds or an HTTP-date. It is 0 when the header is absent,
	// unparsable, or already in the past; callers must then fall back to their
	// own backoff (Apple does not document whether 429 responses carry it).
	RetryAfter time.Duration `json:"-"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("applebusiness: API error %d: %s - %s", e.StatusCode, e.Errors[0].Code, e.Errors[0].Detail)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("applebusiness: API error %d: %s", e.StatusCode, e.RawBody)
	}
	return fmt.Sprintf("applebusiness: API error %d", e.StatusCode)
}

// エラーボディの読み取り上限と、Error() に残す断片の長さ。
const (
	errBodyReadLimit  = 64 << 10 // 64 KiB
	errBodySnippetLen = 200
)

// decodeAPIError builds an *APIError from a non-2xx response. JSON:API error
// documents populate Errors; anything else (HTML from a load balancer, plain
// text, ...) is kept as a truncated RawBody snippet so the caller still gets
// a clue about what the server said. The response headers and the parsed
// Retry-After are always kept. The body is read up to errBodyReadLimit; the
// caller remains responsible for draining and closing it.
func decodeAPIError(resp *http.Response) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyReadLimit))
	if json.Unmarshal(raw, e) == nil && len(e.Errors) > 0 {
		return e
	}
	e.RawBody = bodySnippet(raw)
	return e
}

// bodySnippet は Error() に載せるための、前後の空白を除いて先頭 errBodySnippetLen バイトに切った本文を返す。
func bodySnippet(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > errBodySnippetLen {
		s = s[:errBodySnippetLen]
	}
	return s
}

// TokenError is a non-200 response from the OAuth token endpoint, returned when
// an access token cannot be obtained. It happens before the API request is
// sent, so it is not an *APIError: API calls (Get / Create / Client.Do, ...)
// return it wrapped in *url.Error, and Client.AccessToken returns it directly.
// Detect it with errors.As, or with IsRateLimited / IsUnauthorized, which match
// token errors as well as API errors.
//
// A 429 here usually means access tokens are being minted too often (e.g. a new
// Client per API call); reuse one Client, or one token source, per credential.
type TokenError struct {
	StatusCode int
	// Code is the OAuth 2.0 "error" value, e.g. invalid_client or invalid_request.
	Code string
	// Description is the OAuth 2.0 "error_description" value.
	Description string
	// RawBody holds a truncated snippet of the response body when it was not an
	// OAuth 2.0 JSON error document. It is empty when Code or Description is set.
	RawBody string `json:"-"`
	// Header holds the response headers. Like APIError.Header it is excluded
	// from JSON so that logging a TokenError does not dump headers.
	Header http.Header `json:"-"`
	// RetryAfter is the parsed Retry-After header (delay-seconds or HTTP-date);
	// 0 when absent, unparsable or in the past.
	RetryAfter time.Duration `json:"-"`
}

func (e *TokenError) Error() string {
	if e.Code != "" || e.Description != "" {
		return fmt.Sprintf("applebusiness oauth: token failed (%d): %s %s", e.StatusCode, e.Code, e.Description)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("applebusiness oauth: token failed (%d): %s", e.StatusCode, e.RawBody)
	}
	return fmt.Sprintf("applebusiness oauth: token failed (%d)", e.StatusCode)
}

// retryable は時間を置けば解消し得るトークンエラー（429 / 5xx）かを返す。
// 認証情報の誤り（invalid_client 等）は再試行しても結果が変わらない。
func (e *TokenError) retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// unauthorizedTokenCodes は、認証情報そのものの誤りを示す OAuth 2.0 のエラーコード
// （RFC 6749 §5.2）。ステータスが 400 でも IsUnauthorized で true にする。
var unauthorizedTokenCodes = map[string]bool{
	"invalid_client":      true,
	"invalid_grant":       true,
	"unauthorized_client": true,
}

// decodeTokenError builds a *TokenError from a non-200 token endpoint response.
// An OAuth 2.0 JSON error document populates Code / Description; anything else
// is kept as a truncated RawBody snippet. The body is read up to
// errBodyReadLimit; the caller remains responsible for draining and closing it.
func decodeTokenError(resp *http.Response) *TokenError {
	e := &TokenError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyReadLimit))
	var doc struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if json.Unmarshal(raw, &doc) == nil && (doc.Error != "" || doc.Description != "") {
		e.Code, e.Description = doc.Error, doc.Description
		return e
	}
	e.RawBody = bodySnippet(raw)
	return e
}

// Typed error predicates. Each returns true when err (or a wrapped error) is an
// *APIError with the corresponding HTTP status. IsRateLimited and IsUnauthorized
// also match a *TokenError from the token endpoint; the others match *APIError
// only, so a token endpoint failure never looks like, e.g., a missing resource.
//
//	if applebusiness.IsNotFound(err) { ... }

// IsNotFound reports whether err is a 404 from the API (*APIError only).
func IsNotFound(err error) bool { return statusIs(err, 404) }

// IsRateLimited reports whether err is a 429 from the API (*APIError, including
// the value returned when retries are exhausted or disabled) or from the token
// endpoint (*TokenError). Read RetryAfter via errors.As to learn when to try
// again; it is 0 when the server did not say.
func IsRateLimited(err error) bool {
	if statusIs(err, 429) {
		return true
	}
	var te *TokenError
	return errors.As(err, &te) && te.StatusCode == http.StatusTooManyRequests
}

// IsUnauthorized reports whether err is a 401 from the API (*APIError), or a
// token endpoint failure caused by the credentials (*TokenError with status 401
// or error code invalid_client / invalid_grant / unauthorized_client).
func IsUnauthorized(err error) bool {
	if statusIs(err, 401) {
		return true
	}
	var te *TokenError
	return errors.As(err, &te) && (te.StatusCode == http.StatusUnauthorized || unauthorizedTokenCodes[te.Code])
}

// IsForbidden reports whether err is a 403 from the API (*APIError only; it can
// occur, e.g., on insufficient relationship permissions).
func IsForbidden(err error) bool { return statusIs(err, 403) }

// IsConflict reports whether err is a 409 from the API (*APIError only).
func IsConflict(err error) bool { return statusIs(err, 409) }

func statusIs(err error, code int) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == code
	}
	return false
}
