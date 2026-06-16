# Requirements Document

Issue: [#12](https://github.com/hitoshiichikawa/apple-business-go/issues/12)

## Introduction

`buildClientAssertion` は JWT クライアントアサーションを exp = 180 日（Apple の許容最大値）で
署名していたが、本実装はトークン取得のたびに新規生成しており長い exp を使う理由がない。
署名済みアサーションが漏えいした場合（プロキシのアクセスログ、誤ログ出力等）、第三者は exp まで
それを使ってアクセストークンを取得し続けられる。exp を短くして漏えい時の再利用窓を縮める。

## Requirements

### Requirement 1: アサーション TTL の短縮

**Objective:** As a メンテナ, I want アサーションの有効期限が必要最小限であること, so that 漏えい時の悪用可能期間が最小化される

#### Acceptance Criteria

1. The クライアントアサーション shall exp = 生成時刻 + 10 分で署名される
2. The クライアントアサーション shall クロックスキュー対策として iat を生成時刻の 30 秒前に補正する
3. When テストでアサーションをデコードしたとき, the exp − iat shall TTL(10 分) + iat 補正(30 秒) に一致する
4. The トークン取得フロー（フォーム項目・エンドポイント・スコープ） shall 変更されない

## Non-Functional Requirements

### NFR 1: 互換性・検証

1. The 変更 shall 公開 API を変更しない
2. The 実環境（Apple の実トークンエンドポイント）での動作確認 shall マージ前にメンテナが `examples/smoke-test -token-only` で実施する（Apple 仕様上 exp は「最大 180 日」で下限要求の記載はないが、実測で確定させる）

## Out of Scope

- アサーションのキャッシュ・再利用機構（毎回生成のまま）
- `tokenSkew`（アクセストークン側の余裕）の変更
- トークンエンドポイント URL 等の他の OAuth パラメータ

## Open Questions

- なし（issue の判断ポイント「実環境確認の運用」は、PR では単体テストまで・マージ前にメンテナが
  smoke-test 実施、で確定。手順は本ファイル NFR 1.2 と PR 本文に記載）
