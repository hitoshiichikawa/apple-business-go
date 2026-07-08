# 引き継ぎプロンプト — apple-business-go（Claude Code 用）

あなたはこのリポジトリ（`apple-business-go`）の開発を引き継ぐ。これは **Apple Business API（旧 Apple Business/School Manager API）の非公式 Go SDK**。
以下を読み、まず「最初にやること」を実行してから作業する。日本語で応答すること。書き物（README/ドキュメント/講義資料）では「共有」という語を使わない（「コア」「共通基盤」等で言い換える）。

---

## 0. 最初にやること（必ず最初に実行）

```bash
go mod tidy
go build ./...          # examples 含む全パッケージがビルドできること
go vet ./...
gofmt -l .              # 何も出力されないこと（出たら gofmt -w . で整形）
go test -race ./...     # 全パッケージ ok（examples は [no test files] でOK）
```

- 直近の状態: `go test -race ./...` は**全パッケージ green**を実機で確認済み（examples 4つだけ `[no test files]`）。
- この環境（前任=Claude.ai）では **Go ツールチェーンが無く未コンパイル**のまま書いた差分があるため、上記が通ることをまず確認すること。落ちたらそこから直す。
- モジュールは **Go 1.23**（`iter` の range-over-func を使用）。リポジトリ名＝モジュールパス（`github.com/hitoshiichikawa/apple-business-go`）。

---

## 1. このSDKの全体像

JSON:API 形式。OAuth2 client_credentials + ES256 JWT クライアントアサーション。

- `applebusiness/`（コア）: `Client`、`NewClient(cfg, ...Option)`、OAuth(`oauth.go`)、汎用ジェネリクス
  `List`/`ListSeq`(range-over-func)/`Get`/`Relationship`/`Create`/`Update`/`Delete`/`ModifyRelationship`、
  型付きエラー判定 `IsNotFound`/`IsRateLimited`/`IsUnauthorized`/`IsForbidden`/`IsConflict`、
  `Client.AccessToken()`(認証単体確認)、エンベロープ型(`types.go`)、`APIError`、リトライ(429/5xx指数バックオフ)。
- ドメイン: `devices/`（orgDevices・mdmServers・appleCareCoverage・割り当てactivities）、`people/`（users・userGroups）、
  `apps/`（apps・packages＝読み取り）、`blueprints/`（CRUD＋リレーション）、`configurations/`（CRUD・CUSTOM_SETTING）、
  `auditevents/`（読み取り。共通エンベロープ＋eventData型付け）。
- `examples/`: `list-devices`、`smoke-test`（疎通確認）、`dump-all`（全リード系の応答ダンプ）、
  `assign-devices`（割り当て/解除＋結果CSV取得）、`write-test`（全書き込みAPIの動作確認）。
- `docs/apple-business-api-datatypes.md`（DocC由来の型・列挙・eventData一覧／**信頼できる一次資料**）、
  `docs/apple-business-api-reference.md`（書き込み仕様等）。
- 認証情報は全 examples 共通の環境変数で渡す（下記）。秘密鍵・トークンはコミット/ログ出力しない。

### 環境変数（examples 共通）
| 変数 | 必須 | 説明 |
|---|---|---|
| `AXM_CLIENT_ID` | ✓ | API アカウントの Client ID（`BUSINESSAPI.xxxx…`） |
| `AXM_KEY_ID` | ✓ | Key ID（JWT ヘッダ `kid`） |
| `AXM_PRIVATE_KEY_PATH` | ✓ | EC P-256 秘密鍵 `.pem` のパス |
| `AXM_TEAM_ID` |  | 省略時は Client ID を使用（AxM では同一が通例） |
| `AXM_SCOPE` |  | `business.api`/`school.api`（省略時 client_id 接頭辞で自動判定） |
| `AXM_BASE_URL` |  | 既定 `https://api-business.apple.com`（ASM は `https://api-school.apple.com`） |
| `AXM_TOKEN_URL` |  | 既定 `https://account.apple.com/auth/oauth2/token`（検証用に上書き可） |

---

## 2. 実APIで確定済みの事実（重要・実機検証済み）

これらは推測ではなく、ユーザーの実資格情報で叩いて確認済み。勝手に「直さない」こと。

- **OAuth**: トークンPOST先 `https://account.apple.com/auth/oauth2/token`。アサーションの `aud` は
  `https://account.apple.com/auth/oauth2/v2/token`。ヘッダ `alg=ES256, typ=JWT, kid=Key ID`。
  クレーム `iss=Team ID`(=client_id), `sub=client_id`, `iat`, `exp`(最大180日), `jti`(UUID)。
  `scope=business.api|school.api`。アクセストークン60分。`filter[...]` のパーセントエンコード送信は受理される。
