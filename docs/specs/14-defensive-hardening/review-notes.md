# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 0680791
- Compared to: main..HEAD（コミット `feat: defensive hardening ...`）

## Verified Requirements

- 1.1 — `Do` 成功時のデコードを `io.LimitedReader{N: maxResponseBytes+1}` 化。`TestDoRejectsOversizedResponse` が上限超過エラーを確認
- 1.2 — エラーボディは `errBodyReadLimit`(64 KiB) 上限（#16 で導入）
- 1.3 — `drainAndClose` が `io.LimitReader(body, maxDrainBytes=1 MiB)` で読み捨て
- 2.1 — `newJTI() (string, error)` 化し `buildClientAssertion` が伝播（固定 jti を出さない）
- 3.1 — `RelationshipIDs` / `modifyRel` で `checkRel`。`TestRelValidation` がパス操作風 / 未知 / 空 rel を送信前に拒否、受信 0 件を確認
- 3.2 — 既知 `RelOrgDevices` は従来どおり通過
- 4.1 — `ListRange` が `q` を `merged` にコピーしてから Set。`TestListRangeDoesNotMutateQuery` が不変を確認
- 4.2 — nil q も従来どおり動作

## Findings

なし

## Summary

4 つの独立した堅牢化を実装し、再現可能な 3 項目（サイズ上限 / rel 検証 / q 非破壊）をテストで確認。rand.Read 失敗（Req 2.1）は強制再現不能のためレビューで担保。

RESULT: approve
