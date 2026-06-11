# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 6a06ffb
- Compared to: main..HEAD（コミット `refactor(applebusiness): unify pagination loops ...`）

## Verified Requirements

- 1.1 — `followPages[P pager]` が唯一の links.next ループ。`List` は `ListSeq` を収集、`Relationship` も `followPages` を使用
- 1.2 — `List` / `ListSeq` / `Relationship` のシグネチャ不変
- 1.3 — `TestListFollowsPagination` / `TestListSeqIterates` を無変更でパス
- 1.4 — `ListSeq` の yield=false 時に handle が false を返し追従停止（早期 break テストでカバー）
- 2.1 — `drainAndClose` が `io.Copy(io.Discard, body)` 後に Close、全 Close 経路で使用

## Findings

なし

## Summary

ページ追従が単一ヘルパに集約され、既存テストが無変更で通ることで挙動同一性を確認した。

RESULT: approve
