# Contributing to apple-business-go

ご関心ありがとうございます。本プロジェクトは **Apple Business API の非公式 Go SDK** です。

## 開発環境

- Go 1.23 以上（`iter` の range-over-func を使用）。
- 依存は最小限（`golang-jwt/jwt/v5`, `golang.org/x/oauth2`）。新規依存は慎重に。

## ビルドと検証

PR 前に、以下がすべて通ることを確認してください（CI と同じ内容です）。

```bash
go build ./...
go vet ./...
gofmt -l .                 # 出力が無いこと（あれば gofmt -w .）
golangci-lint run ./...    # revive / gosimple など
go test -race ./...
```

`golangci-lint` は次で導入できます（CI と同じバージョン）。

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

## コミット / PR

- コミットは [Conventional Commits](https://www.conventionalcommits.org/)（`feat:` / `fix:` / `docs:` / `chore:` など）。
- ユーザー影響のある変更は `CHANGELOG.md` の `[Unreleased]` に追記してください。
- 公開シンボルの doc コメントは英語、説明的なコメントは日本語で構いません。
- JSON:API のフィールド名は Apple の camelCase に合わせ、列挙値は定数化します（`Service` パターンに従う）。

## API 仕様について

本 SDK は非公式です。エンドポイントや挙動は公開実装から抽出・検証していますが、
不確実な点は `docs/apple-business-api-reference.md` / `docs/apple-business-api-datatypes.md` を参照してください。
実機で確認した事実は `CHANGELOG.md` の `Verified` セクションに残します。

## セキュリティ

脆弱性は公開 issue ではなく、[Security Policy](SECURITY.md) に従って報告してください。
秘密鍵・`client_id`・トークン等の機密情報は、issue / PR / ログに含めないでください。
