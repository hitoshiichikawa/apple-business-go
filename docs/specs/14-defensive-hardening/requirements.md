# Requirements Document

Issue: [#14](https://github.com/hitoshiichikawa/apple-business-go/issues/14)

## Introduction

単体では深刻度が低いが放置すると事故の芽になる防御的な堅牢化 4 点をまとめて対応する:
(1) レスポンスボディを上限なしでデコードしておりメモリを際限なく消費し得る、
(2) `newJTI` が `rand.Read` のエラーを無視し失敗時に固定値の jti になり得る（Go 1.23）、
(3) `blueprints` の関係名 `rel` が未検証で URL パスへ連結されパス操作になり得る、
(4) `auditevents.ListRange` が引数の `url.Values` を書き換え呼び出し側へ副作用が漏れる。

## Requirements

### Requirement 1: レスポンスサイズ上限

**Objective:** As a SDK 利用者, I want 異常に大きい応答が上限で打ち切られること, so that 異常応答でプロセスのメモリが枯渇しない

#### Acceptance Criteria

1. If 成功レスポンスのボディが上限（32 MiB）を超えるとき, the client shall 上限超過が分かるエラーを返す
2. The エラーレスポンスの読み取り shall 上限付き（64 KiB。#16 で導入済み）である
3. The ボディの読み捨て（drain） shall 上限付き（1 MiB。超過時は接続再利用を諦めて Close）である

### Requirement 2: jti の乱数エラー処理

**Objective:** As a SDK 利用者, I want jti が常に CSPRNG 由来であること, so that 失敗時に予測可能な jti のアサーションが署名されない

#### Acceptance Criteria

1. If `rand.Read` がエラーを返したとき, the アサーション生成 shall エラーとして失敗する（固定値の jti で署名しない）

### Requirement 3: 関係名 rel の検証

**Objective:** As a SDK 利用者, I want 未知の関係名が拒否されること, so that 外部入力が rel に渡ってもパス操作にならない

#### Acceptance Criteria

1. If `RelationshipIDs` / `AddTo` / `RemoveFrom` / `Replace` に既知の `Rel*` 定数以外の rel が渡されたとき, the blueprints Service shall HTTP リクエストを発行せずエラーを返す
2. When 既知の `Rel*` 定数を渡したとき, the blueprints Service shall 従来どおり動作する

### Requirement 4: 入力 url.Values の非破壊

**Objective:** As a SDK 利用者, I want 渡した url.Values が変更されないこと, so that 同じ url.Values を複数の呼び出しで安全に再利用できる

#### Acceptance Criteria

1. When `ListRange` を呼び出したとき, the auditevents Service shall 引数の `url.Values` を変更しない
2. While q が nil の状態, the `ListRange` shall 従来どおり動作する

## Non-Functional Requirements

### NFR 1: 互換性

1. The 変更 shall 公開 API のシグネチャを変更しない（検証はエラー返却であり panic しない）
2. The レスポンス上限 shall 正常な大規模ページ応答を妨げない十分大きな値とする

## Out of Scope

- リトライ方針の変更（Issue #11）
- links.next のホスト検証（Issue #10）
- 新規依存の追加

## Open Questions

- なし（issue の判断ポイント「rel は許可リスト照合かエスケープか」は許可リスト照合で確定。
  型安全性が高く、Apple が関係名を追加した場合は Rel* 定数追加と同時に許可リストも更新される構造のため）
