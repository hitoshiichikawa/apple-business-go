package applebusiness

import "net/http"

// Option は NewClient の挙動を上書きする関数オプション。
type Option func(*options)

type options struct {
	baseURL    string
	tokenURL   string
	maxRetries int
	userAgent  string
	httpClient *http.Client
}

// WithBaseURL は API のベースURLを上書きする（既定は DefaultBusinessBaseURL）。
func WithBaseURL(u string) Option {
	return func(o *options) {
		if u != "" {
			o.baseURL = u
		}
	}
}

// WithTokenURL は OAuth トークン端点を上書きする（テストや /auth/oauth2/v2/token の指定に使用）。
func WithTokenURL(u string) Option {
	return func(o *options) {
		if u != "" {
			o.tokenURL = u
		}
	}
}

// WithMaxRetries は 429 / 5xx 時のリトライ回数を設定する（0 は既定値の4として扱われる）。
func WithMaxRetries(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxRetries = n
		}
	}
}

// WithUserAgent は API リクエストの User-Agent ヘッダを設定する。
func WithUserAgent(s string) Option {
	return func(o *options) {
		o.userAgent = s
	}
}

// WithHTTPClient は基盤の *http.Client（Transport / Timeout）を指定する。
// アクセストークンの注入（OAuth2 トランスポート）は維持される。
func WithHTTPClient(hc *http.Client) Option {
	return func(o *options) {
		if hc != nil {
			o.httpClient = hc
		}
	}
}
