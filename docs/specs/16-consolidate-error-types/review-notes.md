# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: e2a07f4
- Compared to: main..HEAD（コミット `refactor(applebusiness): consolidate error types ...`）

## Verified Requirements

- 1.1 — `APIError` / `ErrorObject` / `decodeAPIError` / `Is*` 述語がすべて errors.go に集約
- 1.2 — 公開 `ErrorObject`（Status/Code/Title/Detail、JSON タグ同一）
- 1.3 — `TestAPIErrorDecodeAndClassifier`（`e.Errors[0].Code` アクセス）が無変更でパス
- 2.1 — 非 JSON ボディは `io.LimitReader(64KiB)` 読み + 200 文字断片を `RawBody` へ
- 2.2 — `TestAPIErrorNonJSONBody` が `RawBody` と `Error()` への断片埋め込みを検証
- 2.3 — `decodeAPIError` は `errBodyReadLimit`(64 KiB) 上限で読む

## Findings

なし

## Summary

エラー定義の集約と named type 化はソース互換を維持。非 JSON ボディの可視化を新規テストで確認した。

RESULT: approve
