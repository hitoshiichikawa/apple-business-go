# Requirements Document

Issue: [#11](https://github.com/hitoshiichikawa/apple-business-go/issues/11)

## Introduction

`(*Client).Do` は 429 / 5xx を全メソッド一律で最大 4 回まで指数バックオフ再送していた。
POST（作成）は冪等でなく、サーバ側でコミット後に 5xx やタイムアウトが返るケースで再送すると
書き込みが二重実行される。影響先は `devices.Assign/Unassign`（実デバイスの MDM 割り当て変更）、
`blueprints.Create`、`configurations.Create`、`ModifyRelationship`(POST)。`docs/dev/CLAUDE.md`
にも既知課題として記載されていた。429 は「処理されなかった」ことが確実なため POST でも再送可、
5xx・ネットワークエラーは処理済みの可能性があるため POST では再送しない方針とする。

## Requirements

### Requirement 1: POST の再試行制限

**Objective:** As a SDK 利用者, I want POST が二重実行され得る状況で自動再送されないこと, so that デバイス割り当て等の書き込みが重複しない

#### Acceptance Criteria

1. If POST リクエストが 5xx を受けたとき, the client shall 再送せずに 1 回でエラー（`APIError`）を返す
2. If POST リクエストが 429 を受けたとき, the client shall 従来どおりバックオフ再送する
3. If POST リクエストがネットワークエラー（レスポンス未受領）になったとき, the client shall 再送せずエラーを返す
4. When POST が 5xx でエラーになるとき, the APIError shall レスポンスのエラーボディ（JSON:API errors）を保持する（従来はリトライ枯渇時にステータスコードのみだった）

### Requirement 2: 他メソッドの従来挙動維持

**Objective:** As a SDK 利用者, I want 読み取りや冪等な書き込みのリトライが維持されること, so that 一時的な障害への耐性が落ちない

#### Acceptance Criteria

1. When GET / DELETE / PATCH が 429 または 5xx を受けたとき, the client shall 従来どおりバックオフ再送する

### Requirement 3: 方針の文書化

**Objective:** As a SDK 利用者, I want リトライ方針がドキュメントから分かること, so that 呼び出し側で適切な後処理（作成確認など）を設計できる

#### Acceptance Criteria

1. The `Do` / `Create` の doc コメント shall POST の再試行ルールを明記する
2. The CHANGELOG shall 挙動変更（behavior change）として本変更を記載する
3. The README の roadmap shall 「Idempotency review of write retries」を完了として更新する

## Non-Functional Requirements

### NFR 1: 互換性

1. The 変更 shall 公開 API のシグネチャを変更しない

## Out of Scope

- Idempotency-Key 等の重複排除機構（Apple API 側のサポート不明）
- リトライポリシーの設定可能化（`WithRetryPolicy` 等の API 追加）

## Open Questions

- なし（issue の判断ポイント 2 件はともに安全側で確定: PATCH は JSON:API の属性置換で実質冪等とみなし再送可、
  POST のネットワークエラーは到達済みの可能性を考慮し再送しない）
