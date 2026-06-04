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
	httpClient *http.Client
	maxRetries int
	userAgent  string
	ts         oauth2.TokenSource
}

// NewClient creates an authenticated client from the given credentials.
// In addition to the Config values, settings can be overridden with Options
// (WithBaseURL / WithTokenURL / WithMaxRetries / WithUserAgent / WithHTTPClient).
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
// 429 / 5xx responses are retried with exponential backoff (honoring Retry-After).
// It is normally used by service packages through List/Get/Create.
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

// List walks a list endpoint following links.next until exhausted and returns all items.
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

// Relationship walks a relationships endpoint (an array of {type,id}) across all pages.
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

// Create creates a resource with POST and returns the data from the SingleResponse.
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

// APIError is a JSON:API error response.
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
