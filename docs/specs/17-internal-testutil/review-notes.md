# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 810f301
- Compared to: main..HEAD（コミット `refactor(test): consolidate duplicated test scaffolding ...`）

## Verified Requirements

- 1.1 — `internal/testutil` に `KeyPEM` / `NewClient` / `WriteJSON` を 1 組のみ定義
- 1.2 — 6 ドメインパッケージのローカルヘルパを削除し `testutil.*` へ置換（`grep "func newClient"` が 0 件）
- 1.3 — `go test -race ./...` 全パス
- 1.4 — 各テストのケース本体は不変（ヘルパ呼び出し名のみ置換）
- 2.1 — ヘルパは `internal/` 配下でモジュール外から import 不可

## Findings

なし

## Summary

7 ファイルの重複（-283 行）を internal/testutil に集約。applebusiness 自身は import cycle 回避のため対象外（要件どおり）。`ecdsa.GenerateKey` は testutil と applebusiness/client_test.go の 2 箇所に収束。

RESULT: approve
