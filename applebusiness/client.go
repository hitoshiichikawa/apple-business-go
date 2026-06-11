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
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	DefaultBusinessBaseURL = "https://api-business.apple.com"
	DefaultSchoolBaseURL   = "https://api-school.apple.com"
)

// Config holds the settings for a Client.
type Config struct {
	BaseURL     string // defaults to DefaultBusinessBaseURL
	Credentials Credentials
	HTTPClient  *http.Client // used for OAuth; defaults are applied when nil
	MaxRetries  int          // retry count on 429 / 5xx; 4 when 0
}

// Client is an authenticated HTTP client for the Apple Business / School Manager API.
// Service packages such as devices and people accept this Client and operate on it.
type Client struct {
	baseURL    string
	origin     *url.URL // parsed baseURL; every request must match its scheme+host
	httpClient *http.Client
	maxRetries int
	userAgent  string
	ts         oauth2.TokenSource
}

// errCrossHost guards the bearer token: the oauth2 transport attaches it to
// every request this client makes, so requests whose scheme/host differ from
// the base URL (a poisoned links.next, a redirect to another origin, ...) are
// refused before anything is sent.
var errCrossHost = errors.New("applebusiness: refusing cross-host request")

func sameOrigin(u, origin *url.URL) bool {
	return strings.EqualFold(u.Scheme, origin.Scheme) && strings.EqualFold(u.Host, origin.Host)
}

// NewClient creates an authenticated client.
//
// By default the token source is built from Config.Credentials (client_id,
// key_id and private_key are required). Alternatively, inject a caller-managed
// token source with WithTokenSource; in that case Credentials are optional and
// the injected source is used as-is (see WithTokenSource for why this matters
// for token reuse and private-key residency).
//
// In addition to the Config values, settings can be overridden with Options
// (WithBaseURL / WithTokenURL / WithMaxRetries / WithUserAgent / WithHTTPClient /
// WithTokenSource).
func NewClient(cfg Config, opts ...Option) (*Client, error) {
	o := options{
		baseURL:    cfg.BaseURL,
		maxRetries: cfg.MaxRetries,
		httpClient: cfg.HTTPClient,
	}
	for _, fn := range opts {
		fn(&o)
	}

	// Resolve the token source. An injected source (WithTokenSource) wins and
	// makes Credentials optional; otherwise build one from Credentials, which
	// are then required.
	ts := o.tokenSource
	if ts == nil {
		if cfg.Credentials.ClientID == "" || cfg.Credentials.KeyID == "" || len(cfg.Credentials.PrivateKey) == 0 {
			return nil, errors.New("applebusiness: client_id, key_id and private_key are required (or inject a token source with WithTokenSource)")
		}
		ts = newTokenSource(cfg.Credentials, o.httpClient, o.tokenURL)
	}

	if o.baseURL == "" {
		o.baseURL = DefaultBusinessBaseURL
	}
	if o.maxRetries == 0 {
		o.maxRetries = 4
	}

	origin, err := url.Parse(o.baseURL)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return nil, fmt.Errorf("applebusiness: invalid base URL %q", o.baseURL)
	}

	// Transport for API calls. When WithHTTPClient is supplied, its Transport /
	// Timeout are used as the base; the token-injecting oauth2.Transport is kept
	// on top.
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

	// oauth2.Transport はリダイレクトの各ホップでもトークンを再付与するため、
	// net/http 標準の「クロスホスト時に Authorization を落とす」保護が効かない。
	// CheckRedirect で同一オリジン以外への追従を拒否してトークン送出を防ぐ。
	httpClient := &http.Client{
		Timeout:   apiTimeout,
		Transport: &oauth2.Transport{Source: ts, Base: base},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !sameOrigin(req.URL, origin) {
				return fmt.Errorf("%w: redirect to %q (base %q)", errCrossHost, req.URL, o.baseURL)
			}
			if len(via) >= 10 {
				return errors.New("applebusiness: stopped after 10 redirects")
			}
			return nil
		},
	}
	return &Client{baseURL: o.baseURL, origin: origin, httpClient: httpClient, maxRetries: o.maxRetries, userAgent: o.userAgent, ts: ts}, nil
}

// AccessToken returns the current access token (issuing a new one if needed) and its expiry.
// It is useful for verifying authentication on its own. Regular API calls attach the token
// automatically, so calling this explicitly is not required.
func (c *Client) AccessToken() (string, time.Time, error) {
	tok, err := c.ts.Token()
	if err != nil {
		return "", time.Time{}, err
	}
	return tok.AccessToken, tok.Expiry, nil
}

// BaseURL returns the base URL of this client.
func (c *Client) BaseURL() string { return c.baseURL }

