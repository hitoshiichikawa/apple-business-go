# Apple Business API — 定義リファレンス (v1)

> 本ドキュメントは、設計・実装用にAPI定義を一枚にまとめたものです。

## 0. 出典と取得経路（重要な前提）

- Apple公式ドキュメント（`developer.apple.com/documentation/applebusinessapi`）は
  **JavaScriptレンダリングのSPA**で、本文（コード例・スキーマ）を機械的に取得できません。
  DocCのJSONデータ直取得も、こちら側のfetch制約で不可でした。
- そこで代替として、**このAPIを型付きで実装した公開Goリファレンス**から定義を抽出しています:
  - `micromdm/nanoaxm` (Go) … OAuth実装・資格情報フォーマット
  - `neilmartin83/terraform-provider-axm` (Go) … 全エンドポイント＋エンティティ型＋OAuth
- **信頼度**: 認証フローの主要値は上記2実装で**一致**を確認済み。ただし二次情報由来のため、
  本番投入前にApple公式の各ページで最終確認を推奨（特に下記「要確認」項目）。

---

## 1. 接続情報

| 項目 | 値 |
|---|---|
| ベースURL (ABM) | `https://api-business.apple.com` |
| ベースURL (ASM) | `https://api-school.apple.com` |
| バージョンprefix | `/v1` |
| レート上限 | 100 req/s |
| ページネーション | カーソル方式。`meta.paging.nextCursor` / `links.next`。既定 `limit=100` |
| レスポンス形式 | JSON:API 風（`data` / `links` / `meta`、リレーションは `relationships`） |
| 認可ヘッダ | `Authorization: Bearer <access_token>` |

---

## 2. 認証（OAuth 2.0 / client_credentials + JWT client assertion）

### 2.1 資格情報（ABM/ASMポータルで発行）

| 項目 | 説明 | 例 |
|---|---|---|
| `client_id` | API口座のClient ID | `BUSINESSAPI.f6cb33e8-51b3-4d8c-a041-5952c4e18851` |
| `key_id` | 鍵ID | `b2100d09-ccd1-45db-9f7b-ee362dc6be6a` |
| `private_key` | ポータルからDLする秘密鍵（`.pem`、EC P-256 / ES256用） | — |
| `team_id` | issuer。**AxMでは `client_id` と同一に設定するのが通例** | — |

> 発行できるのは Organization Administrator のみ。API口座は最大50個。
> 専用カスタムロールは「Device API Manager」。

### 2.2 Step 1 — client assertion (JWT) を生成

- 署名アルゴリズム: **ES256**
- ヘッダ: `{ "alg": "ES256", "kid": "<key_id>" }`
- クレーム:

| claim | 値 |
|---|---|
| `iss` | `team_id`（AxMでは `client_id` と同一に設定する実装が一般的） |
| `sub` | `client_id` |
| `aud` | `https://account.apple.com/auth/oauth2/v2/token` |
| `iat` | 現在時刻(UNIX秒) |
| `exp` | `iat` + 最大 **180日** |
| `jti` | UUID v4（リプレイ防止の一意値） |

### 2.3 Step 2 — トークン交換

```
POST https://account.apple.com/auth/oauth2/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
client_id=<client_id>
client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
client_assertion=<上で生成したJWT>
scope=<business.api | school.api>
```

| 項目 | 値 |
|---|---|
| `scope` | `business.api`（ABM）/ `school.api`（ASM）。`client_id` が `BUSINESSAPI.` 始まりなら business |
| レスポンス | `access_token`, `token_type=Bearer`, `expires_in=3600`（=1時間） |

### 2.4 注意・要確認（2実装の差異）

- **`aud` クレーム**: 両実装とも `.../auth/oauth2/v2/token` で一致（`v2` 入り）。
- **POST先**: 実装差あり。`nanoaxm` は `.../auth/oauth2/v2/token`、`terraform-provider-axm` は
  `.../auth/oauth2/token`（`v2` なし）。→ **`v2/token` を第一候補**とし、公式OAuthページで最終確認。
