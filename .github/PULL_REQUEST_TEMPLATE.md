## 概要

<!-- 変更内容と目的 -->

## チェックリスト

- [ ] `go build ./...` / `go vet ./...` が通る
- [ ] `gofmt -l .` が空（整形済み）
- [ ] `golangci-lint run ./...` が通る
- [ ] `go test -race ./...` が通る
- [ ] ユーザー影響のある変更は `CHANGELOG.md` の `[Unreleased]` に追記した
- [ ] 公開 API を変更した場合は、その旨を本文に記載した

## 関連 issue

<!-- 例: Closes #1 -->
