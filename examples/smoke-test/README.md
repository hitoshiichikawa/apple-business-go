# smoke-test — 実トークンでの疎通確認

実際の API アカウント資格情報を使って「トークン取得 →（任意で）読み取りAPI 1件取得」を確認するCLIです。

## 必要なもの（API アカウントから取得）
- EC P-256 の秘密鍵（`.pem`）
- Client ID（例 `BUSINESSAPI.xxxxxxxx-....`）
- Key ID
- Team ID（省略可。AxM では Client ID と同一が通例）

## 環境変数
| 変数 | 必須 | 説明 |
|---|---|---|
| `AXM_CLIENT_ID` | ✓ | Client ID |
| `AXM_KEY_ID` | ✓ | Key ID（JWT ヘッダ kid） |
| `AXM_PRIVATE_KEY_PATH` | ✓ | `.pem` のパス |
| `AXM_TEAM_ID` |  | 省略時は Client ID を使用 |
| `AXM_SCOPE` |  | `business.api` / `school.api`（省略時 client_id 接頭辞で自動判定） |
| `AXM_BASE_URL` |  | 既定 `https://api-business.apple.com`（ASM は `https://api-school.apple.com`） |
| `AXM_TOKEN_URL` |  | 既定 `https://account.apple.com/auth/oauth2/token`（検証用に上書き可） |

## 実行

```bash
export AXM_CLIENT_ID="BUSINESSAPI.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export AXM_KEY_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export AXM_TEAM_ID="$AXM_CLIENT_ID"          # 同一でよい
export AXM_PRIVATE_KEY_PATH="$PWD/private-key.pem"
# export AXM_SCOPE="business.api"            # 必要なら明示

# まず認証だけ確認（トークン取得のみ）
go run ./examples/smoke-test -token-only

# 読み取りAPIを1件叩く（既定は /v1/orgDevices）
go run ./examples/smoke-test
go run ./examples/smoke-test -path /v1/mdmServers
go run ./examples/smoke-test -path /v1/users -raw     # 生JSONを表示
```

成功例:
```
✓ token acquired: eyJhbGci… (expires 2026-06-02T05:00:00Z)
✓ GET /v1/orgDevices → 200 OK
  data: 1 件 (meta.paging.total=1234)
  先頭: type=orgDevices id=XXXXXXXXXX
```

## トラブルシュート
- **401 Unauthorized** … `client_id` / `team_id` / `key_id` / 秘密鍵 / `scope` のいずれか不一致。`-token-only` で切り分け。
- **403 Forbidden** … API アカウントの権限・対象範囲。relationships 系は権限で 403 になり得る。
- **404 Not Found** … `-path` の綴り。`/v1/...` から始める。
- **token exchange failed** … `.pem` が EC P-256 か、`AXM_TOKEN_URL` の指定、時刻ずれ（iat/exp）を確認。

## セキュリティ
- `.pem` や環境変数（トークン含む）は**コミット・ログ出力しない**こと。出力されるトークンは先頭8文字のみにマスクしています。
- 本CLIは**読み取りのみ**。書き込み（割り当て等）は別途 `examples/assign-devices` を参照（admin限定運用を推奨）。