- **`iss`**: `nanoaxm` は `iss=sub=client_id`、`terraform-provider-axm` は `iss=team_id`（別フィールド）。
  AxMでは `team_id == client_id` が通例のため、まずは両方 `client_id` で可。
- assertionは最大180日キャッシュ可。`access_token` は1時間ごとに再取得。

---

## 3. エンドポイント一覧（v1）

> すべて `Authorization: Bearer` 必須。パスは ベースURL + 下記。

### 3.1 ユーザー / グループ
| メソッド | パス | 用途 |
|---|---|---|
| GET | `/v1/users` | ユーザー一覧 |
| GET | `/v1/users/{id}` | ユーザー詳細 |
| GET | `/v1/userGroups` | ユーザーグループ一覧 |
| GET | `/v1/userGroups/{id}` | ユーザーグループ詳細 |
| GET | `/v1/userGroups/{id}/relationships/users` | グループ所属ユーザーID |
| GET | `/v1/organizationalUnits` | 組織単位一覧（**API 2.2+**） |
| GET | `/v1/organizationalUnits/{id}` | 組織単位詳細（**API 2.2+**） |
| GET | `/v1/organizationalUnits/{id}/relationships/users` | 組織単位所属ユーザーID（**API 2.2+**） |

### 3.2 デバイス
| メソッド | パス | 用途 |
|---|---|---|
| GET | `/v1/orgDevices` | デバイス一覧 |
| GET | `/v1/orgDevices/{id}` | デバイス詳細 |
| GET | `/v1/orgDevices/{id}/relationships/assignedServer` | 割り当て先MDM（リレーション/IDのみ） |
| GET | `/v1/orgDevices/{id}/assignedServer` | 割り当て先MDM（詳細情報） |
| GET | `/v1/orgDevices/{id}/appleCareCoverage` | AppleCare保証情報 |

### 3.3 MDMサーバ（デバイス管理サービス）
| メソッド | パス | 用途 |
|---|---|---|
| GET | `/v1/mdmServers` | MDMサーバ一覧 |
| GET | `/v1/mdmServers/{id}` | MDMサーバ単体取得（**API 2.1+**。2.0 までは 403） |
| POST | `/v1/mdmServers` | MDMサーバ作成（**API 2.1+**、201。`serverName`+`serverCertificate` 必須）。**書き込み系** |
| PATCH | `/v1/mdmServers/{id}` | MDMサーバ部分更新（**API 2.1+**、200）。**書き込み系** |
| DELETE | `/v1/mdmServers/{id}` | MDMサーバ削除（**API 2.1+**、204。割り当てデバイスが残ると不可）。**書き込み系** |
| GET | `/v1/mdmServers/{id}/relationships/devices` | サーバ配下のデバイス（シリアル/ID） |

### 3.4 アクティビティ（割り当て/解除）
| メソッド | パス | 用途 |
|---|---|---|
| POST | `/v1/orgDeviceActivities` | 割り当て/解除アクティビティ作成（バッチ）。**書き込み系** |
| GET | `/v1/orgDeviceActivities/{id}` | アクティビティの実行ステータス確認 |

### 3.5 アプリ / パッケージ / ブループリント / 構成 / 監査
| メソッド | パス | 用途 |
|---|---|---|
| GET | `/v1/apps` | アプリ一覧 |
| GET | `/v1/apps/{id}` | アプリ詳細 |
| GET | `/v1/packages` | パッケージ一覧 |
| GET | `/v1/packages/{id}` | パッケージ詳細 |
| GET | `/v1/blueprints` | ブループリント一覧 |
| GET | `/v1/blueprints/{id}` | ブループリント詳細 |
| GET | `/v1/blueprints/{id}/relationships/apps` | ブループリント関連アプリ |
| GET | `/v1/configurations` | 構成一覧 |
| GET | `/v1/configurations/{id}` | 構成詳細 |
| GET | `/v1/auditEvents` | 監査イベント（範囲・絞り込みクエリあり） |

`auditEvents` のクエリ: `start_timestamp`, `end_timestamp`（ISO8601, 必須）, `actor_id`, `subject_id`, `event_type`, `cursor`, `limit`, `fields`。