// Do sends a request to the given absolute URL and decodes the response into out.
// 429 / 5xx responses are retried with exponential backoff (honoring Retry-After),
// with one exception: POST requests are retried only on 429. A 5xx (or a network
// error) after a POST may mean the server already committed the write, so
// retrying could execute it twice; those errors are returned immediately.
// It is normally used by service packages through List/Get/Create.
//
// Because the bearer token is attached to every request, rawurl must point at
// the same scheme+host as the client's base URL; anything else (including
// redirects to another origin) is refused before the request is sent.
func (c *Client) Do(ctx context.Context, method, rawurl string, body []byte, out any) error {
	u, err := url.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("applebusiness: parse request URL: %w", err)
	}
	if !sameOrigin(u, c.origin) {
		return fmt.Errorf("%w: %q (base %q)", errCrossHost, rawurl, c.baseURL)
	}

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
			// クロスホストリダイレクト拒否はリトライしても結果が変わらない。
			if errors.Is(err, errCrossHost) {
				return err
			}
			// POST はレスポンス未受領でもサーバ側で処理済みの可能性があり、
			// 再送すると二重実行になり得るため再試行しない。
			if method == http.MethodPost {
				return err
			}
			lastErr = err
			if attempt == c.maxRetries || !sleepBackoff(ctx, attempt, 0) {
				return err
			}
			continue
		}

		// 429 は「処理されなかった」ことが確実なので全メソッドで再試行する。
		// 5xx は処理済みの可能性があるため、非冪等な POST では再試行しない。
		retryable := resp.StatusCode == http.StatusTooManyRequests ||
			(resp.StatusCode >= 500 && method != http.MethodPost)
		if retryable {
			wait := retryAfter(resp)
			drainAndClose(resp.Body)
			lastErr = &APIError{StatusCode: resp.StatusCode}
			if attempt == c.maxRetries || !sleepBackoff(ctx, attempt, wait) {
				return lastErr
			}
			continue
		}

		if resp.StatusCode >= 400 {
			defer drainAndClose(resp.Body)
			return decodeAPIError(resp)
		}

		if out != nil {
			defer drainAndClose(resp.Body)
			lr := &io.LimitedReader{R: resp.Body, N: maxResponseBytes + 1}
			if err := json.NewDecoder(lr).Decode(out); err != nil {
				if lr.N <= 0 {
					return fmt.Errorf("applebusiness: response body exceeds %d bytes", maxResponseBytes)
				}
				return fmt.Errorf("applebusiness: decode response: %w", err)
			}
		} else {
			drainAndClose(resp.Body)
		}
		return nil
	}
	return lastErr
}

// 異常応答（巨大ボディ）からの保護。正常なページ応答には十分すぎる上限を取る。
// テストから差し替えるため var にしている。
var maxResponseBytes int64 = 32 << 20 // 32 MiB

// drainAndClose 時に読み捨てる最大量。残りがこれを超える場合は接続再利用を
// 諦めて Close する（無制限に読み捨てない）。
const maxDrainBytes = 1 << 20 // 1 MiB

// drainAndClose reads the body (bounded) before closing so the underlying
// keep-alive connection can be reused (a partially read body forces the
// transport to discard the connection).
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxDrainBytes))
	_ = body.Close()
}

// ---------------------------------------------------------------------------
// 内部ヘルパ
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 汎用リクエストヘルパ（サービスパッケージから利用）
// ---------------------------------------------------------------------------

// pager is satisfied by page-shaped responses that carry a links.next URL.
// followPages is the single place that walks a links.next chain; List / ListSeq /
// Relationship are all built on top of it.
type pager interface{ nextLink() string }

func (r ListResponse[A]) nextLink() string      { return r.Links.Next }
func (r RelationshipResponse) nextLink() string { return r.Links.Next }

// followPages GETs endpoint, decodes each page into P and hands it to handle,
// following links.next until exhausted. handle returns false to stop early.
func followPages[P pager](ctx context.Context, c *Client, endpoint string, handle func(P) bool) error {
	for endpoint != "" {
		var page P
		if err := c.Do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
			return err
		}
		if !handle(page) {
			return nil
		}
		endpoint = page.nextLink()
	}
	return nil
}

// List walks a list endpoint following links.next until exhausted and returns all items.
func List[A any](ctx context.Context, c *Client, path string, q url.Values) ([]ResourceObject[A], error) {
	var out []ResourceObject[A]
	for item, err := range ListSeq[A](ctx, c, path, q) {
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// ListSeq enumerates a list endpoint with lazy paging (Go 1.23 range-over-func).
// It yields one item at a time across pages without loading everything into memory.
// On error it yields (zero, err) once and stops.
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
		err := followPages(ctx, c, endpoint, func(page ListResponse[A]) bool {
			for _, item := range page.Data {
				if !yield(item, nil) {
					return false
				}
			}
			return true
		})
		if err != nil {
			var zero ResourceObject[A]
			yield(zero, err)
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

// Relationship walks a relationships endpoint (an array of {type,id}) across all pages.
func Relationship(ctx context.Context, c *Client, path string) ([]Data, error) {
	var out []Data
	err := followPages(ctx, c, c.baseURL+path, func(page RelationshipResponse) bool {
		out = append(out, page.Data...)
		return true
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Create creates a resource with POST and returns the data from the SingleResponse.
//
// POST is not idempotent: Create retries only on 429 (where the server is
// known not to have processed the request). On 5xx or network errors the
// error is returned without retrying — re-run only after confirming the
// resource was not created.
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

// Update updates a resource with PATCH and returns the data from the SingleResponse.
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

// Delete deletes a resource with DELETE (expecting 204 No Content).
func Delete(ctx context.Context, c *Client, path string) error {
	return c.Do(ctx, http.MethodDelete, c.baseURL+path, nil, nil)
}

// ModifyRelationship sends {"data":[{type,id},...]} to a relationships endpoint.
// method is POST (add) / DELETE (remove) / PATCH (replace the set). Success is 204/200.
func ModifyRelationship(ctx context.Context, c *Client, method, path string, items []Data) error {
	raw, err := json.Marshal(struct {
		Data []Data `json:"data"`
	}{Data: items})
	if err != nil {
		return fmt.Errorf("applebusiness: marshal relationship: %w", err)
	}
	return c.Do(ctx, method, c.baseURL+path, raw, nil)
}

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
