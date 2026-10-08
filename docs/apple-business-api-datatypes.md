# Apple Business API — データ型リファレンス（公式 DocC 確認済み）

出典: `developer.apple.com/documentation/applebusinessapi`（JSレンダリングのため DocC データを取得して確認）。
本書は開発時の参照用。型は JSON:API 表記、`!` は必須、`[T]` は配列、`byte` は Base64 エンコード文字列、`date-time` は ISO8601。

カテゴリ件数（📄 API 2.6 時点、2026-10-08 確認）: Objects and Data Types 71 / Type Aliases 25 / Dictionaries 107（監査イベント機構 103 ＋ `ActivationLockStatus` / `ActivityTypeMetadata` / `MdmServerCreateRequest` / `MdmServerUpdateRequest`）。

> **API 2.6 の差分（Apple Business リリースノート）**: 2026-09-16 に Activation Lock ステータス取得（`ActivationLockStatus` / `ActivationLockType`）、2026-09-23 に監査イベントの拡充（`AuditEventType` 33→50 種、`AuditEventIdpVendorName` / `AuditEventIdpProtocol` 新設、`AuditEventCategory` / `AuditEventSubjectType` に値追加）。DocC の「Introduced in 2.6」注記は既存型にも付くため、2.5 / 2.6 のどちらで入ったかの切り分けは ❓未確認。

---

## 1. Type Aliases（列挙値）— 25件

| 型 | 値 |
|---|---|
| `OrgDeviceActivityType` | `ASSIGN_DEVICES`, `UNASSIGN_DEVICES`, `ASSIGN_DEVICES_WITH_MDM_MIGRATION_DEADLINE`（API 2.3+）, `UPDATE_MDM_MIGRATION_DEADLINE`（API 2.3+）, `CANCEL_MDM_MIGRATION`（API 2.3+）, `RELEASE_DEVICES`（API 2.4+） |
| `MdmMigrationStatus`（API 2.3+） | `REQUESTED`, `STARTED`, `SUCCESS`, `FAILED` |
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
| `ActivationLockType`（API 2.6+） | `MDM`（デバイス管理サービスによるロック）, `USER`（個人の Apple Account へのロック）, `NONE`（未ロック） |
| `AuditEventActorType` | `USER`, `API_USER`, `SYSTEM` |
| `AuditEventSubjectType` | `ORGANIZATION`, `USER`, `LOCATION`, `DEVICE`, `COLLECTION`, `DEVICE_MANAGEMENT_SETTING`, `SUBSCRIPTION`, `DOMAIN`, `API_USER`, `ROLE`（API 2.6+）, `IDP`（API 2.6+）, `INVITATION`（API 2.6+） |
| `AuditEventCategory` | `ORGANIZATION`, `ACCOUNT_ACTIVITY`, `DEVICE_INVENTORY`, `PURCHASING`, `DEVICE_MANAGEMENT`, `ACCESS_MANAGEMENT`（API 2.6+）。※ ✅ 実機は `DEVICE_ASSIGNED_TO_SERVER` で DocC に無い `DEVICE_ACTIVITY` を返した（2026-06 観測）。未知の値を前提に扱うこと |
| `AuditEventOutcome` | `SUCCESS`, `FAILURE` |
| `AuditEventPurchaseSourceType` | `APPLE`, `MANUALLY_ADDED`, `RESELLER` |
| `AuditEventReleaseEntityType` | `USER`, `MDM`, `RESELLER`, `API`, `REPLACEMENT` |
| `AuditEventIdpVendorName`（API 2.6+） | `GOOGLE`, `MICROSOFT`, `MICROSOFT_OIDC`（OpenID Connect の Microsoft）, `GENERIC_IDP`（汎用/カスタム IdP） |
| `AuditEventIdpProtocol`（API 2.6+） | `OIDC`, `SAML` |
| `AuditEventType` | §4.2 を参照（50種。API 2.6 で 17 種追加） |

---

## 2. リソース属性（Attributes）— フィールド定義

各リソースは `{ type, id, attributes, links, relationships? }`。以下は `attributes` の中身。

