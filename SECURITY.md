# Security Policy

## Supported Versions

本 SDK は 1.0 未満です。セキュリティ修正は最新のリリース（`v0.x`）に対して提供します。

## Reporting a Vulnerability

脆弱性を発見した場合は、**公開 issue を作成せず**、GitHub の
[Security Advisories](https://github.com/hitoshiichikawa/apple-business-go/security/advisories/new)
から非公開で報告してください。可能であれば、再現手順・影響範囲・想定される修正案を含めていただけると助かります。

状況を確認のうえ、修正方針と公開時期を調整します。

## 秘密情報の取り扱い（利用者向け）

本 SDK は OAuth2 client_credentials + ES256 JWT による認証を行い、**EC 秘密鍵**を扱います。
利用にあたっては次に注意してください。

- 秘密鍵（`.pem`）・`client_id`・`key_id` などの認証情報を **コミット／ログ出力しない**。
- 鍵は呼び出し側がバイト列で渡す設計です。環境変数やシークレットマネージャからの注入を推奨します。
- アクセストークンや JWT クライアントアサーションをログに残さない。
- 取得したアクセストークンは有効期限（既定 60 分）内のみ保持し、不要になったら破棄してください。