- **auditEvents**: `GET /v1/auditEvents` は **`filter[startTimestamp]` が必須**（無いと 400 `PARAMETER_ERROR.REQUIRED`）。
  通常 `filter[endTimestamp]` と併用、ISO8601(RFC3339 UTC)。SDK は `auditevents.ListRange(start, end, q)`。
  応答は共通エンベロープ + `eventData<Event>`（キー名＝`eventDataPropertyKey`、例 `eventDataApiAccountCreatedWithKey:{keyId}`）。
- **mdmServers**: 属性は一覧の各要素に入る。2.0 までは `GET_COLLECTION` のみ（単体 `GET /v1/mdmServers/{id}` は 403）だったが、
  **API 2.1（2026-06-03）で単体 GET + POST/PATCH/DELETE のフル CRUD に対応**（SDK: `GetMdmServer`/`CreateMdmServer`/`UpdateMdmServer`/`DeleteMdmServer`）。
- **mdmServers の割り当てデバイス**: `GET /v1/mdmServers/{id}/relationships/devices`（linkage＝`{type:orgDevices,id}` のID一覧）**のみ可**。
  related の `GET /v1/mdmServers/{id}/devices` は **403**（`GET_RELATED` 不可, allowed: GET_RELATIONSHIP）。
  フル属性は各 `orgDevices/{id}` を個別取得（SDK `devices.MdmServerDeviceList` がそれを実施＝N+1）。
- **割り当てアクティビティ**: `status` `IN_PROGRESS`→`COMPLETED`（**部分失敗でも COMPLETED**）。
  `subStatus` `COMPLETED_WITH_ERROR`（一部失敗。`downloadUrl` の CSV に明細。全件成功時はおそらく `COMPLETED`＝未観測）。
  代表的な失敗要因: **既に同じサーバへ割り当て済みのデバイスを再割り当て**（状態不変＋エラー計上）。
  `PollActivity` は「処理中(`IN_PROGRESS` 等)以外」を終了とみなす実装。
- **結果CSV(downloadUrl)**: 署名付き・期限付き。`curl` で取るなら `&` を含むため**シングルクォート必須**。
  `assign-devices -activity <id> -save-results <path>` で取り直し（再取得で署名URLが更新され期限切れ回避）。

---

## 3. 未確定 / 今後の確認

- アクティビティ全件成功時の `subStatus`（おそらく `COMPLETED`、未観測）。
- `auditEvents` のフィールド無しイベント（`DEVICE_IS_ERASED`/`DOMAIN_*`/`EXTERNAL_ACCOUNT_*` 等）の実ペイロード有無。
- `Configuration` 作成時の Apple 側プロファイル検証の要件（`write-test` の最小 `.mobileconfig` が通るか）。
- 旧来「不可」とされた操作（デバイス release、MDMトークン更新 等）の現在の可否。

---

## 4. ユーザー（Hitoshi）の好み・運用ルール

- 応答は**日本語**。書き物で「共有」を使わない。
- **明確化の質問→手順→プレースホルダ/コメント付きコード→表/図→最終成果物**の順で、実装可能な粒度を好む。高水準の要約より具体。
- 差分は **GitHub-ready**（README・マッピング表・分割テンプレート等）。インクリメンタルに精緻化。
- 破壊的/書き込み操作は**安全側**（既定ドライラン、明示フラグ、後始末、現状プレビュー）。誤りは率直に認めて直す。
- このリポジトリは未公開。タグ運用は暫定 `v0.2.0`（追加は後方互換）。push はユーザーが行う。
  （前任は毎ターン `rm -rf .git && git init` で作り直してZIP納品していた。Claude Code では通常の git 運用でよい。）

---

## 5. 推奨タスク（次の一手の候補）

1. **「最初にやること」を緑にする**（最優先。落ちている所があれば修正）。
2. `write-test` を実機で軽く検証（まずドライラン→`-yes`）。Configuration 作成が Apple 検証で失敗する場合は
   最小 `.mobileconfig`（`examples/write-test/main.go` の `sampleMobileconfig`）を有効なプロファイルに調整。
3. コアの堅牢性: 書き込みリトライの冪等性（POST 二重送信の回避）レビュー、`golangci-lint` 完全パス。
4. ドキュメント整合（`docs/*.md`・`CHANGELOG.md`・`ROADMAP.md` を実装に合わせて維持）。
5. 別リポジトリ `abm-scanner`（Web UI）の `internal/store`(pgx)・`internal/sync` 実装（必要になれば）。

作業のたびに `go build ./... && go vet ./... && gofmt -l . && go test -race ./...` を回し、`CHANGELOG.md` を更新すること。
