# Requirements Document

Issue: [#10](https://github.com/hitoshiichikawa/apple-business-go/issues/10)

## Introduction

`applebusiness.Client` は `oauth2.Transport` により全リクエストへ Bearer トークンを自動付与する。
一方で `List` / `ListSeq` / `Relationship` はレスポンス中の `links.next` を検証なしに次の GET 先と
して使い、`http.Client` は `CheckRedirect` 未設定でリダイレクトを追従する。`oauth2.Transport` は
ホップ毎にトークンを再付与するため、net/http 標準の「クロスホスト時に Authorization を落とす」
保護が効かない。汚染された `links.next` や外部ホストへの 3xx により、アクセストークンが
第三者ホストへ送出され得る。Client の発行する全リクエストを base URL と同一オリジン
（scheme + host）に制限する。

## Requirements

### Requirement 1: 同一オリジン外へのリクエスト拒否

**Objective:** As a SDK 利用者, I want トークン付きリクエストの宛先が API ホストに限定されること, so that レスポンス汚染や誘導によるトークン漏えいを防げる

#### Acceptance Criteria

1. If `links.next` が base URL と異なる scheme または host を指すとき, the client shall リクエストを発行せずエラーを返す
2. If API が異なるオリジンへのリダイレクト（3xx）を返したとき, the client shall リダイレクト先へリクエストを送らずエラーを返す
3. If `Do` に base URL と異なるオリジンの絶対 URL が渡されたとき, the client shall リクエストを発行せずエラーを返す
4. When 同一オリジンの `links.next` を追従するとき, the client shall 従来どおりページングを継続する
5. The 拒否エラー shall 拒否した宛先 URL と base URL を含む

### Requirement 2: 拒否時の無駄な再試行の抑制

**Objective:** As a SDK 利用者, I want オリジン違反が即時にエラーになること, so that リトライ待ちで遅延しない

#### Acceptance Criteria

1. If クロスホスト拒否が発生したとき, the client shall バックオフ再試行をせず即座にエラーを返す

## Non-Functional Requirements

### NFR 1: 互換性

1. The 変更 shall 公開 API のシグネチャを変更しない
2. While base URL が不正（パース不能・scheme/host 欠落）な状態, the `NewClient` shall エラーを返す（従来はリクエスト時まで遅延していた失敗の前倒し）

## Out of Scope

- 許可ホストの追加設定（ホワイトリスト等のオプション API）。issue の判断ポイントは「エスケープハッチなし」で確定（`Do` は API パス用と文書化。必要になれば別 issue で追加）
- リトライ方針の変更（Issue #11）

## Open Questions

- なし
