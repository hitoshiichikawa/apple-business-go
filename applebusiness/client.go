package applebusiness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

const (
	DefaultBusinessBaseURL = "https://api-business.apple.com"
	DefaultSchoolBaseURL   = "https://api-school.apple.com"
)

// Config は Client の設定。
type Config struct {
	BaseURL     string // 既定 DefaultBusinessBaseURL
	Credentials Credentials
	HTTPClient  *http.Client // OAuth用。nilなら既定
	MaxRetries  int          // 429 / 5xx 時のリトライ回数。0なら4
}

// Client は Apple Business / School Manager API への認証済みHTTPクライアント。
// devices / people などのサービスパッケージはこの Client を受け取って動作する。
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	userAgent  string
	ts         oauth2.TokenSource
}

// NewClient は資格情報から認証済みクライアントを生成する。
// Config の値に加え、Option（WithBaseURL / WithTokenURL / WithMaxRetries / WithUserAgent / WithHTTPClient）で上書きできる。
func NewClient(cfg Config, opts ...Option) (*Client, error) {
	if cfg.Credentials.ClientID == "" || cfg.Credentials.KeyID == "" || len(cfg.Credentials.PrivateKey) == 0 {
		return nil, errors.New("applebusiness: client_id, key_id and private_key are required")
	}

	o := options{
		baseURL:    cfg.BaseURL,
		maxRetries: cfg.MaxRetries,
		httpClient: cfg.HTTPClient,
	}
	for _, fn := range opts {
		fn(&o)
	}

	if o.baseURL == "" {
		o.baseURL = DefaultBusinessBaseURL
	}
	if o.maxRetries == 0 {
		o.maxRetries = 4
	}

	// API 呼び出し用トランスポート。WithHTTPClient 指定時はその Transport / Timeout を基盤に使い、
	// トークン注入（oauth2.Transport）は維持する。
	var base = http.DefaultTransport
	apiTimeout := 60 * time.Second
	if o.httpClient != nil {
		if o.httpClient.Transport != nil {
			base = o.httpClient.Transport
		}
		if o.httpClient.Timeout != 0 {
			apiTimeout = o.httpClient.Timeout
		}
	}

	ts := newTokenSource(cfg.Credentials, o.httpClient, o.tokenURL)
	httpClient := &http.Client{
		Timeout:   apiTimeout,
		Transport: &oauth2.Transport{Source: ts, Base: base},
	}
	return &Client{baseURL: o.baseURL, httpClient: httpClient, maxRetries: o.maxRetries, userAgent: o.userAgent, ts: ts}, nil
}

// AccessToken は現在のアクセストークン（必要なら新規発行）と有効期限を返す。認証単体の疎通確認に使う。
// 通常の API 呼び出しでは内部で自動付与されるため、明示的に呼ぶ必要はない。
func (c *Client) AccessToken() (string, time.Time, error) {
	tok, err := c.ts.Token()
	if err != nil {
		return "", time.Time{}, err
	}
	return tok.AccessToken, tok.Expiry, nil
}

// BaseURL はこのクライアントのベースURLを返す。
func (c *Client) BaseURL() string { return c.baseURL }

// Do は与えられた絶対URLにリクエストし、レスポンスを out にデコードする。
// 429 / 5xx は指数バックオフ（Retry-After優先）で再試行する。
// 通常はサービスパッケージが List/Get/Create 経由で利用する。
func (c *Client) Do(ctx context.Context, method, rawurl string, body []byte, out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, rawurl, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt == c.maxRetries || !sleepBackoff(ctx, attempt, 0) {
				return err
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			wait := retryAfter(resp)
			_ = resp.Body.Close()
			lastErr = &APIError{StatusCode: resp.StatusCode}
			if attempt == c.maxRetries || !sleepBackoff(ctx, attempt, wait) {
				return lastErr
			}
			continue
		}

		if resp.StatusCode >= 400 {
			defer func() { _ = resp.Body.Close() }()
			return decodeAPIError(resp)
		}

		if out != nil {
			defer func() { _ = resp.Body.Close() }()
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("applebusiness: decode response: %w", err)
			}
		} else {
			_ = resp.Body.Close()
		}
		return nil
	}
	return lastErr
}

// ---------------------------------------------------------------------------
// 汎用リクエストヘルパ（サービスパッケージから利用）
// ---------------------------------------------------------------------------

