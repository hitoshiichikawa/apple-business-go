# Tasks

Issue: [#14](https://github.com/hitoshiichikawa/apple-business-go/issues/14)

- [x] 1. レスポンスボディの上限化
  - `Do` のデコードを LimitedReader 化、`drainAndClose` の読み捨て上限化
  - _Requirements: 1.1, 1.3_
- [x] 2. `newJTI` のエラー伝播
  - `(string, error)` 化し `buildClientAssertion` で伝播
  - _Requirements: 2.1_
- [x] 3. blueprints の rel 許可リスト検証
  - `RelationshipIDs` / `modifyRel` の入口で `checkRel`
  - _Requirements: 3.1, 3.2_
- [x] 4. `ListRange` の非破壊化
  - 引数 q をコピーしてから filter を Set
  - _Requirements: 4.1, 4.2_
- [x] 5. テスト追加と全体回帰
  - `TestDoRejectsOversizedResponse` / `TestRelValidation` / `TestListRangeDoesNotMutateQuery`、`go test -race ./...` 全パス
  - _Requirements: 1.1, 3.1, 3.2, 4.1, 4.2_
