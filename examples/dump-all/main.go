// Command dump-all は、全カテゴリの「読み取り専用」エンドポイントを順に呼び出し、
// 実際のレスポンス(JSON)を標準出力に表示する確認用ツール。
//
// 安全のため GET のみ。割り当て/作成/更新/削除などの書き込みは実組織を変更するため一切行わない。
//
// 各「一覧」エンドポイントは先頭 limit 件だけ取得し、先頭要素の id を使って
// 詳細・サブリソース（assignedServer / appleCareCoverage / relationships 等）へドリルダウンする。
// 取得できない（404/403/空）エンドポイントはスキップして次へ進む（途中で止まらない）。
//
// 認証情報は smoke-test と同じ環境変数から読み込む:
//
//	AXM_CLIENT_ID, AXM_KEY_ID, AXM_PRIVATE_KEY_PATH（必須）
//	AXM_TEAM_ID, AXM_SCOPE, AXM_BASE_URL, AXM_TOKEN_URL（任意）
//
// 使い方:
//
//	go run ./examples/dump-all                      # 全カテゴリを既定 limit=3 で
//	go run ./examples/dump-all -limit 1             # 1件ずつ
//	go run ./examples/dump-all -only devices        # 特定カテゴリのみ
//	go run ./examples/dump-all -json > dump.json    # 機械可読(JSON配列)で保存
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

