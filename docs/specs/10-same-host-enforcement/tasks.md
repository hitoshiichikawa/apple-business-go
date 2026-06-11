# Tasks

Issue: [#10](https://github.com/hitoshiichikawa/apple-business-go/issues/10)

- [x] 1. `Client.origin` の導入と `NewClient` での base URL 検証
  - パース不能 / scheme・host 欠落で即エラー
  - _Requirements: NFR 1.2_
- [x] 2. `Do` 入口の同一オリジン検証と非リトライ化
  - `errCrossHost` sentinel + `sameOrigin` ヘルパ
  - _Requirements: 1.1, 1.3, 1.5, 2.1_
- [x] 3. `CheckRedirect` でリダイレクトホップを制限
  - 同一オリジン以外を拒否、10 ホップ上限を自前維持
  - _Requirements: 1.2, 2.1_
- [x] 4. テスト追加と回帰確認
  - `TestListRefusesCrossHostNext` / `TestDoRefusesCrossHostRedirect`、既存テスト全パス
  - _Requirements: 1.1, 1.2, 1.4_
