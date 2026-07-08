# Apple Business API — データ型リファレンス（公式 DocC 確認済み）

出典: `developer.apple.com/documentation/applebusinessapi`（JSレンダリングのため DocC データを取得して確認）。
本書は開発時の参照用。型は JSON:API 表記、`!` は必須、`[T]` は配列、`byte` は Base64 エンコード文字列、`date-time` は ISO8601。

カテゴリ件数: Objects and Data Types 67 / Type Aliases 21 / Dictionaries 69（＝監査イベント機構）。

---

## 1. Type Aliases（列挙値）— 21件

| 型 | 値 |
|---|---|
| `OrgDeviceActivityType` | `ASSIGN_DEVICES`, `UNASSIGN_DEVICES` |
| `SupportedOS` | `SUPPORTED_OS_UNSPECIFIED`, `SUPPORTED_OS_IPADOS`, `SUPPORTED_OS_IOS`, `SUPPORTED_OS_MACOS`, `SUPPORTED_OS_TVOS`, `SUPPORTED_OS_WATCHOS`, `SUPPORTED_OS_VISIONOS` |
| `AppleCareCoverageStatus` | `ACTIVE`, `INACTIVE` |
| `AppleCareCoveragePaymentType` | `ABE_SUBSCRIPTION`, `PAID_UP_FRONT`, `SUBSCRIPTION`, `NONE` |
| `UserStatus` | `NEW`, `RELEASED`, `ACTIVE`, `DEACTIVATED`, `LOCKED`, `LOCKED_FOR_SHARED_IPAD` |
| `UserPhoneNumberType` | `WORK`, `HOME`, `MOBILE` |
| `UserGroupStatus` | `ACTIVE` |
| `UserGroupType` | `STANDARD`, `SMART` |
| `BlueprintStatus` | `ACTIVE`, `TO_BE_DELETED` |
| `ConfigurationType` | `AIR_DROP`, `AIR_PRINT`, `APP_ACCESS`, `APPLE_INTELLIGENCE_SIRI`, `APPLICATION_LAYER_FIREWALL`, `AUTHENTICATION_SCREEN_LOCK`, `CERTIFICATE`, `CONFERENCE_ROOM_DISPLAY`, `CONTENT_CACHING`, `CUSTOM_PROFILE`, `CUSTOM_SETTING`, `DATA_MANAGEMENT`, `ENERGY_SAVER`, `FILE_VAULT`, `GATE_KEEPER`, `ICLOUD`, `LOGIN_WINDOW`, `MEDIA_MANAGEMENT`, `SOFTWARE_UPDATE`, `VPN`, `WEB_CLIP`, `WEB_FILTER`, `WIFI` |
| `ConfigurationPlatform` | `PLATFORM_MACOS`, `PLATFORM_IOS`, `PLATFORM_TVOS`, `PLATFORM_VISIONOS` |
| `DeviceEraseStatus` | `NOT_ERASED`, `ERASED` |
| `DeviceLockStatus` | `LOCKED`, `UNLOCKED` |
| `LostModeStatus` | `ENABLED`, `DISABLED` |
| `AuditEventActorType` | `USER`, `API_USER`, `SYSTEM` |
| `AuditEventSubjectType` | `ORGANIZATION`, `USER`, `LOCATION`, `DEVICE`, `COLLECTION`, `DEVICE_MANAGEMENT_SETTING`, `SUBSCRIPTION`, `DOMAIN`, `API_USER` |
| `AuditEventCategory` | `ORGANIZATION`, `ACCOUNT_ACTIVITY`, `DEVICE_INVENTORY`, `PURCHASING`, `DEVICE_MANAGEMENT` |
| `AuditEventOutcome` | `SUCCESS`, `FAILURE` |
| `AuditEventPurchaseSourceType` | `APPLE`, `MANUALLY_ADDED`, `RESELLER` |
| `AuditEventReleaseEntityType` | `USER`, `MDM`, `RESELLER`, `API`, `REPLACEMENT` |
| `AuditEventType` | §4.2 を参照（33種） |

---

## 2. リソース属性（Attributes）— フィールド定義

各リソースは `{ type, id, attributes, links, relationships? }`。以下は `attributes` の中身。

