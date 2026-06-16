# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 30acfd5
- Compared to: main..HEAD（コミット `feat(applebusiness): restrict requests and redirects ...`）

## Verified Requirements

- 1.1 — `Do` 入口で `sameOrigin(u, c.origin)` を検証、違反は送信前に `errCrossHost`。`followPages` 経由で List/ListSeq/Relationship をカバー
- 1.2 — `CheckRedirect` が同一オリジン外を拒否
- 1.3 — `Do` の rawurl 検証（テストサーバは常に同一オリジン）
- 1.4 — `TestListFollowsPagination` 等の同一ホスト追従が回帰なし
- 1.5 — エラーに宛先 URL と base URL を含む
- 2.1 — クロスホスト拒否はリトライループに入らず即 return
- NFR 1.2 — `NewClient` が base URL のパース失敗 / scheme・host 欠落で即エラー

## Findings

なし（`TestListRefusesCrossHostNext` / `TestDoRefusesCrossHostRedirect` で外部サーバ受信 0 件を確認）

## Summary

トークン外部送出経路（汚染 links.next・クロスホストリダイレクト・Do の任意 URL）を同一オリジン制限で塞いだ。oauth2.Transport のホップ毎付与に対し CheckRedirect での遮断が要点。

RESULT: approve