---

## 4. 主要エンティティのフィールド定義

### 4.1 User（`/v1/users`）
| フィールド | 型 | 説明 |
|---|---|---|
| `id` | String | ユーザーID |
| `managed_apple_account` | String | Managed Apple Account |
| `email` | String | メールアドレス |
| `first_name` / `middle_name` / `last_name` | String | 氏名 |
| `job_title` | String | 役職 |
| `department` / `division` / `cost_center` | String | 部門 / 部署 / コストセンター |
| `employee_number` | String | 従業員番号 |
| `phone_numbers` | List<{`phone_number`,`type`}> | 電話番号 |
| `is_external_user` | Boolean | 外部ユーザーか |
| `status` | String | ステータス |
| `role_ou_list` | List<{`ou_id`,`role_name`}> | ロールと組織単位の対応 |
| `start_date_time` / `created_date_time` / `updated_date_time` | String | 各日時 |
| `type` | String | リソース種別 |

### 4.2 User Group（`/v1/userGroups`）
| フィールド | 型 | 説明 |
|---|---|---|
| `id` | String | グループID |
| `name` | String | グループ名 |
| `group_type` | String | グループ種別 |
| `ou_id` | String | 組織単位ID |
| `status` | String | ステータス |
| `total_member_count` | Number | メンバー数 |
| `user_ids` | List<String> | 所属ユーザーID |
| `created_date_time` / `updated_date_time` | String | 各日時 |
| `type` | String | リソース種別 |

### 4.2.1 Organizational Unit（`/v1/organizationalUnits`、API 2.2）
| フィールド | 型 | 説明 |
|---|---|---|
| `id` | String | 組織単位ID |
| `name` | String | 組織単位名 |
| `description` | String | 説明 |
| `created_date_time` / `updated_date_time` | String | 各日時 |
| `type` | String | `organizationalUnits` |

所属ユーザーは `GET /v1/organizationalUnits/{id}/relationships/users`（linkage＝`{type:users,id}`）で取得。

### 4.3 Org Device（`/v1/orgDevices`）
| フィールド | 型 | 説明 |
|---|---|---|
| `id` | String | 不透明なリソースID |
| `serial_number` | String | シリアル番号 |
| `device_model` | String | モデル名 |
| `product_family` | String | iPhone / iPad / Mac / AppleTV / Watch / Vision |
| `product_type` | String | 例: iPhone14,3 / iPad13,4 / MacBookPro14,2 |
| `color` / `device_capacity` | String | 色 / 容量 |
| `part_number` / `order_number` | String | 部品番号 / 注文番号 |
| `order_date_time` / `added_to_org_date_time` / `released_from_org_date_time` | String | 注文日時 / 組織追加日時 / 解放日時(未解放はnull) |
| `status` | String | **ASSIGNED / UNASSIGNED**（ASSIGNEDなら別APIで割り当て先取得） |
| `wifi_mac_address` / `bluetooth_mac_address` | String | MACアドレス |
| `ethernet_mac_address` | List<String> | 内蔵EthernetのMAC |
| `imei` / `meid` | List<String> | IMEI / MEID（あれば） |
| `eid` | String | EID（あれば） |
| `purchase_source_id` / `purchase_source_type` | String | 購入元ID / 種別（Apple Customer / Reseller） |
| `updated_date_time` | String | 最終更新日時 |
| `type` | String | リソース種別 |

### 4.4 MDM Server（`/v1/mdmServers`）
| フィールド | 型 | 説明 |
|---|---|---|
| `id` | String | リソースID |
| `server_name` | String | サービス名 |
| `server_type` | String | **MDM / APPLE_CONFIGURATOR / APPLE_MDM**（読み取り専用） |
| `status` | String | **ACTIVE / INACTIVE / DELETED**（API 2.1+、読み取り専用） |
| `device_count` | Integer | 割り当てデバイス数（API 2.1+、読み取り専用） |
| `enable_mdm_disown_flag` | Boolean | デバイスの disown 許可（API 2.1+） |
| `default_product_families` | [String] | 既定割り当ての製品ファミリ **APPLE_TV / IPAD / IPHONE / IPOD / MAC / VISION / WATCH**（API 2.1+、更新可） |
| `last_connected_date_time` / `last_connected_ip` | String | 最終接続日時 / 接続元IP（API 2.1+、読み取り専用） |
| `created_date_time` / `updated_date_time` | String | 各日時 |
| `type` | String | `mdmServers` |

