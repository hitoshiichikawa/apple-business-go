# Tasks

Issue: [#9](https://github.com/hitoshiichikawa/apple-business-go/issues/9)

- [x] 1. x/oauth2 を v0.30.0 へ更新
  - `go get golang.org/x/oauth2@v0.30.0 && go mod tidy`
  - `go build ./...` / `go vet ./...` / `go test -race ./...` で回帰確認
  - _Requirements: 1.1, 1.2, 1.4_
- [x] 2. govulncheck で CVE 解消を確認
  - 最新パッチツールチェーンで `govulncheck ./...` → "No vulnerabilities found"
  - _Requirements: 1.3_
- [x] 3. dependabot.yml の ignore をバージョン条件付きに修正
  - `versions: [">=0.31.0"]` を追加し、コメントの誤った前提を訂正
  - _Requirements: 2.1, 2.2_
