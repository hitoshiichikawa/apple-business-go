# Review Notes

<!-- idd-claude:review round=1 model=claude-fable-5 timestamp=2026-06-11T22:21:08Z -->

## Reviewed Scope

- Branch: security/core-hardening
- HEAD commit: 1beb630
- Compared to: main..HEAD（コミット `feat(applebusiness)!: stop retrying POST on 5xx ...`）

## Verified Requirements

- 1.1 — `retryable` 判定で `resp.StatusCode >= 500 && method != POST`。`TestPostNotRetriedOn5xx` が受信 1 回を確認
- 1.2 — 429 は全メソッド再送。`TestPostRetriedOn429` が受信 2 回・成功を確認
- 1.3 — ネットワークエラー時 `method == POST` で即 return
- 1.4 — POST 5xx は `>= 400` 分岐で `decodeAPIError` を通り、テストが `Errors[0].Code == "INTERNAL"` を確認
- 2.1 — `TestGetRetriedOn500` / 既存 `TestRetryThenSuccess` で GET の従来挙動を確認
- 3.1 — `Do` / `Create` の doc コメントに POST ルールを明記
- 3.2 — CHANGELOG（Unreleased）に behavior change として記載
- 3.3 — README roadmap の「Idempotency review of write retries」を完了に更新

## Findings

なし

## Summary

非冪等 POST の二重実行リスクを 5xx/ネットワークエラー非再送で解消。429 と他メソッドの挙動は維持。breaking change として CHANGELOG に明記済み。

RESULT: approve
