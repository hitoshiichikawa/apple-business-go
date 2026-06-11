# Design Document

Issue: [#15](https://github.com/hitoshiichikawa/apple-business-go/issues/15)

## Overview

`applebusiness/client.go` 内部のみのリファクタリング。ページ追従を `followPages` ヘルパに
集約し、`List` は `ListSeq` の収集、`Relationship` は同ヘルパの直接利用に書き換える。
あわせてボディの drain ヘルパを導入する。

## Goals / Non-Goals

- **Goals**: links.next ループの単一化、全経路でのボディ drain
- **Non-Goals**: 公開 API 変更、ホスト検証（#10）、サイズ上限（#14）

## Components and Interfaces

```go
// 非公開。ページ型が満たす契約
type pager interface{ nextLink() string }

func (r ListResponse[A]) nextLink() string      { return r.Links.Next }
func (r RelationshipResponse) nextLink() string { return r.Links.Next }

// 唯一のページ追従ループ。handle が false を返したら早期終了
func followPages[P pager](ctx context.Context, c *Client, endpoint string, handle func(P) bool) error
```

- `List[A]` → `ListSeq[A]` を range で収集（エラーは即 return）
- `ListSeq[A]` → `followPages[ListResponse[A]]`。yield が false を返したら handle も false を返し
  追従を停止（Req 1.4）。エラー時は従来どおり `(zero, err)` を 1 回 yield して終了
- `Relationship` → `followPages[RelationshipResponse]` で append
- `drainAndClose(body)` — `io.Copy(io.Discard, body)` してから Close。`Do` の
  成功 / 4xx / 429・5xx リトライ / out==nil の全経路で使用

## Error Handling

- `followPages` は `Do` のエラーをそのまま返す。`ListSeq` のイテレータ契約
  （エラーを 1 回 yield して停止、break 後は yield しない）は従来と同一
- 早期 break 時（handle が false）は `followPages` が nil を返すため、エラー yield は発生しない

## Testing Strategy

- 既存テストを**無変更で**全通し（Req 1.3 / 回帰検知）
  - `TestListFollowsPagination`: 2 ページ追従・件数・順序
  - `TestListSeqIterates`: 遅延列挙と早期 break
- drain は挙動が外部から観測しにくいため、既存テストのパス（接続まわりの回帰なし）で確認