### OrgDevice.Attributes（`/v1/orgDevices`）
`addedToOrgDateTime`(date-time), `releasedFromOrgDateTime`(date-time), `color`(string), `deviceCapacity`(string),
`deviceModel`(string), `eid`(string), `imei`([string]), `meid`([string]), `wifiMacAddress`(string),
`bluetoothMacAddress`(string), `ethernetMacAddress`([string]), `orderDateTime`(date-time), `orderNumber`(string),
`partNumber`(string), `productFamily`(string), `productType`(string), `purchaseSourceType`(string),
`purchaseSourceId`(string), `serialNumber`(string), `status`(string), `updatedDateTime`(date-time),
`releaserEntityType`(string), `releaserId`(string)

### MdmDevice.Attributes（`/v1/mdmDevices`）
> Apple の組み込みデバイス管理サービス（Apple MDM）に登録されたデバイス。`GET /v1/mdmDevices`（クエリ: `fields[mdmDevices]`, `limit`）で一覧取得。SDK は `ListMdmDevices`。
`deviceName`(string), `enrolledUserId`(string), `productFamily`(string), `serialNumber`(string)

### MdmDeviceDetail.Attributes（`/v1/mdmDevices/{id}/details`）
> `GET /v1/mdmDevices/{id}/details`（クエリ: `fields[mdmDeviceDetails]`）で取得。SDK は `MdmDeviceDetails`。
`bluetoothMacAddress`(string), `deviceEraseStatus`(DeviceEraseStatus), `deviceLockStatus`(DeviceLockStatus),
`deviceModel`(string), `deviceName`(string), `ethernetMacAddress`(string), `imei`([string]),
`isFileVaultEnabled`(boolean), `isFirewallEnabled`(boolean), `lastCheckInDateTime`(date-time),
`lostModeStatus`(LostModeStatus), `meid`([string]), `osVersion`(string), `platform`(string),
`serialNumber`(string), `storageFreeCapacity`(integer), `storageTotalCapacity`(integer), `wifiMacAddress`(string)

### MdmServer.Attributes（`/v1/mdmServers`＝Device Management Services）
> **API 2.1（2026-06-03）でフル CRUD 化**: `GET /v1/mdmServers/{id}`（単体取得。2.0 までは 403 だった）/ `POST /v1/mdmServers`（作成、201）/ `PATCH /v1/mdmServers/{id}`（部分更新、200）/ `DELETE /v1/mdmServers/{id}`（204。**割り当てデバイスが残っていると削除不可**）。SDK は `GetMdmServer` / `CreateMdmServer` / `UpdateMdmServer` / `DeleteMdmServer`。
> 作成は `serverName`(string!) + `serverCertificate`(MdmServerCertificate!) が必須、`enableMdmDisownFlag`(boolean) は任意。更新はすべて任意で、加えて `defaultProductFamilies`([MdmServerProductFamily]) を変更可。
> `MdmServerCertificate`: `name`(string!)＝ファイル名, `data`(string!)＝Base64 の X.509 証明書。
> 割り当てデバイス: `GET /v1/mdmServers/{id}/relationships/devices` のみ可（linkage＝`{type:orgDevices,id}` のID一覧）。related の `GET /v1/mdmServers/{id}/devices` は **`GET_RELATED` 不可（403, allowed: GET_RELATIONSHIP）**。フル属性は各 `orgDevices/{id}` を個別取得。SDK は `MdmServerDevices`（ID）/ `MdmServerDeviceList`（ID→個別Getでフル）。
`createdDateTime`(date-time), `defaultProductFamilies`([MdmServerProductFamily]), `deviceCount`(integer, RO),
`enableMdmDisownFlag`(boolean), `lastConnectedDateTime`(date-time, RO), `lastConnectedIp`(string, RO),
`serverName`(string), `serverType`(string, RO), `status`(MdmServerStatus, RO), `updatedDateTime`(date-time)
> 列挙値 — `MdmServerStatus`: `ACTIVE` | `INACTIVE` | `DELETED` / `MdmServerProductFamily`: `APPLE_TV` | `IPAD` | `IPHONE` | `IPOD` | `MAC` | `VISION` | `WATCH`

