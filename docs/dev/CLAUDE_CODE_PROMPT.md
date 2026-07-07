# Claude Code プロンプト文案（apple-business-go）

> 下のブロックをそのまま Claude Code に貼り付けてください。リポジトリ直下（`apple-business-go/`）で実行する想定です。

---

あなたは `apple-business-go`（Apple Business API の非公式 Go SDK）の開発を担当します。
まず `CLAUDE.md`、`README.md`、`docs/apple-business-api-datatypes.md`（データ型の一次情報）、
`docs/apple-business-api-reference.md`（特に §7 書き込み仕様 / §8 公式確認済み）、`ROADMAP.md` を読み、
既存コードのパターン（`applebusiness` コア + 各ドメインの `Service` パターン、汎用ヘルパ
`List/Get/Relationship/Create/Update/Delete/ModifyRelationship`）を把握してから着手してください。

## 現状（2026-06 時点）
- 全8カテゴリを実装済み（読み取り + 書き込み）。
- 各リソースの `Attributes` は **公式 DocC（`docs/apple-business-api-datatypes.md`）と全フィールド一致済み**。
  列挙値は各パッケージで定数化済み（`apps.OSIOS`, `devices.EraseStatusErased`, `people.UserGroupSmart`,
  `configurations.PlatformVisionOS`, `blueprints.StatusActive` ほか）。
- `auditevents` は「共通エンベロープ `AuditEventCommonAttributes` + イベント固有 `eventData`」モデル。
  共通項目は型付きで保持し、`Payload(&v)` でイベント固有ペイロードをデコード（主要イベント型を同梱）。
- 追加済み: Functional Options（`With*`）、型付きエラー判定（`Is*`）、`ListSeq`（range-over-func）、
  コアの単体テスト（`httptest`）、CHANGELOG / `.golangci.yml` / CI。`go.mod` は Go 1.23。
- まだ無いもの: 各ドメインパッケージの単体テスト、書き込みリトライの冪等性レビュー、lint 完全パス、追加 Example。
- この開発環境には Go ツールチェーンが無く **未ビルド**。まず `go mod tidy && go build ./... && go vet ./...` を通すこと。

## ゴール
実装済みの SDK を、テストで固めつつ実用レベルへ引き上げる。既存の設計・命名・JSON:API 形式を踏襲し、
コアは薄く・ドメインは `Service` パターンを維持。新規依存は最小限に。後方互換を壊さない。

## タスク（優先度順）

1. **ビルド健全化（最初に）** — `go mod tidy`（`go.sum` 生成）→ `go build ./...` → `go vet ./...` を緑に。
   `gofmt -l .` の差分をゼロに。コンパイル/整形エラーがあれば最小修正。

2. **ユニットテスト拡充（最優先）** — コアは `applebusiness/client_test.go` にあり。各ドメインを `net/http/httptest` でテーブル駆動。外部通信はしない。
   - `applebusiness`: トークン交換（リクエスト形式・JWTクレーム・scope自動判定）、`List` のページング（`links.next` 追従）、
     `Get/Create/Update/Delete/ModifyRelationship` のメソッド・パス・ボディ、`APIError` デコード、429 / `Retry-After` リトライ。
   - `devices`: 割り当て/解除の `activityType`、`mdmServer`/`devices` リレーションのボディ、`PollActivity` の終了判定。
   - `blueprints`: `Create`(attributes+relationships)、`Update`、`AddTo`/`RemoveFrom`/`Replace` が
     POST/DELETE/PATCH で `{"data":[{type,id}]}` を送ること。
   - `configurations`: `Create`（`CUSTOM_SETTING` で `customSettingsValues` 必須、`configurationProfile` は Base64）、`Update`。
   - `auditevents`: `List` のページング、`UnmarshalJSON` が共通項目 + `eventData*` を収集すること、`Payload(&v)` のデコード。
   - `apps` / `people`: パスと属性デコード。
   - 目標カバレッジ: コア 85%+、各ドメイン 70%+。

3. **書き込みのリトライ/冪等性レビュー** — コア `Do` は現状 5xx を再試行する。POST（作成）の二重実行を避けるため、
   書き込み（POST/PATCH/DELETE）の再試行ポリシーを見直す（冪等メソッドのみ再試行、または呼び出し側で制御できるオプション化）。
   挙動を変える場合はテストを伴う。

4. **API設計の磨き込み** — 後方互換を壊さない範囲で。
   - Functional Options（`WithHTTPClient` / `WithUserAgent` / `WithMaxRetries` / `WithBaseURL`）。
   - 型付きエラー判定（`IsNotFound` / `IsRateLimited` / `IsUnauthorized`）。
   - ページングのイテレータ（Go 1.23 range-over-func。全件メモリ展開を避ける別APIとして追加、既存 `List` は維持）。

5. **任意の拡張**（必要に応じて。一次情報は `docs/apple-business-api-datatypes.md`）
   - 監査イベントの残り `eventData` 型（全33種のうち未型付けのもの）を追加。型は datatypes.md §4.3 準拠。
   - ~~`MdmDevice` / `MdmDeviceDetail` 型に対応するエンドポイント配線~~ → 実装済み（`GET /v1/mdmDevices` / `GET /v1/mdmDevices/{id}/details`。公式 DocC で確定）。
   - 不足エンドポイントは既存の `Service` パターン・汎用ヘルパで追加。

6. **品質・CI** — `golangci-lint` 導入、`go test -race`、`go vet ./...`、`go doc` 用 Example テスト。

## 制約・注意
- **非公式**。残る要確認事項（`ROADMAP.md` §7 / `CLAUDE.md`）はコメントで「要確認」を残し、勝手に確定させない。
  ※ フィールド名・列挙値・監査モデル・Blueprints/Configurations の書き込み仕様は確認済み（`datatypes.md` / reference §7-§8）。
- **秘密情報をコミットしない**（`.pem` / `.env`）。テストはダミー鍵・モックサーバで行う。
- 公開シンボルの doc コメントは英語、説明コメントは日本語可。コミットは小さく、Conventional Commits 推奨。
- アプリ（`abm-scanner`）はスコープ外。SDK はアプリに依存しないこと。

## 進め方
1. タスク1（ビルド健全化）→ タスク2（テスト）の順。パッケージ単位で PR 相当の小さな単位で進める。
2. 各タスク完了時に `go test -race ./...` が緑であること。

## 完了条件
- `go build ./...` / `go vet ./...` / `gofmt -l .`（差分なし）/ `go test -race ./...` がすべて成功。
- 上記カバレッジ目安を満たすテストがある。
- 公開APIの後方互換を維持（破壊的変更が要る場合は理由を明記）。
