# Requirements Document

Issue: [#9](https://github.com/hitoshiichikawa/apple-business-go/issues/9)

## Introduction

`go.mod` が要求する `golang.org/x/oauth2 v0.21.0` は CVE-2025-22868（GO-2025-3488、
`golang.org/x/oauth2/jws` の不正トークンパースによる過大メモリ消費）の該当バージョンである。
本 SDK は `jws` を直接呼ばないため実到達性は低いが、公開 SDK の最小バージョンは利用側に伝播し、
脆弱性スキャナに警告される。さらに `.github/dependabot.yml` の ignore 設定が「x/oauth2 0.22.0+ は
go 1.24/1.25 要求」という誤った前提で更新を全面停止しており、修正版への自動追従を妨げている
（実際は v0.30.0 まで go 1.23.0 互換）。

## Requirements

### Requirement 1: x/oauth2 の脆弱性該当バージョンの解消

**Objective:** As a SDK 利用者, I want 依存に既知 CVE 該当版が含まれないこと, so that 自プロジェクトの脆弱性スキャンで本 SDK 由来の警告が出ない

#### Acceptance Criteria

1. The go.mod shall `golang.org/x/oauth2` を v0.27.0 以上（採用値: v0.30.0）で要求する
2. The go.mod shall 最低 Go バージョンとして 1.23 系（`go 1.23.0`）を維持する
3. When `govulncheck ./...` を最新パッチのツールチェーンで実行したとき, the scan shall GO-2025-3488 / CVE-2025-22868 を報告しない
4. When `go build ./...` / `go vet ./...` / `go test -race ./...` を実行したとき, the module shall すべて成功する

### Requirement 2: dependabot ignore の前提修正

**Objective:** As a メンテナ, I want dependabot が go 1.23 互換の範囲で x/oauth2 更新 PR を出せること, so that 将来のセキュリティ修正に自動追従できる

#### Acceptance Criteria

1. The dependabot 設定 shall `golang.org/x/oauth2` の ignore を `versions: [">=0.31.0"]` のバージョン条件付きに限定する
2. The ignore コメント shall 実際のバージョン境界（v0.31.0+ が go 1.24、v0.36.0+ が go 1.25 要求）を正しく説明する

## Non-Functional Requirements

### NFR 1: 互換性

1. The SDK shall 公開 API・公開挙動を一切変更しない
2. The go.mod shall 新規依存を追加しない

## Out of Scope

- 最低 Go バージョンの引き上げ（1.24 化して v0.31+ へ追従するかは別判断）
- `golang-jwt/jwt/v5` の更新（v5.3.1 に既知 CVE なし）
- CI への govulncheck 追加（Issue #13）

## Open Questions

- なし
