# scat

[English](README.md)

サービスが**ボット権限**で使うSlack CLIです。人がユーザー権限で使う姉妹ツールは
`scli`です。複数workspaceのprofileは維持し、未使用のマルチサービス機構を撤去します。

刷新の仕様は[ADR-0001](docs/ja/adr/0001-slack-bot-renewal.ja.md)、破壊的変更への対応は
[移行手順](#v1からの移行)を参照してください。

## セットアップ

ビルドにはGo 1.25以降が必要です。公開済みバージョンは
[Releases](https://github.com/nlink-jp/scat/releases)から取得できます。このcheckoutのビルド:

```sh
make build                  # dist/scat
make check                  # go vet ./... + go test ./...
scat config init
scat profile set token      # 非表示の端末入力。トークンを引数に渡さない
scat profile set channel C0123456789
```

[slack-app-manifest.json](slack-app-manifest.json)を
**Create New App → From a manifest → JSON** で取り込むと、bot権限を一括設定できます。
[セットアップ・既存アプリの更新](docs/ja/SLACK_SETUP.ja.md) · [ビルドとテスト](docs/ja/BUILD.ja.md)

実Slackでの動作確認には専用チャンネルで `make e2e` を実行します。
CLIをビルドし、実際の投稿・スレッド・stream・ファイル往復・exportを照合します。
[実E2Eの設定](docs/ja/BUILD.ja.md#実slack-e2e)を参照してください。設定不足はskipではなく失敗になります。

設定ファイルは `~/.config/scat/config.json` です。新規ディレクトリは0700、資格情報ファイルは0600。
既存ファイルの権限が広すぎる場合は警告します。default profileには最初は資格情報がなく、
黙って投稿することはありません。外部操作前に `auth.test` でbotとworkspaceを確認します。
user/app-level tokenは拒否し、scliの資格情報や `SLACK_TOKEN` は参照しません。

```sh
scat profile add another-workspace --channel '#general'
scat profile list
scat profile use another-workspace
scat --profile another-workspace channel list --json
scat profile set limits.max_file_size_bytes 1073741824
scat profile set limits.max_stdin_size_bytes 10485760
scat profile remove unused-profile
scat cache clear
```

`profile add` は `--channel`、`--username`、`--limits-max-file-size-bytes`、
`--limits-max-stdin-size-bytes` に対応します。トークンは後から `profile set token` で設定します。
`profile set` は `channel`、`username`、`token`、上記2つのlimitキーに対応します。
`--profile` は `profile set` の対象選択にも使えます。default/使用中profileは削除できません。
上限は非負で、**0は保存・再読込後も無制限**です。名前解決のcacheは呼び出し内だけで、
`cache clear` は永続cacheがないことを報告します。設定コマンドは `--json` を拒否します。

## サービスモード

`SCAT_MODE=server` を設定し、サービスの秘密情報管理機構から `SCAT_TOKEN` を注入します。
任意の変数は `SCAT_CHANNEL`、`SCAT_USERNAME`、`SCAT_MAX_FILE_SIZE`、`SCAT_MAX_STDIN_SIZE`。
既定値はファイル1 GiB、stdin 10 MiBで、単位はバイトです。`SCAT_CACHE_DIR` は任意の永続cache用に
予約されていますが、この実装は永続cacheを書きません。

server modeはprofileファイルを読まず、対話入力をせず、`--config`・`--profile`・ローカル設定操作を
拒否します。不正なmodeや旧 `SCAT_PROVIDER` は明示的にエラーにします。

## コマンド

共通フラグは `--config`、`--profile/-p`、`--quiet/-q`、`--debug`、`--json`、`--version/-V`。
quietはstderrの補助情報だけを抑制し、結果・警告・エラーは消しません。
debugはトークンや本文を含めず、設定選択の診断を表示します。

```sh
scat post 'Hello' -c '#general'
printf 'Hello\n' | scat post
scat post --from-file message.txt --user U0123456789
scat post 'Reply' -c C0123456789 --thread 1704067200.123456
scat post --format blocks '[{"type":"divider"}]'
scat post --format payload --from-file message.json
scat post 'Preview' --dry-run
```

入力の優先順は引数、`--from-file`、stdinです。`--format` は `text`、`blocks`
（配列またはblocksを含むobject）、`payload` に対応します。payloadの対象フィールドは
`text`、`blocks`、`attachments`、`unfurl_links`、`unfurl_media`、`mrkdwn`。
宛先と実行名義はJSONでなくCLI/configから決めます。追加フラグは `--username`、`--icon-emoji`、
`--thread`、`--unfurl-links`、`--unfurl-media`、`--mrkdwn`。
明示したCLIの真偽値はpayloadより優先します。名前・アイコン変更にはSlackの権限と適切な許可が必要です。

postはtimestampを出力し、`--json` なら `{"ts":"...","channel":"..."}` を出力します。
`--user` はDMを開き、明示的な `--channel` とは併用できません。

```sh
tail -f service.log | scat post --stream -c C0123456789
printf 'pipeline text\n' | scat post --tee
```

streamは3秒ごと、または4,000 Unicode文字ごとに送信し、EOFで残りを送ります。
入力・送信失敗とキャンセルはエラーです。stdin上限はstreamを含む呼び出し全体に適用します。
thread・書式指定は全バッチに適用します。`--tee` はtext stdinをstdoutへ複製し、結果IDを抑制します。
`--json`、引数・ファイル入力、構造化形式との併用は拒否します。
streamはtext stdinだけに対応し、`--dry-run` を拒否します。

```sh
scat upload --file report.pdf -c C0123456789 --comment 'Report'
cat report.pdf | scat upload --file - --filename report.pdf --user U0123456789
scat upload --file report.pdf -c C0123456789 --thread 1704067200.123456
```

uploadにも `--dry-run` があり、stdinには `--filename` が必須です。
短縮形は `-f`（`--file`）、`-c`（`--channel`）、`-m`（`--comment`）です。ファイル入力は通常ファイルに限ります。
入力はメモリ使用量を制限しながらprivate一時snapshotへ固定します。割当前に宛先への参加状況を確認し、
バイト転送後に明示的なchannelと任意の**親**timestampで共有を完了します。
完了成功後だけfile IDを出力し、`--json` では `{"files":[{"id":"..."}],"channel":"..."}` を返します。
完了処理は429を含め自動再送しません。結果が不明ならfile IDと段階を報告します。
ファイル共有のbroadcastは非対応です。

```sh
scat channel list --json
scat user list --json
scat channel create alerts --topic 'Service alerts' --description 'Automation' --invite U0123456789
scat channel create incident-2024-01 --private --invite U0123456789
scat channel invite C0123456789 U0123456789 @on-call
```

一覧は選択profileだけを対象にし、JSONは配列です。ID指定なら名前一覧取得を避け、曖昧な名前はエラーにします。
`channel invite` は既存チャネルへの追加招待です。作成時の `channel create --invite` とは別に使えます。
招待はユーザー・ユーザーグループに対応し、グループIDならユーザー名検索を避けます。
作成はIDまたは `{"id":"...","name":"..."}` を出力します。
招待JSONは `{"channel":"...","users":["..."]}`、通常表示はstderrです。
`channel create --private` はprivate channelを作成します。botは参加している間だけ扱えます。
create/inviteは `--dry-run` に対応し、その際は名前解決しません。
作成後のtopic・purpose設定や招待が失敗した場合は作成済みIDを報告し、自動再作成・削除を行いません。

## Export

```sh
scat channel export C0123456789 --output messages.json --save-dir attachments
scat channel export '#general' --start 2024-01-01T00:00:00Z --end 2024-02-01T00:00:00Z
scat channel export C0123456789 --format text
```

exportは[scliのデータモデル](docs/ja/EXPORT_FORMAT.ja.md)に合わせ、リッチ添付、raw blocks/text、
ファイルmetadataを保持します。既定はJSON、`--output -` はstdoutです。
`--format text` は同じ取得データを表示し、`--json` とは併用できません。

**期間は親を排他的な境界で選び、選択したスレッドの返信をすべて取得します。**
期間外の返信も含まれます。選択されなかった古い親への新しい返信を網羅する機能ではありません。
出力は親とその返信の順で、全体の時系列順ではありません。broadcast返信の重複を除去します。

メッセージ・スレッド取得失敗では、出力前にexport全体をエラーにします。名前補完やファイル保存の失敗では
警告とID/metadataを保持し、未保存ファイルの `local_path` は空文字になります。
`--quiet` でも警告は表示します。出力・添付ファイルは書込成功後だけ置き換えます。
後段でexportが失敗しても、それまでに保存した添付は残ることがあります。
scliと同様に、exportのメモリ使用量は取得履歴に比例します。

downloadは認証redirectを制限し、HTTP 200のログインHTMLを添付と区別します。
正当なHTML/JSONはmetadataと応答を照合して保持し、判別不能なら警告して保存に失敗します。
添付名から指定ディレクトリ外へ書き込ませず、既存のsymlink保存先は拒否します。
APIは30秒、ファイル転送は30分のtimeoutで、キャンセル可能です。HTTP 429の再試行回数を制限し、
結果不明の変更操作とupload完了処理は再送しません。

## v1からの移行

更新前に設定とスクリプトをバックアップしてください。**全profile**から `provider` と `endpoint` を削除し、
`SCAT_PROVIDER` を解除します。旧フィールドは移行案内付きエラーにし、mock設定を黙って実botへ変換しません。
botの資格情報はpromptまたはサービス環境変数から設定します。既存exportは変更しません。

| v1 | v2 |
|----|----|
| `export log --channel X` | `channel export X` |
| `--start-time`、`--end-time` | `--start`、`--end` |
| `--output-files DIR` | `--save-dir DIR`（保存先を明示。autoなし） |
| `--output-format` | exportの `--format` |
| `--silent`、`--iconemoji`、`--noop` | `--quiet`、`--icon-emoji`、`--dry-run` |
| `--provider`、uploadの `--filetype` | 撤去 |
| 全profileを一括一覧 | `--profile` で個別選択 |
| メンション置換・全体時系列順 | 原文・スレッド単位の順序 |

恒久的な互換aliasや実行時mock providerはありません。テストではHTTPとI/Oを注入します。
dry-runはローカル入力を検証し、秘密情報を含まない要約をstderrへ表示します。成功IDは出力せず、
botの本人確認・外部権限確認も行いません。
