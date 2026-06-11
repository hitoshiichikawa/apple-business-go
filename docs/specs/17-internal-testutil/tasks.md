# Tasks

Issue: [#17](https://github.com/hitoshiichikawa/apple-business-go/issues/17)

- [x] 1. `internal/testutil` パッケージを新規作成
  - `KeyPEM` / `NewClient` / `WriteJSON`（`testing.TB` + 可変長 `Option`）
  - _Requirements: 1.1, 2.1_
- [x] 2. 6 ドメインパッケージのローカルヘルパを削除し呼び出しを置換
  - `newClient(t, h)` → `testutil.NewClient(t, h)`、`writeJSON(...)` → `testutil.WriteJSON(...)`
  - 未使用になった crypto / pem / httptest / json import を整理
  - _Requirements: 1.2, 1.4_
- [x] 3. 全テスト・lint の通過を確認
  - `go test -race ./...` 全パス、`gofmt -l` 差分なし
  - _Requirements: 1.3_
