# CLAUDE.md

Claude Code 向けのプロジェクトコンテキスト。作業前にこのファイルと
`README.md`、`docs/apple-business-api-datatypes.md`（データ型の一次情報）、
`docs/apple-business-api-reference.md`、`ROADMAP.md`、`CLAUDE_CODE_PROMPT.md` を読むこと。

## プロジェクト

- **apple-business-go** — Apple Business（旧 Apple Business Manager / School Manager）API の **非公式 Go SDK**。
- モジュール: `github.com/hitoshiichikawa/apple-business-go`（Go 1.23。`ListSeq` の range-over-func を使用）。
- 認証は OAuth2 client_credentials + ES256 JWT client assertion。
- Apple Business は device / people / **brand** / support の4本柱。SDK は柱ごとのパッケージで拡張する。
- 全8カテゴリを実装済み（読み取り + 書き込み）。各 `Attributes` は公式 DocC（`datatypes.md`）と全フィールド一致。

## 構成と責務

```
applebusiness/   コア（共通基盤）: Client, Config, Credentials, OAuth2/JWT, transport, リトライ,
                 JSON:API エンベロープ, 汎用ヘルパ（List/Get/Relationship/Create/Update/Delete/ModifyRelationship）
devices/         デバイス管理: orgDevices, mdmServers, appleCareCoverage, 割り当て/解除アクティビティ,
                 mdmDevices（Apple MDM 登録デバイスの一覧・詳細）
blueprints/      Blueprint管理: CRUD + 割り当て（apps/configurations/packages/orgDevices/users/userGroups）
configurations/  Configuration管理: CRUD（CUSTOM_SETTING、configurationProfile=Base64）
apps/            アプリ/パッケージ（読み取り）: apps, packages
auditevents/     監査イベント（読み取り）: auditEvents（共通エンベロープ + eventData + Payload）
people/          ピープル管理: users, userGroups
orgunits/        組織単位（読み取り、API 2.2）: organizationalUnits（List/Get/Members）
examples/        実行サンプル
docs/            apple-business-api-datatypes.md（型の一次情報）/ apple-business-api-reference.md（§7 書き込み・§8 公式確認）
```

## 規約

- **Service パターン**: 各ドメインパッケージは `type Service struct{ c *applebusiness.Client }` と `New(c)` を持ち、
  `*applebusiness.Client` を受け取る。HTTP/認証/ページング/リトライはコアに集約し、ドメイン側は薄く保つ。
- **汎用ヘルパ**（コア）を使う: 読み取り=`List[A]`/`Get[A]`/`Relationship`、書き込み=`Create[A]`(POST)/`Update[A]`(PATCH)/`Delete`/`ModifyRelationship`。
  独自に `http.NewRequest` を書かない（必要なら `(*Client).Do` を使う）。
- **JSON:API** 形式: `data` / `attributes` / `relationships` / `links` / `meta`。フィールド名は Apple の camelCase。
- **列挙値は定数**を使う（`apps.OSIOS`, `devices.EraseStatusErased`/`LockStatusLocked`/`LostModeEnabled`,
  `people.UserGroupSmart`/`UserStatusActive`, `configurations.PlatformVisionOS`, `blueprints.StatusActive` 等）。一覧は datatypes.md §1。
- エラーは `applebusiness.APIError`。ページングは `links.next`（カーソル）を透過処理。
- 公開シンボルの doc コメントは英語、説明的なコメントは日本語可。
- 依存は最小（`golang-jwt/jwt/v5`, `golang.org/x/oauth2`）。新規依存は慎重に。

## ビルド / テスト

```bash
go mod tidy   # go.sum 生成（この環境では未実行）
go build ./...
go vet ./...
gofmt -l .    # 差分が出ないこと
go test -race ./...
```

## 重要な制約・注意

- **非公式**。エンドポイント/挙動の最終確定は Apple 公式ドキュメントで。
  ただしフィールド名・列挙値・監査モデル・Blueprints/Configurations の書き込み仕様は公式 DocC で確認済み（`datatypes.md` / reference §7-§8）。
- **秘密情報を絶対にコミットしない**（`.pem` / `.env` は `.gitignore` 済み）。鍵は呼び出し側がバイト列で渡す。
- 残る要確認（公式で最終確認。詳細は `ROADMAP.md` §4）:
  - トークンPOST先 `/auth/oauth2/token` か `/auth/oauth2/v2/token`（`aud` は v2）
  - `iss` が team_id か client_id か（AxMでは同一が通例）
  - `auditevents.List` の絞り込みクエリ名、アクティビティ終了ステータス文字列
- **書き込みのリトライ**: 現状コアの `Do` は 5xx を指数バックオフで再試行する。POST（作成）の二重実行リスクがあるため、
  書き込み系のリトライ/冪等性は要レビュー（必要なら書き込みは再試行しない等の調整）。
- **監査イベント**: `auditevents.Attributes` は共通項目（actorId/eventDateTime/type 等）を型付きで保持し、
  イベント固有データは `EventData`(生JSON) に収集。`Payload(&v)` で個別型へデコードする（型は datatypes.md §4.3）。

## スコープ外

- アプリ（`abm-scanner`: PostgreSQL/認証/Web API/UI）は**別リポジトリ**。この SDK はアプリに依存しない。