作成/更新の証明書は `serverCertificate: { name(String!), data(String!=Base64 X.509) }`。

### 4.5 Assigned Server Info（`/v1/orgDevices/{id}/assignedServer`）
`device_id`, `server_id`, `server_name`, `server_type`(MDM/APPLE_CONFIGURATOR/APPLE_MDM),
`created_date_time`, `updated_date_time`。

### 4.6 Audit Event（`/v1/auditEvents`）
`id`, `actor_id` / `actor_name` / `actor_type`, `subject_id` / `subject_name` / `subject_type`,
`category`, `event_type`, `outcome`, `event_date_time`, `group_id`,
`event_data_json`（イベント固有ペイロード）, `type`。

---

## 5. 3ペインUIへのマッピング案

| 左ペイン（種別） | 中央（一覧API） | 右ペイン（詳細API） |
|---|---|---|
| Users | `GET /v1/users` | `GET /v1/users/{id}`（＋所属グループ） |
| User Groups | `GET /v1/userGroups` | `GET /v1/userGroups/{id}` ＋ `/relationships/users` |
| Organizational Units | `GET /v1/organizationalUnits` | `GET /v1/organizationalUnits/{id}` ＋ `/relationships/users` |
| Devices | `GET /v1/orgDevices` | `GET /v1/orgDevices/{id}` ＋ `/assignedServer` ＋ `/appleCareCoverage` |
| MDM Servers | `GET /v1/mdmServers` | `/v1/mdmServers/{id}/relationships/devices`（配下端末） |
| Activities | `GET /v1/orgDeviceActivities/{id}` | 同左（ステータス） |
| Audit Events | `GET /v1/auditEvents` | イベント詳細（行展開） |

---

## 6. 制約・注意

- **トークンはフルアクセス**（エンドポイント単位の権限分離なし）。`.pem` を持つ＝そのテナントで可能な全操作が可能。
  保管は暗号化必須、フロントには絶対に出さない。
- 旧来の制約として「デバイスのrelease」「移行期限の設定」「MDMトークンの自動更新」はAPI不可とされてきた。
  ただしAPIは拡張が続いている（users/auditEvents等が追加済み）ため、最新可否は公式で要再確認。
- 本書のフィールド・挙動は公開Go実装（MPL-2.0等）由来の**実測・推定**。確定はApple公式の各エンドポイントページで。

---

## 7. Blueprints / Configurations（組み込みデバイス管理）— 確定版

2026年4月の統合で組み込みMDMが無料化・統合され、**Configurations** と **Blueprints** が利用可能。
Configurations はセキュリティ/ネットワーク等の設定単位、Blueprints は **apps・configurations・packages を束ねて
デバイス/ユーザー/グループへ割り当てる**単位（「一度作れば全体へ展開」）。デバイスへの割り当ては Blueprint の
`orgDevices`（または `users` / `userGroups`）リレーションへの追加で行う（MDMの `orgDeviceActivities` とは別系統）。

> 以下のエンドポイント・リクエストボディ・ステータス・`type` リテラルは、フルCRUDを実装した公開Goリファレンス
> （`terraform-provider-axm`）から確定したもの。実装準拠で信頼度は高いが、最終確認は Apple 公式ページ推奨。

### 7.1 Blueprint — 読み取り

| メソッド | パス | 返却 |
|---|---|---|
| GET | `/v1/blueprints`（`limit` / `cursor`） | Blueprint 配列（カーソルページング） |
| GET | `/v1/blueprints/{id}` | Blueprint 単体 |
| GET | `/v1/blueprints/{id}/relationships/{rel}` | 関連リソースID配列（ページング）。`rel` = `apps` / `configurations` / `packages` / `orgDevices` / `users` / `userGroups` |

