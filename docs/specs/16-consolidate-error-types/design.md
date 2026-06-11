# Design Document

Issue: [#16](https://github.com/hitoshiichikawa/apple-business-go/issues/16)

## Overview

`applebusiness` パッケージ内のエラー定義の移動と小さな公開 API 追加（`ErrorObject` 型、
`APIError.RawBody` フィールド）。HTTP 挙動は変えない。

## Goals / Non-Goals

- **Goals**: errors.go への集約、`ErrorObject` 公開、非 JSON ボディ断片の保持
- **Non-Goals**: エラー分類の拡張、リトライ連動、Error() の既存フォーマット変更（Errors がある場合）

## Components and Interfaces

```go
// errors.go（集約先）
type ErrorObject struct {
    Status string `json:"status"`
    Code   string `json:"code"`
    Title  string `json:"title"`
    Detail string `json:"detail"`
}

type APIError struct {
    StatusCode int
    Errors     []ErrorObject `json:"errors"`
    RawBody    string        `json:"-"` // 非 JSON:API ボディの断片（Errors があるときは空）
}

func decodeAPIError(resp *http.Response) error
```

- `decodeAPIError`: `io.LimitReader(body, 64KiB)` で読み、`json.Unmarshal` が成功して
  `len(Errors) > 0` ならそのまま返す。それ以外は TrimSpace + 200 文字に切った断片を `RawBody` へ
- `Error()`: ① Errors あり → 従来形式（変更なし）② RawBody あり → `"applebusiness: API error %d: %s"`
  ③ どちらもなし → 従来のステータスのみ
- 挙動差分（意図的）: 旧実装では `{}` のような「JSON だが errors なし」のボディは無言で捨てられたが、
  新実装では `RawBody` に残る。情報量が増える方向の差分で後方互換の範囲とする

## Error Handling

- ボディ読み取りエラーは無視（従来どおりベストエフォート。`StatusCode` は常に保持される）
- `RawBody` は `json:"-"` とし、`APIError` を JSON 化する利用側ログへ意図せず生ボディが
  混入する経路を増やさない

## Testing Strategy

- 既存 `TestAPIErrorDecodeAndClassifier`: JSON:API エラーのデコードと述語（無変更でパス = Req 1.3 の回帰確認）
- 新規 `TestAPIErrorNonJSONBody`: HTML ボディの 400 に対し `RawBody` 断片と `Error()` への
  埋め込みを検証（Req 2.1, 2.2）
- コンパイル成功でソース互換を確認（Req 1.3）
