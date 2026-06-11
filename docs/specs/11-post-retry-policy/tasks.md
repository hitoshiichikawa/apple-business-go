# Tasks

Issue: [#11](https://github.com/hitoshiichikawa/apple-business-go/issues/11)

- [x] 1. `Do` のリトライ判定にメソッド条件を追加
  - 5xx の再送対象から POST を除外、ネットワークエラー時も POST は即 return
  - _Requirements: 1.1, 1.3, 2.1_
- [x] 2. テスト追加
  - `TestPostNotRetriedOn5xx` / `TestPostRetriedOn429` / `TestGetRetriedOn500`
  - _Requirements: 1.1, 1.2, 1.4, 2.1_
- [x] 3. 文書化
  - `Do` / `Create` doc コメント、CHANGELOG（Unreleased / behavior change）、README roadmap 更新
  - _Requirements: 3.1, 3.2, 3.3_
