package applebusiness

import (
	"crypto/rand"
	"encoding/json"
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
const (
	tokenURL     = "https://account.apple.com/auth/oauth2/token"
	audienceURL  = "https://account.apple.com/auth/oauth2/v2/token"
	assertionTTL = 180 * 24 * time.Hour
	tokenSkew    = 5 * time.Minute
)

// Credentials は Apple Business / School Manager のポータルで発行する資格情報。
type Credentials struct {
	ClientID   string // 例: "BUSINESSAPI.<uuid>"
	TeamID     string // issuer。AxMでは client_id と同一が通例
	KeyID      string // JWTヘッダ "kid"
	PrivateKey []byte // PEM, EC P-256（ES256用）
	Scope      string // "business.api" / "school.api"。空なら client_id から自動判定
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

type tokenSource struct {
	creds    Credentials
	client   *http.Client
	tokenURL string
}

func newTokenSource(creds Credentials, hc *http.Client, endpoint string) oauth2.TokenSource {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	if endpoint == "" {
		endpoint = tokenURL
	}
	return oauth2.ReuseTokenSource(nil, &tokenSource{creds: creds, client: hc, tokenURL: endpoint})
}

func (s *tokenSource) Token() (*oauth2.Token, error) {
	assertion, err := buildClientAssertion(s.creds)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":            {"client_credentials"},
		"client_id":             {s.creds.ClientID},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
		"scope":                 {s.creds.scope()},
	}

	req, err := http.NewRequest(http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("applebusiness oauth: token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var e struct {
			Err  string `json:"error"`
			Desc string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return nil, fmt.Errorf("applebusiness oauth: token failed (%d): %s %s", resp.StatusCode, e.Err, e.Desc)
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
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.issuer(),
		"sub": c.ClientID,
		"aud": audienceURL,
		"iat": now.Unix(),
		"exp": now.Add(assertionTTL).Unix(),
		"jti": newJTI(),
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

func newJTI() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
