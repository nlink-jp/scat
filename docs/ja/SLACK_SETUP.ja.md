# Slackボットの設定

[English](../en/SLACK_SETUP.md)

bot userを持つSlack appを作成・設定し、必要な操作のscopeを付与してworkspaceへinstall/reinstallします。
bot tokenは `scat profile set token` の非表示端末入力、またはサービスの `SCAT_TOKEN` から渡します。
トークンをコマンド引数にしないでください。設定とserver modeは[README](../../README.ja.md)を参照してください。
scliのユーザー認証とは明確に分離しています。

| 操作 | scope・アクセス |
|------|-----------------|
| 投稿 | `chat:write`。許可された名前・アイコン変更時だけ `chat:write.customize` |
| ユーザーIDへのDM | `im:write` |
| channel名解決・upload参加確認 | `channels:read` / `groups:read`。DM infoは `im:read` / `mpim:read` |
| ユーザー名解決・一覧 | `users:read` |
| グループ名解決・メンバー展開 | `usergroups:read` |
| history・replies export | 会話種別に応じ `channels:history`、`groups:history`、`im:history`、`mpim:history` |
| ファイルdownload | `files:read` と対象ファイルへのアクセス |
| ファイルupload | `files:write` と宛先解決・参加確認の権限 |
| 投稿のためのpublic channel参加 | `channels:join` |
| 作成・topic・purpose | publicは `channels:manage`、privateは `groups:write` |
| 招待 | publicは `channels:manage` / `channels:write.invites`、privateは `groups:write` / `groups:write.invites` |

全scopeをまとめて要求する表ではありません。channel/userの明示的IDなら名前一覧APIを避けます。
uploadはchannel IDでも参加確認のread権限が必要です。private channelにはbotを招待してください。
scatはexportのための自動参加や、ユーザー権限へのfallbackを行いません。

外部操作前に呼び出しごとに `auth.test` で `bot_id` と `team_id` を確認します。
help・ローカル設定・dry-runは認証要求を送りません。現行のreplies資料にはchannelとDMのbot scopeが
記載されています。過去のDM限定という前提や、資料だけで実権限を確認できたという前提を置かず、
リリース前にinstall済みappの実際のアクセスを確認してください。

## ファイル送受信の注意点

uploadは `files.getUploadURLExternal`、返されたHTTPS URLへのraw bytes POST、
`files.completeUploadExternal` の順です。バイト転送にはAPI Bearer/Cookieを付けず、redirectを拒否します。
完了処理にはchannelを明示し、threadへのuploadでは親timestampを使います。
完了処理は一度だけで、再試行しません。宛先への参加が不足する場合、付与scopeでpublic channelへ
参加できるケースを除き、割当前に失敗します。ファイル共有のbroadcastはありません。

downloadはHTTP 200でもログインHTMLが返ることがあります。Bearer送信済みで `files:read` 不足のケースも
含みます。Goは通常、兄弟hostへのredirectでAuthorizationを落とすため、scatは検証済みSlackドメイン内だけ
で再付与し、外部hostやHTTPS降格先には送りません。権限不足を直すために転送範囲の制約を外さないでください。
名前補完・ファイル保存の警告はexport metadataを保持し、必須のhistory/replies取得失敗は全体エラーになります。

SlackはHTML/JSON添付を `text/plain` と分類し、元のバイトを
`application/force-download` で返す場合があります。scatは認証済みSlack host、
`Content-Disposition` のファイル名一致、記録サイズを確認してこのMIME差を許容します。
拡張子やHTTP 200だけでは許可せず、既知のログインHTMLは引き続き拒否します。
実際の往復検証は[BUILD](BUILD.ja.md#実slack-e2e)を参照してください。

実Slackでのpost/upload/downloadテストには、明示的に利用を許可されたfixture workspace/channelが必要です。
オフラインテストの成功は実権限・配信を確認したことにはなりません。

参照: [auth.test](https://docs.slack.dev/reference/methods/auth.test/)、
[replies](https://docs.slack.dev/reference/methods/conversations.replies/)、
[upload割当](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/)、
[upload完了](https://docs.slack.dev/reference/methods/files.completeUploadExternal/)、
[ADRのscope一覧](adr/0001-slack-bot-renewal.ja.md)。
