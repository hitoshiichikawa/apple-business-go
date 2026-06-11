# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 41b67f4
- Compared to: main..HEAD（コミット `feat(applebusiness): shorten client-assertion exp ...`）

## Verified Requirements

- 1.1 — `assertionTTL = 10 * time.Minute`、`exp = now + assertionTTL`
- 1.2 — `assertionIatSkew = 30 * time.Second`、`iat = now - assertionIatSkew`
- 1.3 — `TestClientAssertionShortTTL` が `exp - iat == 630s`、jti・aud を検証
- 1.4 — フォーム項目・エンドポイント・スコープは不変（`TestTokenExchangeAndGet` 回帰なし）

## Findings

### Finding 1
- **Target**: NFR 1.2
- **Category**: missing test（実環境確認の未了）
- **Detail**: 短い exp を Apple のトークンエンドポイントが受理するかは実測でのみ確定できる。秘密鍵を要するためエージェントでは未実施。
- **Required Action**: マージ前にメンテナが `go run ./examples/smoke-test -token-only` で token acquired を確認する（tasks.md タスク 3 / PR 本文に明記済み）。**コードレビュー上の欠陥ではなく運用ゲート**のため、approve を妨げない。

## Summary

TTL 短縮と iat 補正は単体テストで確認済み。実トークンエンドポイントでの受理確認のみメンテナ作業として残る（PR 本文に明記）。

RESULT: approve