// List はリストエンドポイントを links.next が尽きるまで辿り全件返す。
func List[A any](ctx context.Context, c *Client, path string, q url.Values) ([]ResourceObject[A], error) {
	endpoint := c.baseURL + path
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}
	var out []ResourceObject[A]
	for endpoint != "" {
		var page ListResponse[A]
		if err := c.Do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data...)
		endpoint = page.Links.Next
	}
	return out, nil
}

// ListSeq はリストエンドポイントを遅延ページングで列挙する（Go 1.23 range-over-func）。
// 全件をメモリに展開せず、ページを跨いで1件ずつ yield する。エラー時は (zero, err) を1度 yield して終了。
//
//	for item, err := range applebusiness.ListSeq[devices.DeviceAttributes](ctx, c, "/v1/orgDevices", nil) {
//	    if err != nil { return err }
//	    use(item)
//	}
func ListSeq[A any](ctx context.Context, c *Client, path string, q url.Values) iter.Seq2[ResourceObject[A], error] {
	return func(yield func(ResourceObject[A], error) bool) {
		endpoint := c.baseURL + path
		if len(q) > 0 {
			endpoint += "?" + q.Encode()
		}
		for endpoint != "" {
			var page ListResponse[A]
			if err := c.Do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
				var zero ResourceObject[A]
				yield(zero, err)
				return
			}
			for _, item := range page.Data {
				if !yield(item, nil) {
					return
				}
			}
			endpoint = page.Links.Next
		}
	}
}

func Get[A any](ctx context.Context, c *Client, path string) (*ResourceObject[A], error) {
	var resp SingleResponse[A]
	if err := c.Do(ctx, http.MethodGet, c.baseURL+path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Relationship は relationships エンドポイント（{type,id} 配列）を全ページ辿る。
func Relationship(ctx context.Context, c *Client, path string) ([]Data, error) {
	endpoint := c.baseURL + path
	var out []Data
	for endpoint != "" {
		var resp RelationshipResponse
		if err := c.Do(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Data...)
		endpoint = resp.Links.Next
	}
	return out, nil
}

// Create は POST でリソースを作成し、SingleResponse のデータを返す。
func Create[A any](ctx context.Context, c *Client, path string, body any) (*ResourceObject[A], error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("applebusiness: marshal body: %w", err)
	}
	var resp SingleResponse[A]
	if err := c.Do(ctx, http.MethodPost, c.baseURL+path, raw, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Update は PATCH でリソースを更新し、SingleResponse のデータを返す。
func Update[A any](ctx context.Context, c *Client, path string, body any) (*ResourceObject[A], error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("applebusiness: marshal body: %w", err)
	}
	var resp SingleResponse[A]
	if err := c.Do(ctx, http.MethodPatch, c.baseURL+path, raw, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Delete は DELETE でリソースを削除する（204 No Content を期待）。
func Delete(ctx context.Context, c *Client, path string) error {
	return c.Do(ctx, http.MethodDelete, c.baseURL+path, nil, nil)
}

// ModifyRelationship は relationships エンドポイントに {"data":[{type,id},...]} を送る。
// method は POST（追加）/ DELETE（削除）/ PATCH（集合の置換）。成功は 204/200。
func ModifyRelationship(ctx context.Context, c *Client, method, path string, items []Data) error {
	raw, err := json.Marshal(struct {
		Data []Data `json:"data"`
	}{Data: items})
	if err != nil {
		return fmt.Errorf("applebusiness: marshal relationship: %w", err)
	}
	return c.Do(ctx, method, c.baseURL+path, raw, nil)
}

// ---------------------------------------------------------------------------
// 内部ヘルパ / エラー
// ---------------------------------------------------------------------------

func sleepBackoff(ctx context.Context, attempt int, base time.Duration) bool {
	d := base
	if d <= 0 {
		d = time.Duration(math.Min(math.Pow(2, float64(attempt)), 32)) * 500 * time.Millisecond
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func retryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

// APIError は JSON:API のエラーレスポンス。
type APIError struct {
	StatusCode int
	Errors     []struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("applebusiness: API error %d: %s - %s", e.StatusCode, e.Errors[0].Code, e.Errors[0].Detail)
	}
	return fmt.Sprintf("applebusiness: API error %d", e.StatusCode)
}

func decodeAPIError(resp *http.Response) error {
	e := &APIError{StatusCode: resp.StatusCode}
	_ = json.NewDecoder(resp.Body).Decode(e)
	return e
}