### OrgDeviceActivity.Attributes（`/v1/orgDeviceActivities`）
`createdDateTime`(date-time), `status`(string), `subStatus`(string), `completedDateTime`(date-time), `downloadUrl`(string)
> 観測値（実API）: `status` は `IN_PROGRESS`→`COMPLETED`（部分失敗でも COMPLETED）。`subStatus` は `COMPLETED_WITH_ERROR`（一部失敗。`downloadUrl` の CSV に明細）。全件成功時の `subStatus` はおそらく `COMPLETED`（未観測）。
> `COMPLETED_WITH_ERROR` の代表例: **既に同じ MDM サーバへ割り当て済みのデバイスを再割り当て**した場合（状態は変わらずエラー計上）。事前に `assignedServer` で現在の割り当て先を確認すると無駄なエラーを避けられる。

### AppleCareCoverage.Attributes
`status`(AppleCareCoverageStatus), `paymentType`(AppleCareCoveragePaymentType), `description`(string),
`startDateTime`(date-time), `endDateTime`(date-time), `isRenewable`(boolean), `isCanceled`(boolean),
`contractCancelDateTime`(date-time), `agreementNumber`(string)

### User.Attributes（`/v1/users`）
`firstName`(string), `middleName`(string), `lastName`(string), `status`(UserStatus), `managedAppleAccount`(string),
`isExternalUser`(boolean), `roleOuList`([UserRoleOuMapping]), `email`(string), `employeeNumber`(string),
`costCenter`(string), `division`(string), `department`(string), `jobTitle`(string),
`phoneNumbers`([UserPhoneNumber]), `startDateTime`(date-time), `createdDateTime`(date-time), `updatedDateTime`(date-time)

### UserGroup.Attributes（`/v1/userGroups`）
`ouId`(string), `name`(string), `type`(UserGroupType), `totalMemberCount`(integer), `status`(UserGroupStatus),
`createdDateTime`(date-time), `updatedDateTime`(date-time)

### App.Attributes（`/v1/apps`）
`name`(string), `bundleId`(string), `version`(string), `supportedOS`([SupportedOS] = 配列), `isCustomApp`(boolean),
`appStoreUrl`(string), `websiteUrl`(string)

### Package.Attributes（`/v1/packages`）
`name`(string), `url`(string), `hash`(string), `bundleIds`([string]), `description`(string), `version`(string),
`createdDateTime`(date-time), `updatedDateTime`(date-time)

### Blueprint.Attributes（`/v1/blueprints`）
`name`(string), `description`(string), `status`(BlueprintStatus), `appLicenseDeficient`(boolean),
`createdDateTime`(date-time), `updatedDateTime`(date-time)

### Configuration の attributes（`/v1/configurations`）
- `ConfigurationCommon`: `type`(ConfigurationType!), `name`(string), `configuredForPlatforms`([ConfigurationPlatform]),
  `createdDateTime`(date-time), `updatedDateTime`(date-time)
- `ConfigurationCustomSetting`（CUSTOM_SETTING）: 上記 + `customSettingsValues`(CustomSettingsValues)
- `CustomSettingsValues`: `configurationProfile`(.mobileconfig の中身＝**生 XML 文字列**), `filename`(string)
  - ⚠️ DocC スキーマは `byte`(Base64) と表記するが、**実機は生 XML をそのまま要求**（Base64 化すると 400 `plist type mismatch`）。
    `PayloadContent` も**非空必須**（空配列は 400）。`examples/write-test` で確認。

### 補助オブジェクト
- `UserPhoneNumber`: `phoneNumber`(string), `type`(UserPhoneNumberType)
- `UserRoleOuMapping`: `roleName`(string), `ouId`(string)

---

## 3. Objects and Data Types（67件・一覧）

### リソースオブジェクト
`OrgDevice` / `MdmDevice` / `MdmDeviceDetail` / `MdmServer` / `OrgDeviceActivity` / `AppleCareCoverage` /
`User` / `UserGroup` / `App` / `Package` / `Blueprint` / `Configuration`
各 `{ type!, id!, attributes, links(ResourceLinks), relationships? }`。

### リクエストボディ
| 型 | 用途 |
|---|---|
| `OrgDeviceActivityCreateRequest` | デバイス割り当て/解除アクティビティ作成 |
| `BlueprintCreateRequest` / `BlueprintUpdateRequest` | Blueprint 作成 / 更新 |
| `Blueprint{Apps,Configurations,Packages,OrgDevices,Users,UserGroups}LinkagesRequest` | リレーション割り当て（`{data:[{type,id}]}`） |
| `ConfigurationCreateRequest` / `ConfigurationUpdateRequest` | Configuration 作成 / 更新 |

