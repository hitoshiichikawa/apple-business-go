package applebusiness

import (
	"net/http"

	"golang.org/x/oauth2"
)

// Option is a functional option that overrides NewClient's behavior.
type Option func(*options)

type options struct {
	baseURL    string
	tokenURL   string
	maxRetries int
	// maxRetriesSet は WithMaxRetries が指定されたか。WithMaxRetries(0) は
	// 「未指定」ではなく「リトライなし」を意味するため、指定の有無を別に持つ。
	maxRetriesSet bool
	userAgent     string
	httpClient    *http.Client
	tokenSource   oauth2.TokenSource
}

// WithBaseURL overrides the API base URL (defaults to DefaultBusinessBaseURL).
func WithBaseURL(u string) Option {
	return func(o *options) {
		if u != "" {
			o.baseURL = u
		}
	}
}

// WithTokenURL overrides the OAuth token endpoint (used for tests or to specify /auth/oauth2/v2/token).
func WithTokenURL(u string) Option {
	return func(o *options) {
		if u != "" {
			o.tokenURL = u
		}
	}
}

// WithMaxRetries sets the retry count on 429 / 5xx responses (and on network
// errors for non-POST requests). When given, it takes precedence over
// Config.MaxRetries.
//
// n <= 0 disables retries: every request is sent exactly once and the first
// failure is returned as-is, e.g. a 429 *APIError whose RetryAfter and Header
// tell the caller when to try again. Use it when rate limiting is handled above
// the SDK (such as a job queue that pauses on 429) so retries are not doubled.
func WithMaxRetries(n int) Option {
	return func(o *options) {
		o.maxRetries = max(n, 0)
		o.maxRetriesSet = true
	}
}

// WithUserAgent sets the User-Agent header on API requests.
func WithUserAgent(s string) Option {
	return func(o *options) {
		o.userAgent = s
	}
}

// WithHTTPClient sets the underlying *http.Client (Transport / Timeout).
// Access token injection (the OAuth2 transport) is preserved.
func WithHTTPClient(hc *http.Client) Option {
	return func(o *options) {
		if hc != nil {
			o.httpClient = hc
		}
	}
}

// WithTokenSource injects a caller-managed oauth2.TokenSource. When provided,
// NewClient does NOT build its own token source from Config.Credentials and does
// NOT require Credentials to be set; the injected source is solely responsible
// for producing access tokens (it may cache / refresh / rotate them however the
// caller wants). This decouples the OAuth token lifecycle from the *Client
// lifetime: a long-lived, shared token source can be reused across short-lived
// Clients so the token endpoint is not hit on every request, and the caller can
// control private-key residency (e.g. decrypt the key only on token refresh).
//
// If both Config.Credentials and WithTokenSource are supplied, the injected
// token source wins and Credentials are ignored for authentication.
//
// A nil ts is ignored (NewClient falls back to the Credentials-based source).
func WithTokenSource(ts oauth2.TokenSource) Option {
	return func(o *options) {
		if ts != nil {
			o.tokenSource = ts
		}
	}
}
