# Design Document

Issue: [#13](https://github.com/hitoshiichikawa/apple-business-go/issues/13)

## Overview

CI ワークフロー単体の変更。トークン権限の最小化と、脆弱性スキャンジョブの追加のみで、
ビルド・テスト・リントの既存ジョブには触れない。

## Goals / Non-Goals

- **Goals**: GITHUB_TOKEN の read 限定、push/PR ごとの govulncheck 実行
- **Non-Goals**: 新ワークフロー、SHA ピン留め、lint 設定変更

## Decisions

| 論点 | 決定 | 根拠 |
|---|---|---|
| permissions の置き場所 | workflow トップレベル | 全ジョブ（将来の追加ジョブ含む）に一括適用され、宣言漏れを防ぐ |
| govulncheck の実行方法 | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` | サードパーティアクション（golang/govulncheck-action）への依存を増やさない。@latest で DB・検出ロジックが常に最新 |
| vuln ジョブの Go バージョン | `go-version: stable` | x/vuln 最新版のビルドは新しめの Go を要求する。古いパッチのツールチェーンだと標準ライブラリの修正済み脆弱性が誤検出される（go1.25.1 で 19 件検出 → go1.25.11 で 0 件を実測）。モジュールの最低 Go (1.23) でのビルド可否は build ジョブが担保するため、vuln ジョブは stable で問題ない |

## File Structure Plan

```
.github/workflows/ci.yml    # permissions: contents: read（トップレベル）+ vuln ジョブ追加
```

## Error Handling

- govulncheck は到達可能な脆弱性検出時に exit 3 を返し、ジョブが fail する（Req 2.2 をそのまま満たす）
- 到達不能（imported but not called）の informational findings では fail しない（govulncheck の既定挙動）

## Testing Strategy

- ローカルで `GOTOOLCHAIN=go1.25.11 govulncheck ./...` → "No vulnerabilities found" を確認済み（Req 2.3 相当）
- PR の CI 実行で build / lint / vuln の 3 ジョブが green になることを確認（Req 1.2, 2.1）
- 権限はワークフロー実行ログの "GITHUB_TOKEN Permissions" 欄で `contents: read` を確認
