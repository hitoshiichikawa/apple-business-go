# apple-business-go

**Go SDK for Apple Business**（非公式）。Apple Business / School Manager の API
（`api-business.apple.com` / `api-school.apple.com`）を呼び出すためのライブラリです。
OAuth2（client_credentials + ES256 JWT）認証、デバイス・ユーザー・MDMサーバ等の取得、
デバイスの**割り当て / 解除（書き込み）**に対応します。

> 2026年4月、Apple Business Manager・Apple School Manager・Apple Business Essentials は
> 単一プラットフォーム **「Apple Business」** に統合されました（device / people / **brand** / support の4本柱）。
> 本SDKはこの上位ブランドに合わせ、将来のピラー追加（brand management 等）を見据えた構成にしています。

> ⚠️ **非公式SDK**です。API定義は公開実装から抽出・検証したものですが、本番投入前に Apple 公式での最終確認を推奨します
> （詳細は `docs/apple-business-api-reference.md`）。

---

## パッケージ構成（4本柱に対応）

```
apple-business-go/
├── applebusiness/   コア（共通基盤）: Client, Config, Credentials, OAuth2/JWT, transport, pagination, errors
├── devices/         デバイス管理: orgDevices, mdmServers, orgDeviceActivities, appleCareCoverage
├── blueprints/      Blueprint管理: CRUD + 割り当て（apps/configurations/devices/users/groups）
├── configurations/  Configuration管理: CRUD（CUSTOM_SETTING プロファイル）
├── apps/            アプリ/パッケージ（読み取り）: apps, packages
├── auditevents/     監査イベント（読み取り）: auditEvents
├── people/          ピープル管理: users, userGroups
├── brand/           ブランド管理（将来 / Apple Business Connect 系）
└── support/         サポート（将来）
```

- サービスパッケージ（`devices` / `people`）は `*applebusiness.Client` を受け取って動作します。
- 認証・リトライ・ページネーションは `applebusiness` コアに集約。柱が増えてもコアは不変です。

---

## インストール

```bash
go get github.com/hitoshiichikawa/apple-business-go/applebusiness
go get github.com/hitoshiichikawa/apple-business-go/devices
```

## クイックスタート（読み取り）

```go
import (
    "github.com/hitoshiichikawa/apple-business-go/applebusiness"
    "github.com/hitoshiichikawa/apple-business-go/devices"
)

pem, _ := os.ReadFile("abm_private_key.pem") // EC P-256 (.pem)

c, err := applebusiness.NewClient(applebusiness.Config{
    BaseURL: applebusiness.DefaultBusinessBaseURL, // ASM は DefaultSchoolBaseURL
    Credentials: applebusiness.Credentials{
        ClientID:   "BUSINESSAPI.xxxxxxxx-....",
        TeamID:     "BUSINESSAPI.xxxxxxxx-....", // AxM では client_id と同一が通例
        KeyID:      "xxxxxxxx-....",
        PrivateKey: pem,
        Scope:      "business.api", // 空なら client_id から自動判定
    },
})

devs, err := devices.New(c).List(context.Background(), nil) // ページングは自動
for _, d := range devs {
    fmt.Println(d.Attributes.SerialNumber, d.Attributes.Status)
}
```

## クイックスタート（書き込み: 割り当て / 解除）

```go
svc := devices.New(c)
act, err := svc.Assign(ctx, serverID, []string{deviceID1, deviceID2})
// 解除なら svc.Unassign(ctx, serverID, deviceIDs)

final, err := svc.PollActivity(ctx, act.ID, 3*time.Second)
fmt.Println(final.Attributes.Status, final.Attributes.SubStatus)
```

実行サンプルは [`examples/`](./examples)（`list-devices`, `assign-devices`, `smoke-test`, `dump-all`, `write-test`）。疎通確認は [`smoke-test`](./examples/smoke-test)、全リード系の応答ダンプは [`dump-all`](./examples/dump-all)。

## クイックスタート（監査イベント）

```go
// /v1/auditEvents は filter[startTimestamp] が必須。期間指定は ListRange を使う
events, _ := auditevents.New(c).ListRange(ctx, time.Now().AddDate(0, 0, -7), time.Now(), nil)
for _, ev := range events {
    a := ev.Attributes // 共通項目: a.Type, a.ActorID, a.EventDateTime ...
    if a.Type == "DEVICE_ASSIGNED_TO_SERVER" {
        var d auditevents.DeviceAssignedToServer
        _ = a.Payload(&d) // d.SerialNumber, d.TargetServerName
    }
}
```

---

## 認証

資格情報は Apple Business / School Manager のポータルで発行（Organization Administrator のみ）。

