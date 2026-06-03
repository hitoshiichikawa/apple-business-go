# Changelog

形式は [Keep a Changelog](https://keepachangelog.com/) に準拠し、バージョニングは SemVer（1.0 までは破壊的変更があり得る）。

## [Unreleased]

### Added
- コミュニティ/運用ファイル: `CONTRIBUTING.md` / `CODE_OF_CONDUCT.md` / Issue・PR テンプレート / Dependabot 設定。
- `devices` / `people` に godoc Example を追加。

## [0.3.0] - 2026-06-03

### Added
- `Client.AccessToken()`（認証単体の疎通確認用）。
- `auditevents`: 残りの `eventData` 型を追加（`AccountRoleLocation`/`AccountRoleLocationChanged`/`ApiAccountRoleLocationChanged`/`ApiAccountKey`/`ApiAccountNameChanged`）＋ `AuditEventType` 定数33種。
- `examples/write-test`: 全書き込みAPIを安全に試すCLI（既定ドライラン、実行は `-yes`）。Configuration を先に作成し、その無害な Web クリップ Configuration を Blueprint の中身に流用。Blueprint は実機仕様（中身＋割り当て先の両カテゴリ必須）に対応し、割り当て先は userGroups→users の順で1件のみ一時付与（デバイス直接割り当ては回避）して最後に削除。割り当て(assign)はオプトイン＋復元。
- 各ドメインパッケージの単体テスト（`devices`/`people`/`apps`/`blueprints`/`configurations`/`auditevents`、`httptest`・テーブル駆動）。
- `devices.MdmServerDeviceList(serverID)`: MDMサーバ割り当て済みデバイスを relationships のID→各 `orgDevices/{id}` 個別取得でフル取得（related は `GET_RELATED` 不可のため）。
- `auditevents.ListRange(start, end, q)`: 必須の `filter[startTimestamp]`/`filter[endTimestamp]` を付与して取得。
- `examples/smoke-test`: 実トークンでトークン取得＋読み取りAPIを確認するCLI。
- `examples/dump-all`: 全カテゴリの読み取り専用エンドポイントを順に叩き、生レスポンスを標準出力するCLI（`-limit`/`-only`/`-json`）。

### Changed
- `auditevents`: 識別子を Go 命名規約に合わせ `Api*` → `API*` にリネーム（型 `APIAccountKey`/`APIAccountNameChanged`/`APIAccountRoleLocationChanged`、定数 `TypeAPIAccount*`、フィールド `APIAccountRoleLocationList`）。golangci-lint(revive var-naming) 対応で、JSON タグ・定数値・API のキー名は不変。

### Verified
- `COMPLETED_WITH_ERROR` の一因が「同一 MDM サーバへの再割り当て」であることを確認（状態不変・エラー計上）。docs に注記。
- 割り当てアクティビティの実値を確認: `status` `IN_PROGRESS`→`COMPLETED`、`subStatus` `COMPLETED_WITH_ERROR`（部分失敗）、`downloadUrl` に CSV ログ。
- 監査イベントの実応答が共通エンベロープ＋`eventData<Event>`（例 `eventDataApiAccountCreatedWithKey:{keyId}`）と一致。`filter[...]` のパーセントエンコード送信も実機で受理を確認。
- `mdmServers/{id}/relationships/devices` は linkage（type=orgDevices のID）。related `…/devices` は `GET_RELATED` 不可（実APIで確認）。フル属性は `orgDevices/{id}` を個別取得。
- `mdmServers` は GET_COLLECTION のみ（単体 GET は 403）。`ListMdmServers` のコメントと docs に反映。
- `auditEvents` は `filter[startTimestamp]` が必須（実APIで確認）。`List` のドキュメントと `ListRange` に反映。
- OAuth フロー（token端点・aud・iss・ES256・scope・有効期限）を Apple の公開フローと実アサーションで確認し、コメント/ROADMAP を更新。
- 書き込み4 API を実トークンで確認（`examples/write-test -yes` が 6/6 成功）。判明した実機仕様を docs に反映:
  - Configuration の `configurationProfile` は **Base64 ではなく生 .mobileconfig XML**（Base64 は 400 `plist type mismatch`）。`PayloadContent` は非空必須。
  - Blueprint 作成は「中身(apps/packages/configurations)」と「割り当て先(orgDevices/users/userGroups)」を**両カテゴリ最低1つずつ必須**（`reference.md` の「relationships 任意」を訂正）。
  - Blueprint の `name` はスペース・括弧不可（英数字・ハイフン等のみ）。Configuration の `name` は許容。

## [0.2.0] - 2026-06-01

### Added
- `NewClient` に Functional Options を追加: `WithBaseURL` / `WithTokenURL` / `WithMaxRetries` / `WithUserAgent` / `WithHTTPClient`（`NewClient(cfg)` は後方互換）。
- 型付きエラー判定: `IsNotFound` / `IsRateLimited` / `IsUnauthorized` / `IsForbidden` / `IsConflict`。
- `ListSeq[A]`（Go 1.23 range-over-func）による遅延ページング。全件をメモリに展開しない。
- コアの単体テスト（`net/http/httptest`）と Example、CI（gofmt / vet / build / test -race / golangci-lint）。
- データ型リファレンス `docs/apple-business-api-datatypes.md`（列挙・属性・オブジェクト・監査を DocC から集約）。

### Changed
- `examples/write-test`: ドライランでも認証（トークン取得）を確認して `✓ token acquired` を表示し、`DRY RUN` と明示。中断と誤解されやすい環境変数の文言を削除。
- `examples/assign-devices`: `-activity <id>`（既存アクティビティの状態確認/結果取得、書き込みなし）と `-save-results <path>`（結果CSVを `downloadUrl` から取得して保存）を追加。
- `PollActivity` の終了判定を実値に合わせて変更（`IN_PROGRESS` 等の処理中以外を終了とみなす）。status/subStatus 定数を追加。
- `examples/assign-devices`: 安全版に刷新（既定ドライラン＋現状プレビュー、実行は `-yes`、`-server`/`-devices` フラグ、完了ポーリング）。
- 監査イベント `auditevents.Attributes` を「共通エンベロープ + イベント固有 `eventData`」モデルへ。`Payload(&v)` を追加。
- 各リソースの `Attributes` を公式 DocC と全フィールド一致（`devices` に `releaserEntityType`/`releaserId`、`AppleCareCoverage` の `status`、`MdmDevice`/`MdmDeviceDetail` 追加 等）。
- `people` の `UserGroup` 属性キーを `groupType` → `type`（`UserGroupType`）に修正。
- 列挙値を各パッケージで定数化（`apps.OSIOS`, `devices.EraseStatusErased`, `configurations.PlatformVisionOS` 等）。
- リトライ枯渇時の戻り値を `*APIError`（ステータス付き）に統一（型付き判定が効くように）。
- `go.mod` を Go 1.23 に更新（`iter` 利用のため）。

## [0.1.0]
- 初期リリース: `applebusiness` コア（OAuth2 ES256 / リトライ / ページング / JSON:API / 汎用ヘルパ）、
  `devices` / `people` / `apps` / `blueprints` / `configurations` / `auditevents`、examples、`v0.1.0` タグ。
