# Design Document

Issue: [#10](https://github.com/hitoshiichikawa/apple-business-go/issues/10)

## Overview

検証を `Do` の入口 1 箇所（+ リダイレクト用の `CheckRedirect`）に置く。#15 で ページング追従が
`followPages` → `Do` に一本化されたため、`Do` の事前検証だけで `List` / `ListSeq` /
`Relationship` / `Get` / 書き込み系の全経路がカバーされる。

## Goals / Non-Goals

- **Goals**: 全リクエスト・全リダイレクトホップの同一オリジン（scheme+host）強制
- **Non-Goals**: 許可リスト等の柔軟なポリシー、トークン以外のヘッダ制御

## Components and Interfaces

```go
// client.go
type Client struct {
    baseURL string
    origin  *url.URL   // 追加: パース済み baseURL
    ...
}

var errCrossHost = errors.New("applebusiness: refusing cross-host request") // 非公開 sentinel

func sameOrigin(u, origin *url.URL) bool  // scheme と host を大文字小文字無視で比較
```

- `NewClient`: baseURL をパースし scheme/host を検証（不正なら即エラー）。
  `http.Client` に `CheckRedirect` を設定: 同一オリジン以外は `errCrossHost` を wrap して拒否、
  10 ホップ超も拒否（CheckRedirect を設定すると標準の 10 回上限が無効になるため自前で維持）
- `Do`: 冒頭で `url.Parse(rawurl)` → `sameOrigin` チェック。違反は `errCrossHost` を wrap して
  リトライループに入らず return（Req 2.1）
- `httpClient.Do` のエラーが `errors.Is(err, errCrossHost)`（リダイレクト拒否が `*url.Error` に
  wrap されて返る）の場合もリトライせず即 return（Req 2.1）

## Error Handling

- 拒否メッセージ: `applebusiness: refusing cross-host request: "<target>" (base "<baseURL>")`（Req 1.5）
- sentinel は非公開。公開の判別が必要になったらエクスポートを別途検討

## Security Considerations

- oauth2.Transport はトランスポート層で各ホップにトークンを付けるため、net/http の
  クロスホスト Authorization 削除では防げない — CheckRedirect での遮断が必須
- 比較は scheme + host（port 含む）の完全一致。サブドメインも別オリジン扱い

## Testing Strategy

- `TestListRefusesCrossHostNext`: 偽 API が外部ホストの links.next を返す → エラー、外部サーバ受信 0 件（Req 1.1, 1.5, 2.1）
- `TestDoRefusesCrossHostRedirect`: 302 で外部ホストへ誘導 → エラー、外部サーバ受信 0 件（Req 1.2, 2.1）
- 既存 `TestListFollowsPagination` / `TestListSeqIterates`: 同一ホスト追従の回帰なし（Req 1.4）
- 既存テスト全体で `Do` 事前検証の回帰なし（Req 1.3 はテストサーバ URL が常に同一オリジンであることで担保）
