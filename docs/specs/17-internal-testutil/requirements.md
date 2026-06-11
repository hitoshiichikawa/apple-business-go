# Requirements Document

Issue: [#17](https://github.com/hitoshiichikawa/apple-business-go/issues/17)

## Introduction

devices / people / apps / blueprints / configurations / auditevents の各 `*_test.go` に、
「EC P-256 鍵生成 → PEM 化 → 擬似トークンエンドポイント → `NewClient`」という同型の
セットアップ（`testKeyPEM` / `newClient` / `writeJSON`）が 6 回コピーされている。テスト用
クライアントの仕様変更が 6 箇所の同期修正になり、新ドメインパッケージ追加のたびにコピーが
増える。`internal/testutil` に集約する。

## Requirements

### Requirement 1: テストセットアップの単一化

**Objective:** As a メンテナ, I want テスト用クライアントの組み立てが 1 箇所にあること, so that テスト基盤の変更を 1 箇所の修正で済ませられる

#### Acceptance Criteria

1. The モジュール shall 鍵生成・トークンエンドポイント・クライアント組み立てのヘルパを `internal/testutil` パッケージにただ 1 組だけ持つ
2. The 6 つのドメインパッケージのテスト shall ローカルの重複ヘルパを持たず `testutil` を利用する
3. When `go test -race ./...` を実行したとき, the テスト shall すべて成功する
4. The 各テストのケース本体（何を検証するか） shall 変更されない

### Requirement 2: 公開サーフェスの不変

**Objective:** As a SDK 利用者, I want テスト用シンボルが公開 API に現れないこと, so that SDK の godoc / API サーフェスが汚れない

#### Acceptance Criteria

1. The ヘルパ shall `internal/` 配下に置かれ、モジュール外から import できない

## Non-Functional Requirements

### NFR 1: 制約

1. The 変更 shall 新規外部依存を追加しない

## Out of Scope

- テストケースの追加・削除・変更
- 公開のテスティングユーティリティ（`applebusinesstest` 等）の提供
- `applebusiness` パッケージ自身のテストヘルパ（import cycle のため対象外。下記 Open Questions 参照）

## Open Questions

- なし（issue の判断ポイント「applebusiness 自身のテストを対象外とするか」は、
  `applebusiness`（internal テストパッケージ）→ `testutil` → `applebusiness` の import cycle が
  成立しないため**対象外で確定**。トークン交換そのものを検証するテストでもあり、自前セットアップが妥当）
