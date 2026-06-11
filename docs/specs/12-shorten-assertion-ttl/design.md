# Design Document

Issue: [#12](https://github.com/hitoshiichikawa/apple-business-go/issues/12)

## Overview

`applebusiness/oauth.go` の定数変更と iat 補正のみ。トークン取得フロー自体（フォーム・
エンドポイント・ReuseTokenSource によるアクセストークンの再利用）は変えない。

## Goals / Non-Goals

- **Goals**: assertionTTL 180 日 → 10 分、iat の 30 秒バックデート
- **Non-Goals**: アサーションのキャッシュ、tokenSkew の変更

## Decisions

| 論点 | 決定 | 根拠 |
|---|---|---|
| TTL 値 | 10 分 | アサーションは送信直後にのみ検証される使い捨て。数十秒で足りるが、クロックスキューと再送猶予を見て 10 分。Apple 仕様は「最大 180 日」で下限の記載なし |
| iat 補正 | now − 30 秒 | サーバ時計が進んでいる場合に「未来の iat」で拒否されるのを防ぐ定石 |
| 実環境確認 | マージ前に smoke-test | 短い exp の拒否有無は実測でしか確定できない。`examples/smoke-test -token-only` で token acquired を確認 |

## Components and Interfaces

```go
// oauth.go（変更箇所のみ）
const (
    assertionTTL     = 10 * time.Minute   // 旧: 180 * 24 * time.Hour
    assertionIatSkew = 30 * time.Second   // 新規
)
// buildClientAssertion: "iat": now.Add(-assertionIatSkew).Unix()
```

## Error Handling

変更なし（トークンエンドポイントが exp を拒否した場合は従来どおり
`applebusiness oauth: token failed (...)` がそのまま返る）。

## Testing Strategy

- `TestClientAssertionShortTTL`: トークンエンドポイントに届いた `client_assertion` を
  `ParseUnverified` でデコードし、exp − iat = 630 秒、jti / aud の妥当性を検証（Req 1.1–1.3）
- 既存 `TestTokenExchangeAndGet`: フォーム項目の回帰なし（Req 1.4）
- マージ前の実環境確認: `go run ./examples/smoke-test -token-only`（NFR 1.2、メンテナ実施）
