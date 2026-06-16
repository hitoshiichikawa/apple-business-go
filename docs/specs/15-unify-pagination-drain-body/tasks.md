# Tasks

Issue: [#15](https://github.com/hitoshiichikawa/apple-business-go/issues/15)

- [x] 1. `pager` 契約と `followPages` ヘルパを導入
  - `ListResponse[A]` / `RelationshipResponse` に `nextLink()` を実装
  - _Requirements: 1.1_
- [x] 2. `List` / `ListSeq` / `Relationship` を共通ループへ書き換え
  - `List` は `ListSeq` の収集に、`Relationship` は `followPages` 直接利用に
  - 早期 break 時に後続ページを取得しないこと（既存挙動）を維持
  - _Requirements: 1.2, 1.4_
- [x] 3. `drainAndClose` を導入し `Do` の全 Close 経路を置換
  - _Requirements: 2.1_
- [x] 4. 既存テストの無変更パスを確認
  - `go build` / `go vet` / `gofmt -l` / `go test -race ./applebusiness/`
  - _Requirements: 1.3_
