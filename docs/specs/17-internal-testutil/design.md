# Design Document

Issue: [#17](https://github.com/hitoshiichikawa/apple-business-go/issues/17)

## Overview

テストコードのみのリファクタリング。6 ドメインパッケージに重複していたセットアップヘルパを
`internal/testutil` へ移動し、各テストの呼び出しを置換する。プロダクションコードと
公開 API には触れない。

## Goals / Non-Goals

- **Goals**: 重複ヘルパの単一化（-283 行）、新ドメインパッケージ追加時の再利用
- **Non-Goals**: テストケースの意味変更、公開テストユーティリティ化、applebusiness 自身のテスト変更

## Components and Interfaces

```go
// internal/testutil — モジュール外非公開
func KeyPEM(t testing.TB) []byte                  // EC P-256 鍵を都度生成し SEC1 PEM で返す
func NewClient(t testing.TB, h http.Handler,
    opts ...applebusiness.Option) *applebusiness.Client  // 擬似 API + 擬似トークン端点 + Client
func WriteJSON(t testing.TB, w http.ResponseWriter, status int, v any)
```

- 旧ヘルパとの差分: `testing.TB` 受け（`*testing.T` 限定をやめ将来のベンチ等にも使える）、
  `opts ...Option` 可変長引数（`WithMaxRetries` 等を渡せるよう拡張。既存呼び出しは無引数のまま）
- import 方向: `devices` 等のテスト → `testutil` → `applebusiness`。
  `applebusiness` の internal テストが `testutil` を import すると cycle になるため対象外（要件の Out of Scope）

## File Structure Plan

```
internal/testutil/testutil.go            # 新規（唯一のヘルパ実装）
devices/devices_test.go                  # ヘルパ削除・呼び出し置換
devices/activities_test.go               # 呼び出し置換のみ（ヘルパは devices_test.go 側にあった）
people/ apps/ blueprints/ configurations/ auditevents/  # 同パターン
```

## Error Handling

ヘルパ内の失敗は従来どおり `t.Fatal` / `t.Fatalf`（テスト即失敗）。

## Testing Strategy

- リファクタ自体の検証 = 既存テスト全体の無変更パス（`go test -race ./...`）
- `grep -rn "ecdsa.GenerateKey"` が testutil と applebusiness/client_test.go（意図的残置）の
  2 箇所のみになることを確認
