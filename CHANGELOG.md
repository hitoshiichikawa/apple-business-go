# Changelog

形式は [Keep a Changelog](https://keepachangelog.com/) に準拠し、バージョニングは SemVer（1.0 までは破壊的変更があり得る）。

## [Unreleased]

### Added
- **デバイス管理サービス移行（MDM migration）対応（Apple Business API 2.3 / Apple School Manager API 1.6、2026-08-12 追加）**。デバイスを消去せずに別のデバイス管理サービスへ移行できる:
  - `devices.AssignWithMdmMigrationDeadline`: デバイスを MDM サーバへ割り当てつつ移行期限付きで移行をスケジュール（`activityType=ASSIGN_DEVICES_WITH_MDM_MIGRATION_DEADLINE`）。期限は最大 90 日先（範囲外は 409）。
  - `devices.UpdateMdmMigrationDeadline`: 進行中の移行の期限を変更（`activityType=UPDATE_MDM_MIGRATION_DEADLINE`。`mdmServer` リレーション不要）。
  - `devices.CancelMdmMigration`: 進行中の移行をキャンセル（`activityType=CANCEL_MDM_MIGRATION`。`mdmServer`・metadata 不要）。
  - `devices.ActivityTypeMetadata` 型と activityType 定数 `ActivityAssignWithMdmMigrationDeadline` / `ActivityUpdateMdmMigrationDeadline` / `ActivityCancelMdmMigration` を追加。
- `devices.DeviceAttributes` に API 2.3 の読み取り専用フィールドを追加: `IsMdmMigrationCapable` / `MdmMigrationStatus` / `MdmMigrationDeadlineDateTime`。列挙値定数 `MdmMigrationStatus*`（REQUESTED / STARTED / SUCCESS / FAILED）も追加。
- アクティビティの `status` / `subStatus` 定数を公式ドキュメントの列挙値に合わせて拡充: `StatusStopped` / `StatusFailed`、`SubStatusSubmitted` / `SubStatusPreProcessing` / `SubStatusPending` / `SubStatusProcessing` / `SubStatusPostProcessing` / `SubStatusStopping` / `SubStatusCompletedWithSuccess` / `SubStatusCompletedWithFailure` / `SubStatusCompletedPostProcessingFailed`。

### Deprecated
- `devices.SubStatusCompleted`（`"COMPLETED"`）: 全件成功時の推定値だったが公式列挙に存在しない。`SubStatusCompletedWithSuccess` を使用のこと。

## [0.7.0] - 2026-07-18

### Added
- **`orgunits` パッケージを新設（Apple Business API 2.2、2026-07-15 追加の organizationalUnits）**（#27）。組織単位（OU）は users/userGroups が参照する組織階層エンティティ（`User.roleOuList` の `ouId` / `UserGroup.ouId`）で、people とは管理軸が異なるため専用パッケージとした。読み取り専用:
  - `orgunits.List`: 一覧（`GET /v1/organizationalUnits`、`limit`≤1000、カーソルページング）。
  - `orgunits.Get`: 単体取得（`GET /v1/organizationalUnits/{id}`）。
  - `orgunits.Members`: 所属ユーザーの ID 一覧（`GET /v1/organizationalUnits/{id}/relationships/users`、linkage＝`{type:users,id}`）。
  - 型 `OrganizationalUnit` / `OrganizationalUnitAttributes`（`name` / `description` / `createdDateTime` / `updatedDateTime`）。
- `examples/dump-all`: Organizational Units セクションを追加（読み取り専用の疎通確認）（#27）。
- `examples/write-test`: mdmServers CRUD（API 2.1）の実機スモークテストを追加。自己署名 X.509 証明書をその場で生成して Create→Get→Update→Delete する自己完結方式（デバイス割り当てなし）。既定ドライラン・`-keep` 対応は従来どおり（#25）。

## [0.6.0] - 2026-07-09

### Added
- **`mdmServers` のフル CRUD 対応（Apple Business API 2.1、2026-06-03 追加）**（#23）:
  - `devices.GetMdmServer`: 単体取得（`GET /v1/mdmServers/{id}`。2.0 までは 403 / GET_COLLECTION のみだった）。
  - `devices.CreateMdmServer`: 作成（`POST /v1/mdmServers`）。`serverName` と `serverCertificate`（Base64 X.509）が必須、`enableMdmDisownFlag` は任意。
  - `devices.UpdateMdmServer`: 部分更新（`PATCH /v1/mdmServers/{id}`）。`serverName` / `serverCertificate` / `enableMdmDisownFlag` / `defaultProductFamilies` を個別に変更可（未指定は不変）。
  - `devices.DeleteMdmServer`: 削除（`DELETE /v1/mdmServers/{id}`、204）。割り当てデバイスが残っている場合は削除不可（先に全デバイスの割り当て解除が必要）。
- `devices.MdmServerAttributes` に API 2.1 の新フィールドを追加: `Status` / `DeviceCount` / `EnableMdmDisownFlag` / `DefaultProductFamilies` / `LastConnectedDateTime` / `LastConnectedIP`（#23）。
- `devices.MdmServerCertificate` 型と列挙値定数 `MdmServerStatus*`（ACTIVE / INACTIVE / DELETED）・`MdmProductFamily*`（APPLE_TV / IPAD / IPHONE / IPOD / MAC / VISION / WATCH）を追加（#23）。

## [0.5.0] - 2026-07-07

### Added
- `devices.ListMdmDevices` / `devices.MdmDeviceDetails`: Apple の組み込みデバイス管理サービス（Apple MDM）に登録されたデバイスの一覧・詳細取得（API v2.0 で追加された `GET /v1/mdmDevices` / `GET /v1/mdmDevices/{id}/details`）（#21）。
- `MdmDeviceDetailAttributes` に公式 DocC の全フィールドを反映（`serialNumber` / `osVersion` / `platform` / `meid` / `wifiMacAddress` / `storageFreeCapacity` / `storageTotalCapacity` を追加）（#21）。

## [0.4.1] - 2026-06-17

### Security
- `golang.org/x/oauth2` を v0.21.0 → v0.30.0 に更新し **CVE-2025-22868（GO-2025-3488）** を解消（#9）。Go 1.23 フロアと両立する最新版（v0.31.0+ は Go 1.24 を要求するため見送り）。dependabot の ignore は誤った前提で全更新を止めていたため、`>=0.31.0` のみ skip するよう限定。
- CI を least-privilege 化（workflow レベルで `permissions: contents: read`）し、`govulncheck` ジョブを追加（#13）。

### Changed
- **リトライ方針の変更（#11）**: 非冪等な POST は 429 のみ再試行し、5xx・ネットワークエラーでは再試行せず即エラーを返すように変更（書き込み二重実行の防止）。GET/PATCH/DELETE は従来どおり 429/5xx で再試行。
- **同一オリジン制限（#10）**: `Client` が発行する全リクエスト（`links.next` 追従・リダイレクト含む）の宛先を base URL と同一の scheme+host に制限。違反時はリクエスト送信前にエラー。不正な base URL は `NewClient` が即座にエラーを返す。
- JWT クライアントアサーションの有効期限を 180 日から 10 分に短縮（#12）。毎回新規生成のため公開挙動への影響はなく、漏えい時の再利用窓を縮小。

### Added
- `applebusiness.ErrorObject`: `APIError.Errors` 要素の公開 named type（#16）。
- `APIError.RawBody`: JSON:API 形式でないエラーボディの先頭断片を保持し `Error()` にも表示（#16）。

### Fixed
- レスポンスボディを読み切ってから Close し、keep-alive 接続が再利用されるように（#15）。
- レスポンスボディの読み取りに上限を設定（成功時 32 MiB / エラー時 64 KiB）（#14）。
- `blueprints` の関係名 `rel` を既知の値（`Rel*` 定数）に検証し、不正値はリクエスト送信前にエラー（#14）。
- `auditevents.ListRange` が引数の `url.Values` を変更しないように（#14）。
- `newJTI` が乱数取得エラーを無視しないように（#14）。

## [0.3.2] - 2026-06-03

### Changed
- `README.md` を英語化（pkg.go.dev のトップに表示されるため）。日本語版は持たない。roadmap の古い記述（テスト未・実機検証未）も実態に合わせて更新。

## [0.3.1] - 2026-06-03

### Added
- コミュニティ/運用ファイル: `CONTRIBUTING.md` / `CODE_OF_CONDUCT.md` / Issue・PR テンプレート / Dependabot 設定。
- `devices` / `people` に godoc Example を追加。

### Changed
- 公開シンボル（package / 型 / 関数 / 定数 / フィールド / Example）の doc コメントを **英語化**（pkg.go.dev での可読性向上）。内部の実装コメントは日本語のまま。
- 依存更新（Dependabot）: `golang-jwt/jwt/v5` 5.3.1 / `actions/setup-go` v6 / `actions/checkout` v6。最低 Go は **1.23 を維持**（`golang.org/x/oauth2` 0.22+ と `golangci/golangci-lint-action` v7+ は要件が上がるため `dependabot.yml` で ignore）。

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
