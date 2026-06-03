# assign-devices — デバイスの MDM 割り当て / 解除（書き込み）

組織デバイスを MDM サーバへ**割り当て / 解除**し、生成された `orgDeviceActivities` を完了までポーリングします。

> ⚠️ **書き込み操作です**（実組織のデバイスの MDM 割り当てを変更します）。
> 既定は**ドライラン**（無変更・現状プレビューのみ）。実行は `-yes` が必須です。
> 運用アプリでは admin 限定 + 全件監査での実行を推奨します。

## 環境変数
smoke-test / dump-all と同じ（`AXM_CLIENT_ID` / `AXM_KEY_ID` / `AXM_PRIVATE_KEY_PATH` 必須、`AXM_TEAM_ID` / `AXM_SCOPE` / `AXM_BASE_URL` / `AXM_TOKEN_URL` 任意）。

## 使い方

```bash
# 1) ドライラン（現状プレビューのみ・無変更）
go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1,SERIAL2

# 2) 実行（割り当て）
go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1,SERIAL2 -yes

# 3) 解除
go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1 -unassign -yes

# 4) 実行＋結果CSVを保存
go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1 -yes -save-results result.csv

# 5) 既存アクティビティの状態確認＋結果CSV取得（書き込みなし・署名URLは取り直すので期限切れ回避）
go run ./examples/assign-devices -activity <activityId> -save-results result.csv
```

- `mdmServerId` は `dump-all`（`/v1/mdmServers`）の各要素 `id`、`SERIAL` は orgDevices の `id`（= シリアル）です。
- ドライランでは各デバイスの `serial / model / status / 現在の割り当て先` を表示します（読み取りのみ）。
- `-yes` で実行すると `activityType=ASSIGN_DEVICES|UNASSIGN_DEVICES` の `orgDeviceActivities` を作成し、`-wait`（既定 true）で完了までポーリングして `status` / `subStatus` を表示します。

## フラグ
| フラグ | 既定 | 説明 |
|---|---|---|
| `-server` | （必須） | 対象 MDM サーバ ID |
| `-devices` | （必須） | デバイスID(serial) カンマ区切り |
| `-unassign` | false | 解除する（既定は割り当て） |
| `-yes` | false | 実際に実行（未指定はドライラン） |
| `-wait` | true | 完了までポーリング |
| `-activity` | （空） | 既存アクティビティID（指定時は**新規作成せず**状態確認/結果取得のみ＝書き込みなし） |
| `-save-results` | （空） | 結果CSVの保存先パス（`downloadUrl` から取得） |
| `-timeout` | 5m | 全体タイムアウト |

## 結果CSV（downloadUrl）のダウンロード
- `downloadUrl` は**署名付き・期限付き**（`Expires=...`）。古いURLは失効します。`-activity <id>` で取り直すと新しい署名URLになります。
- 生URLを `curl` で扱う場合は **`&` を含むため必ずシングルクォートで囲む**こと（囲まないと `&` がバックグラウンド実行と解釈されURLが切れる）:
  ```bash
  curl -L -o result.csv 'https://store-033.blobstore.apple.com/...（URL全体）...'
  ```
- URL内の `%22`(=`"`)・`%3B`(=`;`)・`%3D`(=`=`) は正しいパーセントエンコードです（手動でデコードしないこと）。`-save-results` を使えば生URLに触れずに保存できます。

## 注意
- 割り当ては「移動」です。別サーバに割り当て済みのデバイスを指定すると、その割り当てが変わります（ドライランの「現在の割り当て先」で必ず確認を）。
- 終了ステータスの厳密な文字列（`status` / `subStatus`）は実行結果で確定します。`PollActivity` の終了判定は観測値に合わせて調整可能です。