### レスポンスラッパー（JSON:API 定型）
- 単体: `<Resource>Response` = `{ data(Resource!), links(DocumentLinks!) }`（MdmServer 系は `included([OrgDevice])` を含む）
- 複数: `<Resource>sResponse` = `{ data([Resource]!), links(PagedDocumentLinks!), meta(PagingInformation) }`
- リンク（リレーション）: `*LinkagesResponse` / `OrgDeviceAssignedServerLinkageResponse`
- 該当: `OrgDeviceResponse`, `OrgDevicesResponse`, `MdmDeviceResponse`, `MdmDevicesResponse`, `MdmDeviceDetailResponse`,
  `MdmServerResponse`, `MdmServersResponse`, `MdmServerDevicesLinkagesResponse`, `OrgDeviceActivityResponse`,
  `AppleCareCoverageResponse`, `UserResponse`, `UsersResponse`, `UserGroupResponse`, `UserGroupsResponse`,
  `UserGroupUsersLinkagesResponse`, `AppResponse`, `AppsResponse`, `PackageResponse`, `PackagesResponse`,
  `BlueprintResponse`, `BlueprintsResponse`, `Blueprint*LinkagesResponse`, `ConfigurationResponse`, `ConfigurationsResponse`

### 共通（リンク / ページング / エラー）
- `ResourceLinks`: `self`(uri)
- `DocumentLinks`: `self`(uri!)
- `PagedDocumentLinks`: `first`, `next`, `self`(uri!) — `next` がカーソルページングの次URL
- `RelationshipLinks`: `include`, `related`, `self`(uri)
- `PagingInformation`: `paging`(PagingInformation.Paging!)（`total` / `limit` / `nextCursor` 等）
- `ErrorResponse`: `errors`([ErrorResponse.Errors])、`ErrorLinks`, `JsonPointer`(`pointer`!), `Parameter`(`parameter`!)

---

## 4. Audit Events（`/v1/auditEvents`）

> **クエリ（必須）**: `filter[startTimestamp]` が必須（未指定だと 400 `PARAMETER_ERROR.REQUIRED`）。通常は `filter[endTimestamp]` と併用し、いずれも ISO8601(RFC3339, UTC)。SDK は `auditevents.ListRange(start, end, q)` で付与。

「Dictionaries」69件はすべて監査イベント機構（基底 `AuditEvent` + 各 `AuditEvent<Event>Attributes` ラッパー + 各 `AuditEvent<Event>` ペイロード）。

### 4.1 構造
```
AuditEvent = { type, id, attributes }
attributes = AuditEventCommonAttributes（共通）+ eventData<Event>（イベント固有、キー名は eventDataPropertyKey）
```

**AuditEventCommonAttributes（共通エンベロープ）**:
`eventDateTime`(date-time), `type`(AuditEventType!), `category`(AuditEventCategory), `actorType`(AuditEventActorType),
`actorId`(string), `actorName`(string), `subjectType`(AuditEventSubjectType), `subjectId`(string), `subjectName`(string),
`outcome`(AuditEventOutcome), `groupId`(string), `eventDataPropertyKey`(string)

> `eventDataPropertyKey` が、その時の `attributes` 内に入るイベント固有データのキー名を示す（例 `eventDataDeviceAssignedToServer`）。

