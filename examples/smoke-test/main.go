// Command smoke-test は実トークンでの疎通確認を行う。
// 1) クライアント生成 → 2) トークン取得（認証単体） → 3) 読み取り API を1件取得。
//
// 認証情報は環境変数から読み込む（秘密鍵はコミット・ログ出力しないこと）:
//
//	AXM_CLIENT_ID         例: BUSINESSAPI.xxxxxxxx-....
//	AXM_TEAM_ID           （省略時は client_id を使用。AxM では同一が通例）
//	AXM_KEY_ID            JWT ヘッダ kid
//	AXM_PRIVATE_KEY_PATH  EC P-256 の .pem へのパス
//	AXM_SCOPE             business.api | school.api（省略時は client_id から自動判定）
//	AXM_BASE_URL          省略時 https://api-business.apple.com（ASM は https://api-school.apple.com）
//	AXM_TOKEN_URL         省略時 https://account.apple.com/auth/oauth2/token（検証用に上書き可）
//
// 使い方:
//
//	go run ./examples/smoke-test                 # orgDevices を1件取得
//	go run ./examples/smoke-test -token-only     # 認証（トークン取得）だけ確認
//	go run ./examples/smoke-test -path /v1/mdmServers -raw
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

func main() {
	path := flag.String("path", "/v1/orgDevices", "GET するパス（例 /v1/mdmServers, /v1/users, /v1/apps）")
	limit := flag.Int("limit", 1, "limit クエリ（0なら付与しない）")
	tokenOnly := flag.Bool("token-only", false, "トークン取得のみ（API は呼ばない）")
	raw := flag.Bool("raw", false, "レスポンス JSON をそのまま表示")
	flag.Parse()

	keyPath := os.Getenv("AXM_PRIVATE_KEY_PATH")
	if keyPath == "" {
		fatalf("AXM_PRIVATE_KEY_PATH is required (path to EC .pem)")
	}
	pem, err := os.ReadFile(keyPath)
	if err != nil {
		fatalf("read private key: %v", err)
	}
	if os.Getenv("AXM_CLIENT_ID") == "" || os.Getenv("AXM_KEY_ID") == "" {
		fatalf("AXM_CLIENT_ID and AXM_KEY_ID are required")
	}

	cfg := applebusiness.Config{
		BaseURL: os.Getenv("AXM_BASE_URL"),
		Credentials: applebusiness.Credentials{
			ClientID:   os.Getenv("AXM_CLIENT_ID"),
			TeamID:     os.Getenv("AXM_TEAM_ID"),
			KeyID:      os.Getenv("AXM_KEY_ID"),
			PrivateKey: pem,
			Scope:      os.Getenv("AXM_SCOPE"),
		},
	}
	opts := []applebusiness.Option{applebusiness.WithUserAgent("apple-business-go/smoke-test")}
	if tu := os.Getenv("AXM_TOKEN_URL"); tu != "" {
		opts = append(opts, applebusiness.WithTokenURL(tu))
	}

	c, err := applebusiness.NewClient(cfg, opts...)
	if err != nil {
		fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1) 認証（トークン取得）
	tok, exp, err := c.AccessToken()
	if err != nil {
		fatalf("token exchange failed: %v\n  → client_id / team_id / key_id / private_key / scope と token 端点を確認", err)
	}
	fmt.Printf("✓ token acquired: %s… (expires %s)\n", mask(tok), exp.Format(time.RFC3339))
	if *tokenOnly {
		return
	}

	// 2) 読み取り API 呼び出し
	endpoint := c.BaseURL() + *path
	if *limit > 0 {
		sep := "?"
		if strings.Contains(*path, "?") {
			sep = "&"
		}
		endpoint += sep + "limit=" + strconv.Itoa(*limit)
	}

	var body json.RawMessage
	if err := c.Do(ctx, "GET", endpoint, nil, &body); err != nil {
		reportAPIError(*path, err)
		os.Exit(1)
	}
	fmt.Printf("✓ GET %s → 200 OK\n", *path)

	if *raw {
		fmt.Println(string(body))
		return
	}
	summarize(body)
}

func summarize(body json.RawMessage) {
	var env struct {
		Data json.RawMessage `json:"data"`
		Meta struct {
			Paging struct {
				Total int `json:"total"`
			} `json:"paging"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		fmt.Println("  (レスポンスを解釈できませんでした。 -raw で内容を確認してください)")
		return
	}
	trimmed := strings.TrimSpace(string(env.Data))
	switch {
	case strings.HasPrefix(trimmed, "["):
		var arr []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &arr)
		fmt.Printf("  data: %d 件 (meta.paging.total=%d)\n", len(arr), env.Meta.Paging.Total)
		if len(arr) > 0 {
			fmt.Printf("  先頭: type=%s id=%s\n", arr[0].Type, arr[0].ID)
		}
	case trimmed != "" && trimmed != "null":
		var one struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &one)
		fmt.Printf("  data: 単体 type=%s id=%s\n", one.Type, one.ID)
	default:
		fmt.Println("  data: なし")
	}
}

func reportAPIError(path string, err error) {
	switch {
	case applebusiness.IsUnauthorized(err):
		fmt.Fprintf(os.Stderr, "✗ 401 Unauthorized: %v\n  → client_id / team_id / key_id / private_key / scope を確認\n", err)
	case applebusiness.IsForbidden(err):
		fmt.Fprintf(os.Stderr, "✗ 403 Forbidden: %v\n  → API アカウントの権限・対象を確認\n", err)
	case applebusiness.IsNotFound(err):
		fmt.Fprintf(os.Stderr, "✗ 404 Not Found: %v\n  → パス %q を確認\n", err, path)
	case applebusiness.IsRateLimited(err):
		fmt.Fprintf(os.Stderr, "✗ 429 Rate Limited: %v\n", err)
	default:
		fmt.Fprintf(os.Stderr, "✗ request failed: %v\n", err)
	}
}

func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:8]
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
