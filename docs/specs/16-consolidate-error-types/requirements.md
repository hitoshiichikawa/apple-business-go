# Requirements Document

Issue: [#16](https://github.com/hitoshiichikawa/apple-business-go/issues/16)

## Introduction

エラー関連の定義が `client.go`（`APIError` / `decodeAPIError`）と `errors.go`（`Is*` 述語）に
分散している。また `APIError.Errors` の要素が匿名 struct のため利用側が型名を参照できず、
`decodeAPIError` は非 JSON のエラーボディ（プロキシの HTML 等）を黙って捨てるため障害調査の
手がかりが残らない。エラー定義を 1 ファイルに集約し、要素型を公開 named type にし、
非 JSON ボディの断片を保持する。

## Requirements

### Requirement 1: エラー定義の集約と named type 化

**Objective:** As a SDK 利用者, I want エラー要素を名前付きの型として扱えること, so that エラー要素を変数・関数引数として自然に取り回せる

#### Acceptance Criteria

1. The applebusiness パッケージ shall `APIError` / 要素型 / デコード / `Is*` 述語を `errors.go` に集約する
2. The applebusiness パッケージ shall 公開型 `ErrorObject`（Status / Code / Title / Detail、JSON タグは従来と同一）を提供する
3. When 既存の利用コード（`e.Errors[0].Code` 等のフィールドアクセス）をコンパイルしたとき, the モジュール shall ソース互換でビルドに成功する

### Requirement 2: 非 JSON エラーボディの可視化

**Objective:** As a SDK 利用者, I want JSON でないエラー応答でも本文の断片が得られること, so that 障害時に何が返ってきたか調査できる

#### Acceptance Criteria

1. If エラーレスポンスのボディが JSON:API エラー文書として解釈できないとき, the APIError shall ボディ先頭の断片（上限 200 文字）を `RawBody` に保持する
2. If `Errors` が空で `RawBody` が非空のとき, the `Error()` 文字列 shall ステータスコードと `RawBody` 断片を含む
3. The decodeAPIError shall ボディの読み取りを上限（64 KiB）付きで行う

## Non-Functional Requirements

### NFR 1: 互換性

1. The 変更 shall 公開 API のソース互換を維持する（匿名 struct → 同一フィールドの named struct）
2. The `RawBody` shall `Errors` が取得できた場合は空のままにする（既存のエラーメッセージ形式を変えない）

## Out of Scope

- `Is*` 述語の追加
- リトライとエラーの連動変更（Issue #11）
- 成功レスポンス側のサイズ上限（Issue #14）

## Open Questions

- なし（issue の「判断を委ねたい点」だったボディ断片の保持方法は、公開フィールド `RawBody` + `Error()` への埋め込みの両方を採用）
