# Requirements Document

Issue: [#13](https://github.com/hitoshiichikawa/apple-business-go/issues/13)

## Introduction

`.github/workflows/ci.yml` には `permissions:` ブロックがなく、GITHUB_TOKEN がリポジトリ設定の
デフォルト権限（設定によっては write）で動作する。CI（build / lint）は読み取りしか必要としない。
また依存の既知脆弱性を CI で自動検出する仕組みがなく、実際に x/oauth2 の CVE 該当版（Issue #9）が
検出されないまま残っていた。最小権限の明示と govulncheck の常時実行で再発を防ぐ。

## Requirements

### Requirement 1: GITHUB_TOKEN の最小権限化

**Objective:** As a メンテナ, I want CI のトークン権限が読み取りに限定されること, so that ワークフローや依存アクションが侵害されてもリポジトリへの書き込みができない

#### Acceptance Criteria

1. The CI workflow shall トップレベルで `permissions: contents: read` を宣言し、全ジョブに適用する
2. When push / pull_request で CI が実行されたとき, the 既存 build / lint ジョブ shall 従来どおり成功する

### Requirement 2: 既知脆弱性の自動検出

**Objective:** As a メンテナ, I want 依存・標準ライブラリの既知脆弱性が CI で検出されること, so that CVE 該当版が気付かれないまま残らない

#### Acceptance Criteria

1. When push / pull_request で CI が実行されたとき, the CI shall `govulncheck ./...` を実行する
2. If 到達可能な既知脆弱性が存在するとき, the vuln ジョブ shall 非ゼロ終了で fail する
3. While Issue #9 の x/oauth2 更新が取り込まれている状態, the vuln ジョブ shall 成功する

## Non-Functional Requirements

### NFR 1: CI 所要時間

1. The vuln ジョブ shall 既存ジョブと並列実行され、CI 全体の所要時間を大きく延ばさない（目安: +2 分以内）

## Out of Scope

- リリース自動化等の新ワークフロー追加
- OpenSSF Scorecard アクションの導入
- アクションの SHA ピン留め（メンテナ判断に委ねる。導入する場合は別 issue）
- golangci-lint v2 系への移行

## Open Questions

- なし（SHA ピン留めは Out of Scope として明示的に見送り）
