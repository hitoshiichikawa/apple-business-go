# Tasks

Issue: [#13](https://github.com/hitoshiichikawa/apple-business-go/issues/13)

- [x] 1. ci.yml にトップレベル permissions を追加
  - `permissions: contents: read`
  - _Requirements: 1.1_
- [x] 2. vuln ジョブを追加
  - setup-go は `go-version: stable`（理由は design.md の Decisions 参照）
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  - _Requirements: 2.1, 2.2_
  - _Depends: #9 の依存更新（同 PR 内の先行コミット）_
- [x] 3. ローカルで green を実証
  - `GOTOOLCHAIN=go1.25.11 govulncheck ./...` → "No vulnerabilities found"
  - _Requirements: 2.3_
