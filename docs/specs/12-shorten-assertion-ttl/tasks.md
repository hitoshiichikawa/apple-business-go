# Tasks

Issue: [#12](https://github.com/hitoshiichikawa/apple-business-go/issues/12)

- [x] 1. assertionTTL を 10 分へ短縮し iat 補正を追加
  - oauth.go の定数と buildClientAssertion、冒頭コメントの更新
  - _Requirements: 1.1, 1.2, 1.4_
- [x] 2. アサーションのクレーム検証テストを追加
  - `TestClientAssertionShortTTL`（exp − iat / jti / aud）
  - _Requirements: 1.3_
- [ ] 3. マージ前の実環境確認（メンテナ実施）
  - `go run ./examples/smoke-test -token-only` で token acquired を確認
  - _Requirements: NFR 1.2_