func main() {
	limit := flag.Int("limit", 3, "一覧エンドポイントの取得件数(limit)")
	only := flag.String("only", "", "カテゴリ名の部分一致でフィルタ（例 devices, blueprints, audit）")
	jsonMode := flag.Bool("json", false, "結果を JSON 配列で標準出力（既定は人間可読）")
	auditDays := flag.Int("audit-days", 7, "auditEvents の取得期間（日数。filter[startTimestamp] に使用）")
	timeout := flag.Duration("timeout", 120*time.Second, "全体タイムアウト")
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
	opts := []applebusiness.Option{applebusiness.WithUserAgent("apple-business-go/dump-all")}
	if tu := os.Getenv("AXM_TOKEN_URL"); tu != "" {
		opts = append(opts, applebusiness.WithTokenURL(tu))
	}
	c, err := applebusiness.NewClient(cfg, opts...)
	if err != nil {
		fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 認証（トークン取得）。失敗時はここで停止。
	tok, exp, err := c.AccessToken()
	if err != nil {
		fatalf("token exchange failed: %v", err)
	}
	fmt.Fprintf(os.Stderr, "✓ token acquired: %s… (expires %s)\n", mask(tok), exp.Format(time.RFC3339))
	fmt.Fprintln(os.Stderr, "※ 読み取り専用（GET のみ）。書き込みは行いません。")

	d := &dumper{c: c, ctx: ctx, limit: *limit, jsonMode: *jsonMode, only: strings.ToLower(*only), auditDays: *auditDays}
	d.run()

	if d.jsonMode {
		out, err := json.MarshalIndent(d.results, "", "  ")
		if err != nil {
			fatalf("marshal results: %v", err)
		}
		fmt.Println(string(out))
	}
	fmt.Fprintf(os.Stderr, "\n--- 完了: %d 呼び出し（成功 %d / スキップ(403,404) %d / 失敗 %d） ---\n", d.ok+d.skip+d.fail, d.ok, d.skip, d.fail)
}

// ---- ドライバ ----

type call struct {
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	OK       bool            `json:"ok"`
	Skipped  bool            `json:"skipped,omitempty"`
	Error    string          `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

type dumper struct {
	c              *applebusiness.Client
	ctx            context.Context
	limit          int
	jsonMode       bool
	only           string
	auditDays      int
	results        []call
	ok, skip, fail int
}

func (d *dumper) want(category string) bool {
	return d.only == "" || strings.Contains(strings.ToLower(category), d.only)
}

func (d *dumper) banner(category string) {
	if !d.jsonMode {
		fmt.Printf("\n########## %s ##########\n", category)
	} else {
		fmt.Fprintf(os.Stderr, "… %s\n", category)
	}
}

// get は単一 GET を実行して結果を記録し、レスポンス本文を返す（失敗時 nil）。
func (d *dumper) get(path string) json.RawMessage {
	var body json.RawMessage
	err := d.c.Do(d.ctx, "GET", d.c.BaseURL()+path, nil, &body)
	rec := call{Method: "GET", Path: path}
	switch {
	case err == nil:
		d.ok++
		rec.OK = true
		rec.Response = body
	case applebusiness.IsForbidden(err) || applebusiness.IsNotFound(err):
		// 許可されない操作（GET_INSTANCE 不可など）や未割り当て等。失敗ではなくスキップ扱い。
		d.skip++
		rec.Skipped = true
		rec.Error = err.Error()
	default:
		d.fail++
		rec.Error = err.Error()
	}
	d.record(rec)
	if err != nil {
		return nil
	}
	return body
}

// list は limit クエリを付けて GET する。
func (d *dumper) list(path string) json.RawMessage {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return d.get(path + sep + "limit=" + strconv.Itoa(d.limit))
}

func (d *dumper) record(rec call) {
	if d.jsonMode {
		d.results = append(d.results, rec)
		return
	}
	fmt.Printf("\n=== %s %s ===\n", rec.Method, rec.Path)
	switch {
	case rec.OK:
		fmt.Println(pretty(rec.Response))
	case rec.Skipped:
		fmt.Printf("— skipped: %s\n", rec.Error)
	default:
		fmt.Printf("!! error: %s\n", rec.Error)
	}
}

// run は全カテゴリの読み取りエンドポイントを順に叩く。
func (d *dumper) run() {
	esc := url.PathEscape

	// 1. Devices
	if d.want("Devices") {
		d.banner("Devices")
		body := d.list("/v1/orgDevices")
		if id := firstID(body); id != "" {
			d.get("/v1/orgDevices/" + esc(id))
			d.get("/v1/orgDevices/" + esc(id) + "/assignedServer")
			d.get("/v1/orgDevices/" + esc(id) + "/appleCareCoverage")
		}
	}

	// 2. Device Management Services (MDM servers)
	if d.want("Device Management Services") {
		d.banner("Device Management Services (mdmServers)")
		// mdmServers は GET_COLLECTION のみ許可（単体 GET_INSTANCE は 403）。属性は一覧の各要素に含まれる。
		body := d.list("/v1/mdmServers")
		if id := firstID(body); id != "" {
			// related の …/devices は GET_RELATED 不可。メンバーシップは relationships（IDのみ）から。
			d.list("/v1/mdmServers/" + esc(id) + "/relationships/devices")
		}
	}

	// 3. Users
	if d.want("Users") {
		d.banner("Users")
		body := d.list("/v1/users")
		if id := firstID(body); id != "" {
			d.get("/v1/users/" + esc(id))
		}
	}

	// 4. UserGroups
	if d.want("UserGroups") {
		d.banner("UserGroups")
		body := d.list("/v1/userGroups")
		if id := firstID(body); id != "" {
			d.get("/v1/userGroups/" + esc(id))
			d.list("/v1/userGroups/" + esc(id) + "/relationships/users")
		}
	}

	// 5. Apps and Packages
	if d.want("Apps and Packages") {
		d.banner("Apps")
		if id := firstID(d.list("/v1/apps")); id != "" {
			d.get("/v1/apps/" + esc(id))
		}
		d.banner("Packages")
		if id := firstID(d.list("/v1/packages")); id != "" {
			d.get("/v1/packages/" + esc(id))
		}
	}

	// 6. Blueprints
	if d.want("Blueprints") {
		d.banner("Blueprints")
		body := d.list("/v1/blueprints")
		if id := firstID(body); id != "" {
			d.get("/v1/blueprints/" + esc(id))
			for _, rel := range []string{"orgDevices", "userGroups", "users", "apps", "configurations", "packages"} {
				d.list("/v1/blueprints/" + esc(id) + "/relationships/" + rel)
			}
		}
	}

	// 7. Configurations
	if d.want("Configurations") {
		d.banner("Configurations")
		body := d.list("/v1/configurations")
		if id := firstID(body); id != "" {
			d.get("/v1/configurations/" + esc(id))
		}
	}

	// 8. Organizational Units（API 2.2、読み取り専用）
	if d.want("Organizational Units") {
		d.banner("Organizational Units")
		body := d.list("/v1/organizationalUnits")
		if id := firstID(body); id != "" {
			d.get("/v1/organizationalUnits/" + esc(id))
			d.list("/v1/organizationalUnits/" + esc(id) + "/relationships/users")
		}
	}

	// 9. Audit Events（filter[startTimestamp] が必須）
	if d.want("Audit Events") {
		d.banner("Audit Events")
		end := time.Now().UTC()
		start := end.AddDate(0, 0, -d.auditDays)
		q := url.Values{}
		q.Set("filter[startTimestamp]", start.Format(time.RFC3339))
		q.Set("filter[endTimestamp]", end.Format(time.RFC3339))
		q.Set("limit", strconv.Itoa(d.limit))
		d.get("/v1/auditEvents?" + q.Encode())
	}
}

// ---- ヘルパ ----

// firstID はリスト/単体レスポンスから最初の data.id を取り出す。
func firstID(b json.RawMessage) string {
	if len(b) == 0 {
		return ""
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return ""
	}
	t := bytes.TrimSpace(env.Data)
	if len(t) == 0 {
		return ""
	}
	if t[0] == '[' {
		var arr []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(env.Data, &arr); err == nil && len(arr) > 0 {
			return arr[0].ID
		}
		return ""
	}
	var one struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &one); err == nil {
		return one.ID
	}
	return ""
}

func pretty(b json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return string(b)
	}
	return buf.String()
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
