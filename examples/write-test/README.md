# write-test — 全書き込みAPIの動作確認（安全設計）

SDK の書き込みAPIを一通り実行して確認するツールです。

> ⚠️ **実組織への書き込み**です。
> - 既定は**ドライラン**（無変更）。実行は `-yes` が必須。
> - Blueprint / Configuration は**専用のテスト用リソースを作成→更新→削除**する自己完結方式（既定で後始末。`-keep` で残す）。
> - **デバイス割り当て**は実デバイスの状態を変えるため**明示オプトイン**（`-assign-server` と `-assign-device` の両方指定時のみ）。元の割り当て先へベストエフォートで復元します。

## 実行内容
1. **Blueprints**: `Create` → `Update` →（`-app <id>` 指定時）`AddTo(apps)`→`RemoveFrom(apps)` → `Delete`
2. **Configurations**: `Create`(CUSTOM_SETTING・空の.mobileconfigをBase64) → `Update` → `Delete`
3. **Devices**（オプトイン）: 元の割り当て先を記録 → `Assign(target)` → 復元（元へ`Assign`／無ければ`Unassign`）。各アクティビティを完了までポーリング。

## 使い方

```bash
# ドライラン（実行計画の表示のみ・無変更）
go run ./examples/write-test

# Blueprint/Configuration の作成→更新→削除を実行
go run ./examples/write-test -yes

# Blueprint の apps リレーション付替も試す
go run ./examples/write-test -yes -app <appId>

# 作成物を消さずに残す
go run ./examples/write-test -yes -keep

# 割り当ても試す（実デバイスを変更→復元）
go run ./examples/write-test -yes -assign-server <mdmServerId> -assign-device <serial>
```

## フラグ
| フラグ | 既定 | 説明 |
|---|---|---|
| `-yes` | false | 実際に書き込む（未指定はドライラン） |
| `-keep` | false | 作成したテスト用リソースを削除しない |
| `-app` | （空） | Blueprint の apps リレーション付替に使う app ID |
| `-assign-server` | （空） | 割り当てテストの MDM サーバ ID（`-assign-device` と併用） |
| `-assign-device` | （空） | 割り当てテストのデバイス ID(serial) |
| `-timeout` | 10m | 全体タイムアウト |

## 注意
- `configurationProfile` は仕様どおり **Base64**（`.mobileconfig` を Base64 エンコード）で送ります。
- Configuration の作成は、プロファイル内容を Apple 側が検証して失敗することがあります（その場合もエラーを表示して続行＝エンドポイントの疎通自体は確認できます）。
- 割り当てで「同一サーバへの再割り当て」になると `subStatus=COMPLETED_WITH_ERROR` になります（状態は不変）。
- 失敗が1件でもあると終了コードは 1 です。