- 属性: `name` / `description` / `status` / `appLicenseDeficient` / `createdDateTime` / `updatedDateTime`。
- 注: 権限や状態によっては `GET .../relationships/{rel}` が 403 を返すことがある（リファレンス実装は割り当て状態を
  自前で保持して回避している）。アプリ側で割り当てを保持する設計が無難。

### 7.2 Blueprint — 操作系（書き込み・確定）

**作成** `POST /v1/blueprints` → 200 / 201

```json
{
  "data": {
    "type": "blueprints",
    "attributes": { "name": "Engineering Onboarding", "description": "..." },
    "relationships": {
      "apps":           { "data": [ { "type": "apps", "id": "361309726" } ] },
      "configurations": { "data": [ { "type": "configurations", "id": "config-123" } ] },
      "packages":       { "data": [ { "type": "packages", "id": "pkg-123" } ] },
      "orgDevices":     { "data": [ { "type": "orgDevices", "id": "ABC123" } ] },
      "users":          { "data": [ { "type": "users", "id": "100" } ] },
      "userGroups":     { "data": [ { "type": "userGroups", "id": "grp-1" } ] }
    }
  }
}
```
- `attributes.name` 必須、`description` 任意。
- ⚠️ **実機（2026-06 時点）では `relationships` も必須**: 「中身」(`apps`/`packages`/`configurations`) と
  「割り当て先」(`orgDevices`/`users`/`userGroups`) を**各カテゴリ最低1つずつ**指定する必要がある。欠けると 409
  （中身なし=`MISSING_RESOURCES` / 割り当て先なし=`MISSING_MEMBERS`）。作成と割り当てを分離できない点に注意。`examples/write-test` で確認。
- ⚠️ `name` はスペース・括弧など不可（英数字・ハイフン等のみ。違反は 409 `ENTITY_ERROR.ATTRIBUTE.INVALID`）。
  Configuration の `name` はスペース・括弧可で、リソースごとに名前制約が異なる。

**更新** `PATCH /v1/blueprints/{id}` → 200。`data.type` = `blueprints`、`data.id` 必須。`attributes`（`name` / `description`）は変更分のみ。

**削除** `DELETE /v1/blueprints/{id}` → 204 No Content。

**割り当ての追加 / 削除（リレーション操作）** `{method} /v1/blueprints/{id}/relationships/{rel}` → 204 / 200

```json
{ "data": [ { "type": "orgDevices", "id": "ABC123" }, ... ] }
```
- `POST` = 追加、`DELETE` = 削除、`PATCH` = 集合の置換。
- `{rel}` と各要素の `type` は対応（`orgDevices` / `users` / `userGroups` / `apps` / `configurations` / `packages`）。
- 運用: 現在の集合と目標の集合を差分し、追加分を `POST`・削除分を `DELETE`。

### 7.3 Configuration — 読み取り / 操作系（確定）

属性: `type`（= 構成種別。`AIR_DROP` / `AUTHENTICATION_SCREEN_LOCK` / `CUSTOM_SETTING` 等）/ `name` /
`configuredForPlatforms[]`（`PLATFORM_IOS` / `PLATFORM_MACOS` 等）/
`customSettingsValues` { `configurationProfile`（.mobileconfig XML）, `filename` }（`CUSTOM_SETTING` のみ）/ 日時。

| メソッド | パス | ステータス |
|---|---|---|
| GET | `/v1/configurations`, `/v1/configurations/{id}` | 200 |
| POST | `/v1/configurations` | 200 / 201 |
| PATCH | `/v1/configurations/{id}` | 200 |
| DELETE | `/v1/configurations/{id}` | 204 |

**作成（`CUSTOM_SETTING` の例）** `POST /v1/configurations`

