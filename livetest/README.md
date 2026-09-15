# livetest — 実テナントに対する応答期待の網羅検証

Apple Business API の応答（成功・エラーコード）を、実テナントに対して確認する検証テストです。
`//go:build livetest` タグが付いているため、通常の `go build ./...` / `go test ./...` / CI からは
除外され、ネットワークにも触れません。

## 実行

```bash
export AXM_CLIENT_ID="BUSINESSAPI.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export AXM_KEY_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export AXM_PRIVATE_KEY_PATH="$HOME/.config/apple-business/<tenant>/private-key.pem"
# 任意: AXM_TEAM_ID / AXM_SCOPE / AXM_BASE_URL / AXM_TOKEN_URL

go test -tags livetest ./livetest -v
```

`AXM_CLIENT_ID` が未設定なら、各テストは `t.Skip` で自動的にスキップします。

## ⚠️ 実組織への書き込み

読み取りに加えて、テスト用の **Configuration / Blueprint / MDM サーバを作成・削除**します。
後始末は `t.Cleanup` で必ず行いますが、実行は影響の少ないテナント（例: 検証用）で行ってください。
デバイスへの割り当てや、既存リソースの変更はしません。トークン・秘密鍵は出力しません。

## 検証内容

| テスト | 確認する応答 |
|---|---|
| `TestAuth_AccessTokenSucceeds` | トークン取得が成功する |
| `TestReads_SucceedOrPermission403` | 各読み取り（devices / mdmServers / mdmDevices / users / userGroups / orgunits / apps / packages / configurations / blueprints / auditEvents）が成功する（権限 403 はログのみ） |
| `TestGet_BogusID_IsNotFound` | 存在しない id の取得が 404（`IsNotFound`） |
| `TestBlueprintCreate_Validation` | メンバー無しの作成 → 409 `MISSING_MEMBERS`／リソース無しの作成 → 409 `MISSING_RESOURCES` |
| `TestBlueprintRelationships` | `AddTo`/`RemoveFrom`(apps) 成功、`Replace`(apps / configurations) → 403 `FORBIDDEN_ERROR`（REPLACE 非対応）。`Replace`(orgDevices / packages) は結果を記録のみ |
| `TestMdmServer_CertAlgorithm` | EC 証明書 → 400 `PARAMETER_ERROR.INVALID`、RSA 証明書 → Create/Get/Update/Delete 成功 |

「記録のみ」の項目（メンバー系関連への REPLACE、packages への REPLACE）は、Apple の挙動が未確認のため
`t.Logf` で観測結果を出すだけで、成否は判定しません（`-v` で確認）。

関連: [#44](https://github.com/hitoshiichikawa/apple-business-go/issues/44)