### OrgDevice.Attributes（`/v1/orgDevices`）
`addedToOrgDateTime`(date-time), `releasedFromOrgDateTime`(date-time), `color`(string), `deviceCapacity`(string),
`deviceModel`(string), `eid`(string), `imei`([string]), `meid`([string]), `wifiMacAddress`(string),
`bluetoothMacAddress`(string), `ethernetMacAddress`([string]), `orderDateTime`(date-time), `orderNumber`(string),
`partNumber`(string), `productFamily`(string), `productType`(string), `purchaseSourceType`(string),
`purchaseSourceId`(string), `serialNumber`(string), `status`(string), `updatedDateTime`(date-time),
`releaserEntityType`(string), `releaserId`(string),
`isMdmMigrationCapable`(boolean, API 2.3+), `mdmMigrationStatus`(MdmMigrationStatus, API 2.3+),
`mdmMigrationDeadlineDateTime`(date-time, API 2.3+)
> 移行系フィールド（API 2.3、2026-08-12）: `isMdmMigrationCapable`＝移行可否、`mdmMigrationStatus`／`mdmMigrationDeadlineDateTime` は移行リクエスト済みの場合のみ設定される。

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
> 公式列挙 — `status`: `COMPLETED` | `IN_PROGRESS` | `STOPPED` | `FAILED` / `subStatus`: `SUBMITTED` | `PRE_PROCESSING` | `PENDING` | `PROCESSING` | `POST_PROCESSING` | `STOPPING` | `COMPLETED_WITH_SUCCESS` | `COMPLETED_WITH_ERROR` | `COMPLETED_WITH_FAILURE` | `COMPLETED_POST_PROCESSING_FAILED`。
> 観測値（実API）: `status` は `IN_PROGRESS`→`COMPLETED`（部分失敗でも COMPLETED）。`subStatus` は `COMPLETED_WITH_ERROR`（一部失敗。`downloadUrl` の CSV に明細）。
> 作成リクエストの `attributes` には `activityType`(OrgDeviceActivityType!) に加え、移行系では `activityTypeMetadata`(ActivityTypeMetadata) を指定（API 2.3+）。`ActivityTypeMetadata`: `mdmMigrationDeadlineDateTime`(date-time)＝移行完了期限（最大 90 日先）。`UPDATE_MDM_MIGRATION_DEADLINE` / `CANCEL_MDM_MIGRATION` / `RELEASE_DEVICES` では `mdmServer` リレーションを指定しない。
> `RELEASE_DEVICES`（API 2.4+、2026-08-26）は組織からデバイスをリリースする破壊的操作: 組織への登録が解除され、デバイス登録割り当てが削除され、組み込みデバイス管理サービス（Apple MDM）から登録解除され、Blueprint からも削除される。
> `COMPLETED_WITH_ERROR` の代表例: **既に同じ MDM サーバへ割り当て済みのデバイスを再割り当て**した場合（状態は変わらずエラー計上）。事前に `assignedServer` で現在の割り当て先を確認すると無駄なエラーを避けられる。

### AppleCareCoverage.Attributes
`status`(AppleCareCoverageStatus), `paymentType`(AppleCareCoveragePaymentType), `description`(string),
`startDateTime`(date-time), `endDateTime`(date-time), `isRenewable`(boolean), `isCanceled`(boolean),
`contractCancelDateTime`(date-time), `agreementNumber`(string)

### ActivationLockStatus.Attributes（`/v1/orgDevices/{id}/activationLockStatus`、API 2.6）
> 📄 **API 2.6（2026-09-16）で追加**。`GET /v1/orgDevices/{id}/activationLockStatus`（クエリ: `fields[activationLockStatus]`＝`isLocked`, `lockType`）→ `ActivationLockStatusResponse`。
> `data.type`＝`activationLockStatus`、`data.id`＝デバイスのシリアル番号。組織が管理していないシリアルは 404。SDK は `devices.ActivationLockStatus`。
> `OrgDevice.Relationships` にも `activationLockStatus` が追加された（`assignedServer` / `appleCareCoverage` と同列。relationships 専用エンドポイントは DocC に無い）。
> 📄 注記: デバイスが管理対象として無効な internal-only のロック状態を報告した場合、予期しないサーバーエラーで失敗することがある。
`isLocked`(boolean), `lockType`(ActivationLockType: `MDM` | `USER` | `NONE`。`isLocked` が true のときのロック種別)

### User.Attributes（`/v1/users`）
`firstName`(string), `middleName`(string), `lastName`(string), `status`(UserStatus), `managedAppleAccount`(string),
`isExternalUser`(boolean), `roleOuList`([UserRoleOuMapping]), `email`(string), `employeeNumber`(string),
`costCenter`(string), `division`(string), `department`(string), `jobTitle`(string),
`phoneNumbers`([UserPhoneNumber]), `startDateTime`(date-time), `createdDateTime`(date-time), `updatedDateTime`(date-time)