### 4.2 AuditEventType（33種）
`DEVICE_ADDED_TO_ORG`, `DEVICE_REMOVED_FROM_ORG`, `DEVICE_ASSIGNED_TO_SERVER`, `DEVICE_UNASSIGNED_FROM_SERVER`,
`SUBJECT_HAS_ICLOUD_STORAGE_PURCHASE_ADDED`, `SUBJECT_HAS_ICLOUD_STORAGE_PURCHASE_REMOVED`,
`SUBJECT_HAS_APPLECARE_PURCHASE_ADDED`, `SUBJECT_HAS_APPLECARE_PURCHASE_REMOVED`, `DEVICE_IS_ERASED`,
`CONFIG_SETTINGS_CREATED`, `CONFIG_SETTINGS_UPDATED`, `CONFIG_SETTINGS_DELETED`,
`COLLECTION_CREATED`, `COLLECTION_UPDATED`, `COLLECTION_DELETED`,
`SUBSCRIPTION_CREATED`, `SUBSCRIPTION_UPDATED`, `SUBSCRIPTION_DELETED`,
`ACCOUNT_ROLE_LOCATION_CHANGED`, `ACCOUNT_ADDED`, `ACCOUNT_DELETED`,
`EXTERNAL_ACCOUNT_ASSOCIATED`, `EXTERNAL_ACCOUNT_DISASSOCIATED`,
`DOMAIN_ADDED`, `DOMAIN_REMOVED`, `DOMAIN_VERIFIED`,
`API_ACCOUNT_CREATED_WITH_KEY`, `API_ACCOUNT_CREATED_WITHOUT_KEY`, `API_ACCOUNT_DELETED`,
`API_ACCOUNT_KEY_REVOKED`, `API_ACCOUNT_KEY_GENERATED`, `API_ACCOUNT_ROLE_LOCATION_CHANGED`, `API_ACCOUNT_NAME_CHANGED`

### 4.3 イベント固有ペイロード（フィールドを持つもの）
| eventData 型 | フィールド |
|---|---|
| `AuditEventDeviceAddedToOrg` | `serialNumber`, `purchaseSourceType`(AuditEventPurchaseSourceType), `purchaseSourceId` |
| `AuditEventDeviceRemovedFromOrg` | `serialNumber`, `releaseEntityId`, `releaseEntityType`(AuditEventReleaseEntityType) |
| `AuditEventDeviceAssignedToServer` | `serialNumber`, `targetServerName` |
| `AuditEventDeviceUnassignedFromServer` | `serialNumber` |
| `AuditEventConfigSettingsCreated/Updated/Deleted` | `configType`, `configId`, `configVersion` |
| `AuditEventCollectionCreated/Updated/Deleted` | `name`, `description` |
| `AuditEventSubscriptionCreated/Updated/Deleted` | `planCaption` |
| `AuditEventSubjectHasICloudStoragePurchaseAdded/Removed` | `subscriptionId` |
| `AuditEventSubjectHasAppleCarePurchaseAdded/Removed` | `subscriptionId` |
| `AuditEventAccountRoleLocationChanged` | `accountRoleLocationList`([AuditEventAccountRoleLocation]) |
| `AuditEventApiAccountRoleLocationChanged` | `apiAccountRoleLocationList`([AuditEventAccountRoleLocation]) |
| `AuditEventApiAccountCreatedWithKey` / `KeyGenerated` / `KeyRevoked` | `keyId` |
| `AuditEventApiAccountNameChanged` | `newName` |
| `AuditEventAccountRoleLocation`（補助） | `roleName`, `locationUniqueIdentifier` |

> 上記以外（`DEVICE_IS_ERASED`, `ACCOUNT_ADDED/DELETED`, `EXTERNAL_ACCOUNT_*`, `DOMAIN_*`, `API_ACCOUNT_CREATED_WITHOUT_KEY/DELETED` 等）は eventData にフィールドを持たない（空）か、共通項目のみ。

> SDK 対応: 上記フィールドありペイロードはすべて `auditevents` パッケージに型付け済み（`Payload(&v)` でデコード）。`AuditEventType` も全33種を定数化。

---

## 5. SDK（apple-business-go）でのマッピング

| API | SDK |
|---|---|
| Type Aliases（列挙） | 文字列定数で定義（例 `configurations.TypeCustomSetting`）。本書を一次情報とする。 |
| リソース属性 | 各パッケージの `Attributes` 構造体。`OrgDevice`/`User` 等の未充足フィールドは本書から補完可能。 |
| `CustomSettingsValues.configurationProfile` | DocC 表記は `byte`(Base64) だが**実機は生 .mobileconfig XML 文字列**を要求（Base64 不可、`PayloadContent` は非空必須）。`string` に生 XML を格納。 |
| Audit Events | `auditevents.Attributes`（共通エンベロープ型付き）＋ `EventData`（生JSON）。`Payload(&v)` で個別型へ。`DeviceAssignedToServer` 等の型を同梱。 |

> 本書のフィールドは Apple 公式 DocC（`applebusinessapi`）の取得結果に基づく。値の最終確認は公式ページで。
