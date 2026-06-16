# Design Document

Issue: [#14](https://github.com/hitoshiichikawa/apple-business-go/issues/14)

## Overview

4 つの独立した小規模ハードニング。互いに依存はなく、いずれも公開シグネチャを変えない。

## Goals / Non-Goals

- **Goals**: ボディ上限 / jti エラー伝播 / rel 許可リスト / ListRange 非破壊
- **Non-Goals**: リトライ・ホスト検証（別 issue）、上限値の設定 API 化

## Components and Interfaces

### 1. レスポンス上限（applebusiness/client.go）

```go
var maxResponseBytes int64 = 32 << 20 // テストから差し替えるため var
const maxDrainBytes = 1 << 20
```

- `Do` 成功時: `io.LimitedReader{N: maxResponseBytes + 1}` 越しにデコードし、
  デコード失敗かつ `lr.N <= 0` なら「上限超過」の明確なエラー、それ以外は従来の decode エラー
- `drainAndClose`: 読み捨ても `io.LimitReader(body, maxDrainBytes)` で上限化。残量が多い場合は
  接続再利用を諦める（Close が接続を破棄）

### 2. jti（applebusiness/oauth.go）

- `newJTI() (string, error)` に変更（非公開関数のため互換影響なし）。
  `buildClientAssertion` がエラーを伝播

### 3. rel 許可リスト（blueprints/blueprints.go）

```go
var validRels = map[string]bool{RelApps: true, ..., RelUserGroups: true}
func checkRel(rel string) error // 未知値は "unknown relationship %q (use the Rel* constants)"
```

- `RelationshipIDs` と `modifyRel`（AddTo / RemoveFrom / Replace の共通経路）の入口で検証

### 4. ListRange（auditevents/auditevents.go）

- 引数 `q` を `merged` にコピーしてから `Set`。`url.Values.Set` はキーのスライスを差し替えるため
  浅いコピーで十分（呼び出し側のスライスは変更されない）

## Error Handling

- 上限超過: `applebusiness: response body exceeds <N> bytes`
- 未知 rel: `blueprints: unknown relationship "<rel>" (use the Rel* constants)`（リクエスト送信前）

## Testing Strategy

- `TestDoRejectsOversizedResponse`: 上限を 1 KiB に差し替え、4 KiB の JSON を返すサーバで
  上限エラーを確認（Req 1.1）
- `TestRelValidation`: パス操作風の rel / 未知 rel / 空 rel が**リクエスト送信前に**エラー、
  既知 Rel* は従来どおり（Req 3.1, 3.2）
- `TestListRangeDoesNotMutateQuery`: 呼び出し後に q が不変、nil q も動作（Req 4.1, 4.2)
- Req 2.1（rand.Read 失敗）は強制再現の手段がないためテスト対象外（コードレビューで担保）
