# Slackボットの設定

[English](../en/SLACK_SETUP.md)

## マニフェストからアプリを作成

[slack-app-manifest.json](../../slack-app-manifest.json)を使うと、botと権限を一括で設定できます。
このcheckoutから作る配布アーカイブにも、binaryと同じ場所に同梱します。
トークン・workspace ID・callback URLを含まないため、複数workspaceで使い回せます。

1. [Your Apps](https://api.slack.com/apps)で **Create New App → From a manifest** を選び、
   botを使うworkspaceを指定します。
2. **JSON** を選択してファイル全文を貼り付け、権限の確認画面へ進みます。
   内容を確認してアプリを作成します。アプリ名・bot表示名の既定値は `scat` です。
   変更したい場合は、貼り付ける前にマニフェストの表示名を変更してください。
3. **Install to Workspace** を実行します。管理者承認が必要なworkspaceでは承認を申請します。
   インストール後、**OAuth & Permissions** の **Bot User OAuth Token** をコピーします。
4. 以下を実行し、非表示の入力欄へtokenを貼り付けます。使うのはbot token（`xoxb-`）です。
   user tokenやapp-level tokenではありません。

```sh
scat config init
scat profile set token
scat profile set channel C0123456789
```

アクセス対象のチャネル、特にprivate channelにはbotを追加してください。
マニフェストが設定するのはscopeであり、チャネルへの参加や全会話へのアクセスではありません。
サービスでは同じbot tokenを `SCAT_TOKEN` とし、`SCAT_MODE=server` を設定します。
詳細は[README](../../README.ja.md)を参照してください。tokenはマニフェストやコマンド引数に書きません。

テンプレートには既存機能の権限をまとめています。テキスト・rich投稿と名前/アイコン変更、
ファイル送受信、チャネル/スレッドexport、チャネル作成・既存チャネルへの追加招待、
ユーザー/グループ検索に対応します。作成と招待は `channels:manage`・`groups:write` で対応するため、
招待専用scopeの重複追加は不要です。user scopeは要求しません。

scatはWeb APIを使うため、Events API・Socket Mode・Incoming Webhook・公開サーバー・
OAuth redirect URLは不要です。期限付きOAuth tokenのrefreshを実装していないため、
テンプレートではtoken rotationを無効にしています。stailで使うapp-level tokenやSocket Mode設定も不要です。

## 既存アプリの更新

既存のscatアプリでは、現在の **App Manifest** をバックアップしてからJSON editorを開き、
`oauth_config.scopes.bot` をテンプレートのbot scope一覧でまとめて置き換えます。
その他のアプリ設定は維持してください。保存後、**Reinstall to Workspace** を行って追加権限を付与します。
マニフェストの更新だけでは、インストール済みtokenの権限は更新されません。
scatに必要な、rotationを使わないbot token設定を維持してください。

## 権限の一覧とカスタマイズ

付属テンプレートは投稿専用の最小構成ではなく、既存機能をまとめて使える構成です。
用途を限定する場合はコピーから不要なscopeを削除して取り込みます。
その際に使えなくなる操作は下表で確認できます。テンプレートをそのまま使う場合、権限の個別追加は不要です。

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

channel/userの明示的IDなら名前一覧APIを避けます。
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

マニフェストの参照: [設定ガイド](https://docs.slack.dev/app-manifests/configuring-apps-with-app-manifests/)、
[フィールド仕様](https://docs.slack.dev/reference/app-manifest/)。
リポジトリのテストはJSON構造・対応するbot scope・未対応の認証モードが無効であることを確認します。
Slack上のアプリ作成・インストールは行いません。
