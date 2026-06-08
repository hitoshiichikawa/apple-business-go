package applebusiness

import (
	"net/http"

	"golang.org/x/oauth2"
)

// Option is a functional option that overrides NewClient's behavior.
type Option func(*options)

type options struct {
	baseURL     string
	tokenURL    string
	maxRetries  int
	userAgent   string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
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

// WithMaxRetries sets the retry count on 429 / 5xx responses (0 is treated as the default of 4).
func WithMaxRetries(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxRetries = n
		}
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
