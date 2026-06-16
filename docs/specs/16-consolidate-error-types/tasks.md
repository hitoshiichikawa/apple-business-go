# Tasks

Issue: [#16](https://github.com/hitoshiichikawa/apple-business-go/issues/16)

- [x] 1. `APIError` / `decodeAPIError` を errors.go へ移動
  - client.go からエラー定義を削除し、errors.go に集約
  - _Requirements: 1.1_
- [x] 2. `ErrorObject` named type を導入
  - JSON タグは従来の匿名 struct と同一に保つ
  - _Requirements: 1.2, 1.3_
- [x] 3. 非 JSON ボディの断片保持を実装
  - 64 KiB 上限読み取り + 200 文字断片を `RawBody` へ、`Error()` に埋め込み
  - _Requirements: 2.1, 2.2, 2.3_
- [x] 4. テスト追加と回帰確認
  - `TestAPIErrorNonJSONBody` を追加、既存テスト全パス
  - _Requirements: 1.3, 2.1, 2.2_