| 項目 | 内容 |
|---|---|
| `client_id` | 例: `BUSINESSAPI.<uuid>` |
| `key_id` | JWT ヘッダの `kid` |
| `team_id` | issuer（AxM では `client_id` と同一が通例） |
| 秘密鍵 | `.pem`（EC P-256 / ES256用） |

ES256 で署名した JWT client assertion を生成し、`POST https://account.apple.com/auth/oauth2/token` に
`grant_type=client_credentials` で交換。`scope` は `business.api` / `school.api`。アクセストークンは1時間有効。

---

## 対応エンドポイント

| パッケージ | エンドポイント | API |
|---|---|---|
| `devices` | `List` / `Get` | `/v1/orgDevices`, `/v1/orgDevices/{id}` |
| `devices` | `AssignedServer` / `AppleCareCoverage` | `/v1/orgDevices/{id}/assignedServer`, `.../appleCareCoverage` |
| `devices` | `ListMdmServers` / `MdmServerDevices` | `/v1/mdmServers`, `/v1/mdmServers/{id}/relationships/devices` |
| `devices` | `Assign` / `Unassign` / `GetActivity` / `PollActivity` | `/v1/orgDeviceActivities(/{id})` |
| `people` | `ListUsers` / `GetUser` | `/v1/users`, `/v1/users/{id}` |
| `people` | `ListUserGroups` / `GetUserGroup` / `GroupMembers` | `/v1/userGroups(/{id})(/relationships/users)` |
| `blueprints` | `List` / `Get` / `Create` / `Update` / `Delete` / `AddTo` / `RemoveFrom` / `Replace` / `RelationshipIDs` | `/v1/blueprints(/{id})(/relationships/{rel})` |
| `configurations` | `List` / `Get` / `Create` / `Update` / `Delete` | `/v1/configurations(/{id})` |
| `apps` | `ListApps` / `GetApp` / `ListPackages` / `GetPackage` | `/v1/apps(/{id})`, `/v1/packages(/{id})` |
| `auditevents` | `List` | `/v1/auditEvents`（絞り込みクエリ名は要確認） |

フィールド・列挙値の一次情報は [`docs/apple-business-api-datatypes.md`](./docs/apple-business-api-datatypes.md)、
エンドポイントの詳細は [`docs/apple-business-api-reference.md`](./docs/apple-business-api-reference.md)。

---

## ステータス / ロードマップ

- [x] コア（OAuth2 / リトライ / ページネーション）
- [x] `devices`（読み取り + 割り当て/解除）
- [x] `people`（users / userGroups）
- [x] `apps`（apps / packages）/ `auditevents`（auditEvents）: 実装済み（読み取り）。公式フィールド一致。テストは未
- [x] `blueprints` / `configurations`（組み込みデバイス管理）: 実装済み（CRUD + 割り当て）。公式仕様確認済み。実機検証・テストは未
- [x] 全リソースの `Attributes`・列挙値・監査モデルを公式 DocC で確認し一致（[`docs/apple-business-api-datatypes.md`](./docs/apple-business-api-datatypes.md)）
- [x] Functional Options（`WithBaseURL`/`WithTokenURL`/`WithMaxRetries`/`WithUserAgent`/`WithHTTPClient`）
- [x] 型付きエラー判定（`IsNotFound`/`IsRateLimited`/`IsUnauthorized`/`IsForbidden`/`IsConflict`）
- [x] `ListSeq`（Go 1.23 range-over-func の遅延ページング）
- [x] コアの単体テスト（`httptest`）/ CHANGELOG / golangci-lint / CI（gofmt・vet・build・test -race・lint）
- [ ] 各ドメインパッケージの単体テスト拡充 / 書き込みリトライの冪等性レビュー
- [ ] `brand` / `support` パッケージ（将来）

---

## 開発

Go 1.23+（`ListSeq` の range-over-func を使用）。

```bash
go mod tidy
go build ./...
go vet ./...
gofmt -l .        # 差分なしを確認
go test -race ./...
```

---

## クレジット

API定義・認証フローは以下の公開実装を参考に独自実装したものです（コードの直接流用なし）。

- [`micromdm/nanoaxm`](https://github.com/micromdm/nanoaxm) (Go)
- [`neilmartin83/terraform-provider-axm`](https://github.com/neilmartin83/terraform-provider-axm) (Go)
- ほか `rodchristiansen/asbmutil` (Swift), `EUCTechTopics/PSABM` (PowerShell) 等

## ライセンス

[MIT](./LICENSE)（著作権者表記は適宜変更してください）。
Apple、Apple Business、Apple Business Manager、Apple School Manager は Apple Inc. の商標です。本プロジェクトは Apple とは無関係の非公式実装です。
