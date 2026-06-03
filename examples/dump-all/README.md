# dump-all — 全リード系エンドポイントの応答を標準出力

全カテゴリの**読み取り専用**エンドポイントを順に呼び出し、実際のレスポンス(JSON)を表示します。
一覧→先頭IDで詳細/サブリソースへドリルダウンします。取得できない（404/403/空）箇所はスキップして続行します。

> **安全設計**: GET のみ。割り当て・作成・更新・削除などの**書き込みは一切行いません**（実組織を変更しないため）。

## 環境変数
smoke-test と同じです（`AXM_CLIENT_ID` / `AXM_KEY_ID` / `AXM_PRIVATE_KEY_PATH` 必須、`AXM_TEAM_ID` / `AXM_SCOPE` / `AXM_BASE_URL` / `AXM_TOKEN_URL` 任意）。

## 実行

```bash
go run ./examples/dump-all                    # 全カテゴリ・既定 limit=3（人間可読）
go run ./examples/dump-all -limit 1           # 1件ずつ
go run ./examples/dump-all -only devices      # カテゴリ名の部分一致でフィルタ
go run ./examples/dump-all -only audit
go run ./examples/dump-all > dump.txt         # 応答を丸ごと保存（人間可読）
go run ./examples/dump-all -json > dump.json  # 機械可読(JSON配列)で保存
```

- 進捗・トークン確認・サマリは標準エラー出力（stderr）に出します。**API 応答は標準出力（stdout）**なので、`> file` でリダイレクトすると応答だけ保存できます。
- 出力されるトークンは先頭8文字のみマスク表示。`.pem`・トークンはコミット/共有しないでください。

## 叩く順番（GETのみ）
1. **Devices**: `/v1/orgDevices` → `/v1/orgDevices/{id}` → `/assignedServer` → `/appleCareCoverage`
2. **Device Management Services**: `/v1/mdmServers` → `/v1/mdmServers/{id}` → `/relationships/devices`
3. **Users**: `/v1/users` → `/v1/users/{id}`
4. **UserGroups**: `/v1/userGroups` → `/v1/userGroups/{id}` → `/relationships/users`
5. **Apps / Packages**: `/v1/apps`(→`/{id}`) / `/v1/packages`(→`/{id}`)
6. **Blueprints**: `/v1/blueprints` → `/v1/blueprints/{id}` → `/relationships/{orgDevices,userGroups,users,apps,configurations,packages}`
7. **Configurations**: `/v1/configurations` → `/v1/configurations/{id}`
8. **Audit Events**: `/v1/auditEvents`（**`filter[startTimestamp]` が必須**のため、直近 `-audit-days` 日分を自動付与）

## フラグ
| フラグ | 既定 | 説明 |
|---|---|---|
| `-limit N` | 3 | 一覧の取得件数 |
| `-only S` | （空） | カテゴリ名の部分一致フィルタ（devices, blueprints, audit など） |
| `-json` | false | JSON 配列で出力（`{method,path,ok,error,response}`） |
| `-audit-days N` | 7 | auditEvents の取得期間（日数。`filter[startTimestamp]` に使用） |
| `-timeout D` | 120s | 全体タイムアウト |

応答の生フィールドの意味は、リポジトリの `docs/apple-business-api-datatypes.md` を参照してください。
