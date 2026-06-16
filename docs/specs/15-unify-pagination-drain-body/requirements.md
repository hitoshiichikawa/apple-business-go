# Requirements Document

Issue: [#15](https://github.com/hitoshiichikawa/apple-business-go/issues/15)

## Introduction

`applebusiness/client.go` には「`links.next` で endpoint を更新しながらページを辿る」同型の
ループが `List` / `ListSeq` / `Relationship` の 3 箇所に重複しており、ページング追従への変更
（例: Issue #10 の同一ホスト検証）が 3 箇所の同期修正になる。また `Do` はデコード後にボディを
読み切らずに Close しており、HTTP keep-alive 接続の再利用が阻害され得る。観測可能な挙動を変えずに
内部品質を改善する。

## Requirements

### Requirement 1: ページング追従の一本化

**Objective:** As a メンテナ, I want links.next 追従の実装が 1 箇所であること, so that ページング動作への変更を 1 箇所の修正で済ませられる

#### Acceptance Criteria

1. The client パッケージ shall links.next を追従するループをただ 1 箇所だけ持つ
2. The 公開 API（`List` / `ListSeq` / `Relationship` のシグネチャと挙動） shall 変更されない
3. When 既存テスト（`TestListFollowsPagination` / `TestListSeqIterates` 等）を無変更で実行したとき, the テスト shall すべて成功する
4. While 利用側が `ListSeq` のイテレーションを途中で break した状態, the client shall 後続ページのリクエストを発行しない（既存挙動の維持）

### Requirement 2: レスポンスボディの完全ドレイン

**Objective:** As a SDK 利用者, I want ボディが読み切られてから Close されること, so that ページングの連続リクエストで keep-alive 接続が再利用される

#### Acceptance Criteria

1. When `Do` がレスポンス処理を終えるとき, the client shall ボディを EOF まで読み切ってから Close する（成功・エラー・リトライの全経路）

## Non-Functional Requirements

### NFR 1: 互換性・品質

1. The client shall Go 1.23 の `iter.Seq2`（range-over-func）を維持する
2. The 変更 shall `go vet` / `golangci-lint` を新たな警告なしで通過する

## Out of Scope

- 公開 API の追加・変更
- リトライ方針の変更（Issue #11）
- links.next の同一ホスト検証（Issue #10。本リファクタはその準備）
- ボディサイズ上限（Issue #14）

## Open Questions

- なし