### UserGroup.Attributes（`/v1/userGroups`）
`ouId`(string), `name`(string), `type`(UserGroupType), `totalMemberCount`(integer), `status`(UserGroupStatus),
`createdDateTime`(date-time), `updatedDateTime`(date-time)

### OrganizationalUnit.Attributes（`/v1/organizationalUnits`＝組織単位、API 2.2）
> **API 2.2（2026-07-15）で追加**（読み取り専用）。OU は users/userGroups が参照する組織階層エンティティ（`User.roleOuList` の `ouId` / `UserGroup.ouId`）。
> `GET /v1/organizationalUnits`（一覧、`limit`≤1000、カーソルページング）/ `GET /v1/organizationalUnits/{id}`（単体）/ `GET /v1/organizationalUnits/{id}/relationships/users`（所属ユーザーの ID linkage＝`{type:users,id}`、ページング）。related の `…/users` は無く relationships のみ。SDK は `orgunits.List` / `Get` / `Members`。
`name`(string), `description`(string), `createdDateTime`(date-time), `updatedDateTime`(date-time)

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
`ActivationLockStatus`（API 2.6） / `User` / `UserGroup` / `OrganizationalUnit`（API 2.2） / `App` / `Package` / `Blueprint` / `Configuration`
各 `{ type!, id!, attributes, links(ResourceLinks), relationships? }`。

### リクエストボディ
| 型 | 用途 |
|---|---|
| `OrgDeviceActivityCreateRequest` | デバイス割り当て/解除/MDM移行アクティビティ作成（移行系は `activityTypeMetadata` 付き、API 2.3+） |
| `BlueprintCreateRequest` / `BlueprintUpdateRequest` | Blueprint 作成 / 更新 |
| `Blueprint{Apps,Configurations,Packages,OrgDevices,Users,UserGroups}LinkagesRequest` | リレーション割り当て（`{data:[{type,id}]}`） |
| `ConfigurationCreateRequest` / `ConfigurationUpdateRequest` | Configuration 作成 / 更新 |

### レスポンスラッパー（JSON:API 定型）
- 単体: `<Resource>Response` = `{ data(Resource!), links(DocumentLinks!) }`（MdmServer 系は `included([OrgDevice])` を含む）
- 複数: `<Resource>sResponse` = `{ data([Resource]!), links(PagedDocumentLinks!), meta(PagingInformation) }`
- リンク（リレーション）: `*LinkagesResponse` / `OrgDeviceAssignedServerLinkageResponse`
- 該当: `OrgDeviceResponse`, `OrgDevicesResponse`, `MdmDeviceResponse`, `MdmDevicesResponse`, `MdmDeviceDetailResponse`,
  `MdmServerResponse`, `MdmServersResponse`, `MdmServerDevicesLinkagesResponse`, `OrgDeviceActivityResponse`,
  `AppleCareCoverageResponse`, `ActivationLockStatusResponse`（API 2.6）, `UserResponse`, `UsersResponse`, `UserGroupResponse`, `UserGroupsResponse`,
  `UserGroupUsersLinkagesResponse`, `OrganizationalUnitResponse`, `OrganizationalUnitsResponse`, `OrganizationalUnitUsersLinkagesResponse`（API 2.2）,
  `AppResponse`, `AppsResponse`, `PackageResponse`, `PackagesResponse`,
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

> **クエリ（必須）**: `filter[startTimestamp]` が必須（✅ 未指定だと 400 `PARAMETER_ERROR.REQUIRED`）。通常は `filter[endTimestamp]` と併用し、いずれも ISO8601(RFC3339, UTC)。SDK は `auditevents.ListRange(start, end, q)` で付与。
> 📄 その他のクエリ（DocC）: `filter[actorId]` / `filter[subjectId]` / `filter[type]`（いずれも **1 値のみ**。`filter[type]` は `AuditEventType` の値）、`fields[auditEvents]`（共通項目名と `eventData<Event>` キー名）、`limit`（≤1000）、`cursor`。

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

### 4.2 AuditEventType（50種）
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