```json
{
  "data": {
    "type": "configurations",
    "attributes": {
      "type": "CUSTOM_SETTING",
      "name": "Wi-Fi Configuration",
      "configuredForPlatforms": [ "PLATFORM_IOS", "PLATFORM_MACOS" ],
      "customSettingsValues": {
        "configurationProfile": "<?xml ... .mobileconfig の中身 ...>",
        "filename": "WiFi.mobileconfig"
      }
    }
  }
}
```
- 外側 `data.type` は JSON:API リソース種別 `configurations`、`attributes.type` は構成種別（`CUSTOM_SETTING`）。
- `CUSTOM_SETTING` では `configurationProfile`（.mobileconfig の中身＝**生 XML**。Base64 化は不可）と `filename` が必須。
  `PayloadContent` は**非空**（最低1ペイロード）でないと 400。`examples/write-test` で確認。
- 更新は `data.id` 指定で変更分の `attributes` を送る（`name` はポインタ＝変更分のみ）。

### 7.4 SDK へのマッピング（実装予定）

| 操作 | SDKメソッド（案） |
|---|---|
| `blueprints.List` / `Get` / `Relationship(rel)` | 読み取り |
| `blueprints.Create` / `Update` / `Delete` | 操作系 |
| `blueprints.AddTo(rel, ids)` / `RemoveFrom(rel, ids)` / `Replace(rel, ids)` | 割り当て |
| `configurations.List` / `Get` / `Create` / `Update` / `Delete` | 構成 |

---

## 8. Apps / Packages / Audit Events（公式DocC 確認済み）

出典: `developer.apple.com/documentation/applebusinessapi`（JSレンダリングのためブラウザの DocC データで確認）。

### 8.1 App（`/v1/apps`）
`App = { type, id, attributes: App.Attributes, links }`
App.Attributes: `name`(string) / `bundleId`(string) / `version`(string) /
**`supportedOS`（`[SupportedOS]` = 文字列配列）** / `isCustomApp`(boolean) / `appStoreUrl`(string) / `websiteUrl`(string)
※ `supportedOS` は配列。SDK は `[]string`。

### 8.2 Package（`/v1/packages`）
Package.Attributes: `name` / `url` / `hash` / `bundleIds`(`[string]`) / `description` / `version` /
`createdDateTime`(date-time) / `updatedDateTime`(date-time) — SDK と一致。

### 8.3 Audit Events（`/v1/auditEvents`）
`AuditEvent = { type, id, attributes }`。`attributes` は **共通エンベロープ `AuditEventCommonAttributes`** ＋
**イベント固有ペイロード `eventData<Event>`** の組み合わせ（DocC 確認済み）。
- 共通: `eventDateTime` / `type`(AuditEventType) / `category` / `actorType`/`actorId`/`actorName` /
  `subjectType`/`subjectId`/`subjectName` / `outcome` / `groupId` / `eventDataPropertyKey`
- 固有: `eventDataPropertyKey` が示すキー（例 `eventDataDeviceAssignedToServer`）にイベント固有データ。
  例: `AuditEventDeviceAssignedToServer` = `{ serialNumber, targetServerName }`。
- `AuditEventType` は33種。各 eventData のフィールドは `docs/apple-business-api-datatypes.md` §4 に集約。
- クエリ: **`filter[startTimestamp]` が必須**（未指定で 400）。`filter[endTimestamp]` と併用、ISO8601。SDK は `ListRange` で付与。
→ SDK `auditevents.Attributes` は共通項目を型付きで保持し、`EventData`(生JSON) を `Payload(&v)` で個別型へデコード。
  主要イベント型（`DeviceAssignedToServer` 等）を同梱。
（前回「純ユニオン」と記載したが、`AuditEventCommonAttributes` の存在を確認し訂正。参照実装のフラットな
`actorId`/`actorName` はこの共通エンベロープに相当し、現行公式とも整合する。）

---

## 付録: 参照した公開実装

- `github.com/micromdm/nanoaxm` — Go。OAuth・資格情報フォーマット。
- `github.com/neilmartin83/terraform-provider-axm` — Go。全エンドポイント＋型＋OAuth＋各エンドポイントのサンプル。
- 参考: `rodchristiansen/asbmutil`(Swift), `karthikeyan-mac/AppleBusinessANDSchoolManagerAPI`(Python), `EUCTechTopics/PSABM`(PowerShell)。
