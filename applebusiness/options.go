package applebusiness

import "net/http"

// Option is a functional option that overrides NewClient's behavior.
type Option func(*options)

type options struct {
	baseURL    string
	tokenURL   string
	maxRetries int
	userAgent  string
	httpClient *http.Client
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
