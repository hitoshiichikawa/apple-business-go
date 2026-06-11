# Design Document

Issue: [#11](https://github.com/hitoshiichikawa/apple-business-go/issues/11)

## Overview

`Do` のリトライ判定にメソッド条件を加える最小差分。リトライループの構造・バックオフ・
`Retry-After` の扱いは変えない。

## Goals / Non-Goals

- **Goals**: POST の 5xx／ネットワークエラー非再送、429 全メソッド再送の維持、方針の文書化
- **Non-Goals**: リトライ機構の再設計、冪等キー導入

## Decisions

| 論点 | 決定 | 根拠 |
|---|---|---|
| 429 | 全メソッド再送 | レート制限はサーバが処理を受け付けていないことがほぼ確実 |
| 5xx + POST | 非再送 | 処理済みの可能性があり二重実行リスク。エラーボディを `decodeAPIError` で取得できる副次改善あり（従来のリトライ枯渇パスはステータスのみだった） |
| ネットワークエラー + POST | 非再送 | レスポンス未受領でも到達・処理済みの可能性はゼロでない（安全側） |
| PATCH | 再送（従来どおり） | JSON:API の PATCH は指定属性の置換であり再実行しても同じ結果（実質冪等） |
| DELETE | 再送（従来どおり） | 冪等（2 回目は 404 になり得るが二重実行害なし） |

## Components and Interfaces

```go
// Do 内のリトライ判定（変更箇所のみ）
if err != nil {                      // ネットワークエラー
    if errors.Is(err, errCrossHost) { return err }
    if method == http.MethodPost    { return err }   // 追加
    ...従来のバックオフ...
}
retryable := resp.StatusCode == http.StatusTooManyRequests ||
    (resp.StatusCode >= 500 && method != http.MethodPost)   // 5xx は POST を除外
```

## Error Handling

- POST の 5xx は `resp.StatusCode >= 400` の分岐へ落ち `decodeAPIError` を通る → JSON:API の
  エラー内容が `APIError.Errors` に入る（Req 1.4）

## Testing Strategy

- `TestPostNotRetriedOn5xx`: 500 を返し続けるサーバへ Create → 受信 1 回・`APIError(500)`・
  エラーボディがデコードされている（Req 1.1, 1.4）
- `TestPostRetriedOn429`: 429 → 200 で Create 成功・受信 2 回（Req 1.2）
- `TestGetRetriedOn500`: 500 → 200 で Get 成功・受信 2 回（Req 2.1）
- 既存 `TestRetryThenSuccess` / `TestRateLimitedExhausted`: GET の従来挙動の回帰なし（Req 2.1）
- Req 1.3（ネットワークエラー時の POST 非再送）はコードパス上 `method == POST` の早期 return で
  自明のため専用テストは省略（5xx 側のテストで判定ロジックはカバー）
