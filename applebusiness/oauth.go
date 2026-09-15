package applebusiness

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// OAuth 2.0 (client_credentials) + JWT client assertion (ES256)。
// 方式は Apple の公開フロー（Implementing OAuth for the Apple School and Business Manager API）と、
// 公式手順で生成された実アサーションのデコード結果で確認済み:
//   - トークンPOST先 = https://account.apple.com/auth/oauth2/token （/v2/token ではない）
//   - アサーションの aud = https://account.apple.com/auth/oauth2/v2/token
//   - ヘッダ alg=ES256, typ=JWT, kid=Key ID
//   - クレーム iss=Team ID（AxM では client_id と同一が通例）, sub=client_id, iat, exp(最大180日), jti
//   - フォーム: grant_type=client_credentials, client_id, client_assertion_type=...jwt-bearer,
//     client_assertion, scope=business.api|school.api（本実装はボディ送信。RFC 7521 準拠で URL クエリでも可）
//   - アクセストークン有効期限 60分
//
// アサーションはトークン取得のたびに新規生成するため、exp は Apple の上限
// （180日）ではなく短い TTL を使う。漏えい（プロキシのログ等）時に第三者が
// それを使い続けられる窓を縮めるための措置。クロックスキュー対策として
// iat を少し過去に補正する。
const (
	tokenURL         = "https://account.apple.com/auth/oauth2/token"
	audienceURL      = "https://account.apple.com/auth/oauth2/v2/token"
	assertionTTL     = 10 * time.Minute
	assertionIatSkew = 30 * time.Second
	tokenSkew        = 5 * time.Minute
)

// Credentials are the credentials issued in the Apple Business / School Manager portal.
type Credentials struct {
	ClientID   string // e.g. "BUSINESSAPI.<uuid>"
	TeamID     string // issuer; in AxM this is typically identical to client_id
	KeyID      string // JWT header "kid"
	PrivateKey []byte // PEM, EC P-256 (for ES256)
	Scope      string // "business.api" / "school.api"; inferred from client_id when empty
}

func (c Credentials) scope() string {
	if c.Scope != "" {
		return c.Scope
	}
	if strings.HasPrefix(c.ClientID, "BUSINESSAPI.") {
		return "business.api"
	}
	return "school.api"
}

func (c Credentials) issuer() string {
	if c.TeamID != "" {
		return c.TeamID
	}
	return c.ClientID
}

// complete reports whether the fields required to mint a token are set.
func (c Credentials) complete() bool {
	return c.ClientID != "" && c.KeyID != "" && len(c.PrivateKey) > 0
}

// NewTokenSource returns an oauth2.TokenSource that issues access tokens for the
// Credentials returned by fn and reuses each token until shortly before it
// expires. Use it to share one token per credential across Clients without
// keeping the private key in memory.
//
// fn is called only when a new token is needed: on the first Token call and then
// about once an hour (tokens are valid for one hour). The Credentials it returns,
// including the private key, are used for that single token request and are not
// kept by the source, so the key can stay encrypted at rest and be decrypted
// inside fn just for that call. A different KeyID / PrivateKey returned by a
// later call (key rotation) takes effect on the next refresh.
//
// Create one source per credential (e.g. per tenant), cache it, and pass it to
// every Client for that credential with WithTokenSource. The source is safe for
// concurrent use; callers that need a refresh at the same time wait for a single
// token request.
//
// oauth2.TokenSource.Token takes no context, so fn cannot receive one: apply
// your own timeout inside fn if it calls an external service such as a KMS. The
// token request itself uses the HTTP client's timeout (30 seconds by default).
//
// Only WithTokenURL and WithHTTPClient apply; all other options are ignored.
//
// Token returns an error, without calling the token endpoint, when fn is nil,
// when fn fails (the error is wrapped, so errors.Is / errors.As work), or when
// client_id, key_id or private_key is missing. When the token endpoint rejects
// the request, the error is a *TokenError.
func NewTokenSource(fn func() (Credentials, error), opts ...Option) oauth2.TokenSource {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return newCredentialsTokenSource(fn, o.httpClient, o.tokenURL)
}

// errNilCredentialsFunc は NewTokenSource に nil の fn が渡されたときに Token が返すエラー。
var errNilCredentialsFunc = errors.New("applebusiness oauth: NewTokenSource: credentials func is nil")

// credentialsTokenSource は Token のたびに fn から Credentials を取得してトークンを発行する。
// 秘密鍵を持ち続けないよう、Credentials はフィールドに保持しない（#38）。トークンのキャッシュと
// 同時更新の集約は、包んでいる oauth2.ReuseTokenSource が担う。
type credentialsTokenSource struct {
	fn       func() (Credentials, error)
	client   *http.Client
	tokenURL string
}

func newCredentialsTokenSource(fn func() (Credentials, error), hc *http.Client, endpoint string) oauth2.TokenSource {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	if endpoint == "" {
		endpoint = tokenURL
	}
	return oauth2.ReuseTokenSource(nil, &credentialsTokenSource{fn: fn, client: hc, tokenURL: endpoint})
}

func (s *credentialsTokenSource) Token() (*oauth2.Token, error) {
	if s.fn == nil {
		return nil, errNilCredentialsFunc
	}
	creds, err := s.fn()
	if err != nil {
		return nil, fmt.Errorf("applebusiness oauth: get credentials: %w", err)
	}
	if !creds.complete() {
		return nil, errors.New("applebusiness oauth: client_id, key_id and private_key are required")
	}
	return requestToken(s.client, s.tokenURL, creds)
}

// requestToken は creds からクライアントアサーションを作り、トークン端点で交換する。
// creds（秘密鍵を含む）はこの呼び出しの中でだけ使い、どこにも保持しない。
func requestToken(hc *http.Client, endpoint string, creds Credentials) (*oauth2.Token, error) {
	assertion, err := buildClientAssertion(creds)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":            {"client_credentials"},
		"client_id":             {creds.ClientID},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
		"scope":                 {creds.scope()},
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// bodyclose はヘルパ（drainAndClose）経由のクローズを追跡できないが、直後の defer でクローズしている。
	resp, err := hc.Do(req) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("applebusiness oauth: token request: %w", err)
	}
	defer drainAndClose(resp.Body)

	// 200 以外は *TokenError で返し、呼び出し側が errors.As / IsRateLimited /
	// IsUnauthorized で判定でき、Client.Do が再試行の可否を決められるようにする（#37）。
	if resp.StatusCode != http.StatusOK {
		return nil, decodeTokenError(resp)
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("applebusiness oauth: decode token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: tr.AccessToken,
		TokenType:   tr.TokenType,
		Expiry:      time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second).Add(-tokenSkew),
	}, nil
}

func buildClientAssertion(c Credentials) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.issuer(),
		"sub": c.ClientID,
		"aud": audienceURL,
		"iat": now.Add(-assertionIatSkew).Unix(),
		"exp": now.Add(assertionTTL).Unix(),
		"jti": jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = c.KeyID

	key, err := jwt.ParseECPrivateKeyFromPEM(c.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("applebusiness oauth: parse EC private key: %w", err)
	}
	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("applebusiness oauth: sign assertion: %w", err)
	}
	return signed, nil
}

func newJTI() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Go 1.24+ では rand.Read は失敗しない仕様だが、最低サポートの 1.23 では
		// 失敗し得る。固定値の jti を出さないようエラーとして伝播する。
		return "", fmt.Errorf("applebusiness oauth: generate jti: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