**📄 API 2.6（2026-09-23）で追加された 17 種**:
`ROLE_CREATED`, `ROLE_UPDATED`, `TERMS_ACCEPTED`, `DOMAIN_FEDERATION_ENABLED`, `DOMAIN_FEDERATION_DISABLED`,
`BATCH_PASSWORD_RESET_INITIATED`, `EXTERNAL_ACCOUNT_INVITED`, `ACCOUNT_IMAGE_SET`, `ACCOUNT_IMAGE_REPLACED`, `ACCOUNT_IMAGE_REMOVED`,
`IDP_CREATED`, `IDP_UPDATED`, `IDP_DELETED`, `IDP_ACCOUNT_SYNC_ENABLED`, `IDP_ACCOUNT_SYNC_DISABLED`,
`APPLE_SERVICES_SETTINGS_UPDATED`, `ACCOUNT_PASSWORD_CHANGED`

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
| `AuditEventRoleCreated`（API 2.6+） | `privileges`([string]) |
| `AuditEventRoleUpdated`（API 2.6+） | `privilegesAdded`([string]), `privilegesRemoved`([string]) |
| `AuditEventTermsAccepted`（API 2.6+） | `termsName` |
| `AuditEventDomainFederationEnabled`（API 2.6+） | `activityId`, `scimActivityId` |
| `AuditEventDomainFederationDisabled`（API 2.6+） | `activityId` |
| `AuditEventBatchPasswordResetInitiated`（API 2.6+） | `activityId` |
| `AuditEventAccountPasswordChanged`（API 2.6+） | `activityId` |
| `AuditEventIdpCreated`（API 2.6+） | `vendorName`(AuditEventIdpVendorName), `protocol`(AuditEventIdpProtocol) |
| `AuditEventIdpUpdated` / `AuditEventIdpDeleted`（API 2.6+） | `vendorName`(AuditEventIdpVendorName) |
| `AuditEventIdpAccountSyncEnabled` / `AuditEventIdpAccountSyncDisabled`（API 2.6+） | `vendorName`(AuditEventIdpVendorName), `activityId` |
| `AuditEventAppleServicesSettingsUpdated`（API 2.6+） | `settingsEnabled`([string]), `settingsDisabled`([string]) |

> 上記以外（`DEVICE_IS_ERASED`, `ACCOUNT_ADDED/DELETED`, `EXTERNAL_ACCOUNT_ASSOCIATED/DISASSOCIATED/INVITED`, `DOMAIN_ADDED/REMOVED/VERIFIED`, `API_ACCOUNT_CREATED_WITHOUT_KEY/DELETED`, `ACCOUNT_IMAGE_SET/REPLACED/REMOVED` 等）は eventData にフィールドを持たない（空オブジェクト）か、共通項目のみ（📄 2026-10-08 に DocC で再確認）。

> SDK 対応: 上記フィールドありペイロードはすべて `auditevents` パッケージに型付け済み（`Payload(&v)` でデコード。API 2.6 分は `RoleCreated` / `RoleUpdated` / `TermsAccepted` / `DomainFederationEnabled` / `DomainFederationDisabled` / `BatchPasswordResetInitiated` / `AccountPasswordChanged` / `IDPCreated` / `IDP` / `IDPAccountSync` / `AppleServicesSettingsUpdated`）。`AuditEventType` も全 50 種を `Type*` 定数化。`Category*` / `Subject*` / `Actor*` / `Outcome*` / `IDPVendor*` / `IDPProtocol*` の定数もある。

---

## 5. SDK（apple-business-go）でのマッピング

| API | SDK |
|---|---|
| Type Aliases（列挙） | 文字列定数で定義（例 `configurations.TypeCustomSetting`）。本書を一次情報とする。 |
| リソース属性 | 各パッケージの `Attributes` 構造体。`OrgDevice`/`User` 等の未充足フィールドは本書から補完可能。 |
| `CustomSettingsValues.configurationProfile` | DocC 表記は `byte`(Base64) だが**実機は生 .mobileconfig XML 文字列**を要求（Base64 不可、`PayloadContent` は非空必須）。`string` に生 XML を格納。 |
| Audit Events | `auditevents.Attributes`（共通エンベロープ型付き）＋ `EventData`（生JSON）。`Payload(&v)` で個別型へ。`DeviceAssignedToServer` 等の型を同梱。 |

> 本書のフィールドは Apple 公式 DocC（`applebusinessapi`）の取得結果に基づく。値の最終確認は公式ページで。
