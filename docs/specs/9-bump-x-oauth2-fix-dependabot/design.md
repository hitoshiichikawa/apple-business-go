# Design Document

Issue: [#9](https://github.com/hitoshiichikawa/apple-business-go/issues/9)

## Overview

依存更新のみの小規模変更。コード変更はなく、`go.mod` / `go.sum` の更新と
`.github/dependabot.yml` の ignore 条件の絞り込みを行う。

## Goals / Non-Goals

- **Goals**: CVE-2025-22868 該当版の解消、dependabot の誤った全面 ignore の是正
- **Non-Goals**: Go 最低バージョン変更、他依存の更新、コード変更

## Decisions

| 論点 | 決定 | 根拠 |
|---|---|---|
| 更新先バージョン | v0.30.0 | go 1.23.0 互換の最新。module proxy の `.mod` で確認（v0.27.0〜v0.30.0 = go 1.23.0、v0.31.0+ = go 1.24.0、v0.36.0 = go 1.25.0） |
| `go` ディレクティブ | `1.23` → `1.23.0` を許容 | x/oauth2 v0.30.0 が `go 1.23.0`（パッチ付き表記）を要求するため `go get` が正規化する。最低マイナーバージョンは 1.23 のまま不変 |
| dependabot ignore | `versions: [">=0.31.0"]` | go 1.24 を要求し始める境界。v0.30.0 までの patch/minor 更新 PR は引き続き作成される |

## File Structure Plan

```
go.mod                      # x/oauth2 v0.21.0 → v0.30.0、go 1.23 → 1.23.0（正規化）
go.sum                      # go mod tidy で再生成
.github/dependabot.yml      # ignore に versions 条件を追加、コメント訂正
```

## Error Handling

該当なし（設定・依存のみ）。

## Testing Strategy

- `go build ./...` / `go vet ./...` / `go test -race ./...` の全パス（Req 1.4）
- 最新パッチツールチェーン（`GOTOOLCHAIN=go1.25.11`）での `govulncheck ./...` が
  "No vulnerabilities found" になること（Req 1.3）
- dependabot.yml は GitHub 側でのみ評価されるため、YAML 構文の目視確認 + 次回の weekly run で実挙動確認
