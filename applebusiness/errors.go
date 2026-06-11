package applebusiness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrorObject is a single JSON:API error entry inside an APIError.
type ErrorObject struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// APIError is a JSON:API error response.
type APIError struct {
	StatusCode int
	Errors     []ErrorObject `json:"errors"`
	// RawBody holds a truncated snippet of the response body when it was not
	// a parsable JSON:API error document (e.g. an HTML page from a proxy).
	// It is empty when Errors is populated.
	RawBody string `json:"-"`
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
// a clue about what the server said.
func decodeAPIError(resp *http.Response) error {
	e := &APIError{StatusCode: resp.StatusCode}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyReadLimit))
	if json.Unmarshal(raw, e) == nil && len(e.Errors) > 0 {
		return e
	}
	if s := strings.TrimSpace(string(raw)); s != "" {
		if len(s) > errBodySnippetLen {
			s = s[:errBodySnippetLen]
		}
		e.RawBody = s
	}
	return e
}

// Typed error predicates. Each returns true when err (or a wrapped error) is an
// *APIError with the corresponding HTTP status.
//
//	if applebusiness.IsNotFound(err) { ... }

// IsNotFound reports whether err is a 404.
func IsNotFound(err error) bool { return statusIs(err, 404) }

// IsRateLimited reports whether err is a 429 (including the value returned when retries are exhausted).
func IsRateLimited(err error) bool { return statusIs(err, 429) }

// IsUnauthorized reports whether err is a 401.
func IsUnauthorized(err error) bool { return statusIs(err, 401) }

// IsForbidden reports whether err is a 403 (which can occur, e.g., on insufficient relationship permissions).
func IsForbidden(err error) bool { return statusIs(err, 403) }

// IsConflict reports whether err is a 409.
func IsConflict(err error) bool { return statusIs(err, 409) }

func statusIs(err error, code int) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == code
	}
	return false
}
