# ADR-0001: scat をサービス向け Slack CLI として刷新する

| Field | Value |
|-------|-------|
| Status | **Accepted** — 2026-09-22にメンテナーが実装への進行を承認 |
| Date | 2026-09-22 |
| Binds | scat |
| Decision makers | nlink-jp maintainers |
| Triggered by | 未使用のプロバイダ抽象化と、姉妹 Slack ツールとの動作の差 |

English: [English](../../en/adr/0001-slack-bot-renewal.md).

## Context

scli は人がユーザー権限で使う。scat はサービスがボット権限で使う。
この棲み分けを維持し、scat を Slack 専用にする。メンテナーは後方互換性より
動作の統一を優先することに合意した。本記録は実装前に承認した仕様である。実装はv2開発ブランチに反映し、公開リリースと実Slack検証は別gateとする。

比較基準は scat `24cfd70`、scli `854e6a0`、swrite `c47d9e3`、stail `1323ef2`。
スレッド、リッチ添付、Block Kit、ファイル保存を扱う scli の export を基準にする。
ボット投稿とパイプ利用は swrite を参照する。いずれも無条件にコードをコピーしない。

現行 scat の登録済みプロバイダは Slack とテスト用の2種類だけである。
機能判定フラグ、廃止済み endpoint、プロバイダ選択、サービス別分岐には、
2つ目の実サービスがない。生成時には必要性に関係なくチャンネル・ユーザー・
グループ一覧を取得する。export はリッチコンテンツを失い、一部の取得・保存失敗で
スレッド全体やファイル情報を落とす。該当する境界を作り直す理由はここにあり、
既存のテスト済み動作をすべて捨てる理由ではない。

## Decision

### 1. 製品の範囲

- `scat` という名称、リポジトリ、chatops-series 所属を維持する。現行は1.xのため、
  破壊的変更を含む次版は **v2.0.0** とする。
- 投稿、DM送信、アップロード、stdinストリーミング、export、チャンネル・ユーザー一覧、
  チャンネル作成、ユーザー・グループ招待、名前付きボットプロファイルを残す。
- プロバイダ登録、Capabilities、`endpoint`、`SCAT_PROVIDER`、`--provider`、
  利用者向けの `mock` / `test` モードを撤去する。
- 複数ボットの設定は残す。複数サービスへの対応は撤去する。
- ボット資格情報だけを受け付ける。ユーザー・アプリレベルトークンは診断付きで拒否し、
  scli の資格情報を借用したり別トークンへ切り替えたりしない。ユーザーOAuth、検索・未読管理、
  Socket Mode、Discord、Teams は今回追加しない。
  各呼び出しの最初の外部操作前に `auth.test` を実行し、空でない `bot_id` とworkspaceの
  同一性を確認する。トークンのprefixだけを証拠にしない。結果を再利用するのは同一呼び出し内だけ。
  ローカル設定・help/version・dry-runでは実行しない。dry-runは認証済みの実行主体を
  検証したとは扱わない。
- stail と swrite は独立したツールとして維持する。廃止や全ツール共通ライブラリ化は
  別の判断とし、この ADR は scat だけを拘束する。

### 2. コマンドと出力

| 操作 | 新しいインターフェース |
|------|------------------------|
| 投稿 | `scat post [text] -c <channel>` または `--user <user>`。本文の優先順は引数、`--from-file`、stdin |
| リッチ投稿 | swrite に合わせて `--format text|blocks|payload`。payload は text、blocks、attachments、unfurl指定を保持 |
| スレッド投稿 | `post --thread <ts>`。Slack timestamp文字列を指定 |
| アップロード | `scat upload --file <path|-> -c <channel>`。`--filename`、`--comment`、`--thread`、DM用の `--user` |
| Export | `scat channel export <channel> --output <path|-> --start <RFC3339> --end <RFC3339> --save-dir <dir>` |
| 一覧 | `scat channel list`、`scat user list`、任意の `--json`。選択したプロファイルだけを対象にする |
| 管理 | `channel create <name>` と `--private`、`--topic`、`--description`、`--invite`。`channel invite <channel> <user-or-group>...` |
| ローカル設定 | `config init`、`profile add/list/use/set/remove`、`cache clear` |

共通フラグは `--config`、`--profile/-p`、`--quiet/-q`、`--debug`、`--json`、
`--version/-V`。`--quiet` は stderr の案内だけを抑制し、データ・警告・エラーは
抑制しない。診断には資格情報やリクエスト本文全体を含めない。

`post` の通常stdoutは swrite と同じ投稿timestamp。`--json` は scli と同じ
`{"ts":"...","channel":"..."}`。stream は成功したバッチごとに結果を出す。
`--tee` は stdout を stdin の複製にする明示的なモードとし、投稿結果は出さず、
`--json` と併用不可とする。upload はファイルIDを1行ずつ、`--json` なら
`{"files":[{"id":"..."}],"channel":"..."}`。作成はチャンネルID、または
`{"id":"...","name":"..."}`。招待は診断のみ、`--json` 指定時には成功後に
`{"channel":"...","users":["..."]}`。一覧JSONは配列とし、プロファイル名をキーにしない。
設定コマンドは人向け出力を維持し、専用の出力仕様を定めるまで `--json` は拒否する。

export は `--json` の有無によらず既定でJSON。既存のテキスト表示は
export `--format text` として残し、同一データモデルを表示するだけにする。
`--format text` と `--json` は併用不可。空のJSONコレクションは `[]`。

post/upload/create/invite の `--noop` は `--dry-run` に置き換える。
ローカル入力を検証し、秘匿化した操作概要をstderrへ出す。API呼び出し、Slackへの変更、
成功IDの出力は行わない。参照操作と `--stream` では拒否する。権限の実確認や名前解決は
できないと説明する。本番用のテストプロバイダは不要になる。

### 3. ボット設定とサービス実行

`~/.config/scat/config.json` の `current_profile` と `profiles` を維持する。
プロファイルは `token`、`channel`、`username`、`limits`。
ディレクトリは0700、ファイルは0600で作成し、既存資格情報ファイルの権限が緩ければ警告する。
トークンをコマンド引数に置かない。安全な入力プロンプトは端末付きの明示的な
profile設定操作だけに限定し、サービスコマンドでは入力を求めない。

`SCAT_MODE=server` では `SCAT_TOKEN`、`SCAT_CHANNEL`、`SCAT_USERNAME`、
`SCAT_MAX_FILE_SIZE`、`SCAT_MAX_STDIN_SIZE`、任意の `SCAT_CACHE_DIR` だけから設定する。
プロファイル・キーチェーンを読み書きしない。`--config`、`--profile`、設定管理コマンドは
拒否する。不正なモード値はエラー。ファイル1 GiB、stdin 10 MiBという既定値と、
明示的な0を無制限とする選択肢を維持する。入力を読む前にサイズ値を検証する。

設定は呼び出しごとに一度だけ解決して注入する。パッケージグローバルや検査のない
contextの型アサーションに置かない。チャンネル・ユーザー・グループは必要時だけ解決し、
ID指定では一覧APIを呼ばない。曖昧な名前は先頭の候補を選ばずエラーにする。
serverモードの永続キャッシュは任意。ボット・ワークスペースの同一性で分離し、
資格情報変更時に無効化する。キャッシュの内容・パス名にトークンを含めない。

### 4. Export は scli を基準にする

scli と同じ `export_timestamp`、`channel_name`、`messages` のエンベロープを使う。
今回、新しいエンベロープやschema versionを追加しない。

| 項目 | 仕様 |
|------|------|
| `export_timestamp` | UTC RFC3339。テストでは時計を注入 |
| `channel_name` | 解決した名前に `#` を付ける。解決失敗時は scli と同じ `#<channel ID>` と警告 |
| `user_id` | APIの `user` があればそれを使い、なければ `bot_id` |
| `user_name` | 解決した表示名。空なら省略 |
| `post_type` | `bot_id` があれば `bot`、なければ `user` |
| `timestamp` / `timestamp_unix` | RFC3339 UTC / 未加工のSlack timestamp文字列。識別子をfloat64にしない |
| `text` | APIの原文。メンションを表示名に置換しない |
| `files` | 空でも `[]`。各要素は `id`、`name`、`mimetype`、`local_path` |
| `local_path` | 必ず文字列を出す。保存成功時は絶対パス、その他は `""`。scli の実際の出力型に合わせる |
| `attachments` | scli が定めるlegacy attachmentの項目。なければ省略 |
| `blocks` | raw JSON配列。なければ省略 |
| `thread_timestamp_unix` | APIの `thread_ts`。なければ省略。親では自身のtimestampと等しい場合がある |
| `is_reply` | `thread_ts` が存在し、`ts` と異なる場合にtrue |

親メッセージを古い順に並べ、各親の直後にその返信を古い順に置く。
**スレッド単位の順序**であり、全メッセージの時系列ソートではない。
historyとrepliesの両方に現れるbroadcast返信も含め、`(channel ID, ts)` で重複排除し、
選択した親に属する返信はその親の直後へ置く。親が選択されずhistoryだけに現れる
broadcast返信も一度だけ保持する。

`--start` / `--end` は境界を含めずhistoryのメッセージを選び、選択した親の
スレッド全体を展開する。scli と同様、期間外の返信を含み得る。
選択されなかった古い親への返信を網羅的に探索しない。
**親の選択期間**であって、全投稿の活動時間窓ではないことを明示する。
オフセット・小数秒を含むRFC3339を受け付け、API境界でSlackのマイクロ秒精度を保持する。
start >= end は拒否する。

要求件数より短いページでもcursorがあれば続ける。`has_more` とcursorを整合して扱い、
cursor再出現を検出する。`has_more=true` なのに使えるcursorがなければ、成功扱いや
無限ループにせずエラー。history・thread取得失敗はexport全体のエラーにする。
名前解決失敗は警告してID・本文を保持する。ファイル保存失敗は警告してメタデータを残し、
`local_path` を空にする。メッセージ取得が完了したexport自体は失敗にしない。

scli と同じく、完全なexportを組み立ててから出力する。ファイル出力は同じディレクトリの
一時ファイルへ書き、成功時にrenameする。取得失敗で既存出力を壊さず、stdoutにもJSONを出さない。
stdoutの書き込み失敗はエラーとし、既に書いたバイトは巻き戻せない。
途中失敗時に保存済み添付が残ることはあるが、ロールバックとして利用者のディレクトリを
削除しない。メモリ消費は scli と同じく履歴量に比例する。メモリ上限付きexportは今回の範囲外。

ファイルは保存先配下の `<fileID>_<safe basename>` へ保存する。不正なパス要素や
既存のsymlink出力先を拒否し、開いたディレクトリ（`os.Root`）を基点として親のリンク経由でも
外へ書かない。一時ファイルとatomic replacementで中断時の不完全ファイルを完成品として扱わない。
認証付きダウンロードの初回URLは、userinfo・非標準portのないHTTPS、かつ `slack.com` または
ラベル境界付きの `.slack.com` 配下だけを許可する。未知の初回hostにはtokenを送らずエラー。
許可hostの追加はSlackの所有と実際の必要性を確認してから行う。
scli のredirect保護を維持し、各hopを直前でなく最初のURLに照らして判定する。
HTTPSからのdowngradeは中断する。ドメイン外のHTTPS hopではstdlibがコピーした分も含め
AuthorizationとCookieを明示的に削除し、cookie jarも使わない。
資格情報を送らなかった先のHTMLログイン応答は拒否する。Bearerを送ったことは認証成功の証拠ではない。
scliには `files:read` 不足でも同じHTTP 200のログインHTMLが返った記録がある。
元のhost・許可した兄弟hostでも、既知のlogin/error応答と期待するファイルのmetadata（mimetype等）を照合する。
HTMLファイルを一律拒否せず、正規のHTML添付は保持する。認証エラーとの判別がつかない場合は警告し、
空の `local_path` とmetadataを保持して保存先を置き換えない。private download URLはexportに入れない。

入力時刻がマイクロ秒より細かい場合、下限は切り下げ・上限は切り上げる。
マイクロ秒ちょうどの境界は排他のまま維持する。

### 5. Slack通信と部分失敗

`context.Context` をHTTP・再試行待機・streamへ通す。HTTP client、時計・timer、
ファイルシステム境界、I/Oを注入可能にする。コンストラクタは通信しない。
制御APIのtimeoutは30秒、upload/downloadのバイト転送は30分とし、どちらもキャンセル可能にする。
保持する1 GiB上限に小さなJSON用timeoutを適用しないために分ける。
`Retry-After` 待機中もキャンセル可能にする。
明示的なHTTP 429は最大3試行。ただし、後述する一度だけのupload完了処理は除外する。
Retry-Afterが欠けるか構文不正なら1秒とし、
負値やDurationのoverflowは拒否する。
timeout・切断・5xxで結果不明になった変更操作を自動再実行しない。不確定な結果を報告する。
許可された再試行ごとにrequest bodyを再構築する。サイズ制限付きの本文はバッファへ、
upload入力は一度だけprivate一時ファイルへ固定し、許可された再試行では同じ開いたsnapshotをrewindする。
設定上限を守り、すべての終了経路で一時ファイルを削除する。読み切ったbodyを再利用せず、
結果不明なupload手順全体を最初から繰り返さない。

投稿は明示的な `not_in_channel` の後にpublic channelへ一度joinし、投稿を一度だけ再試行する。
uploadは宛先を解決し、割当前に `conversations.info` で参加状況を確認して、必要ならpublicへ一度joinする。
このためuploadではID指定でもchannel-read scopeが必要。参加状況不明・privateへの未参加はエラー。
DM宛先には `conversations.open` を使う。完了処理の再送で参加失敗を回復しない。
参照・exportのためにはjoinせず、private参加条件を迂回せず、実行主体を切り替えない。
作成後のtopic・purpose・招待はtransactionではない。
後段失敗時は作成済みチャンネルIDと失敗段階を報告して非ゼロ終了し、
自動削除や作成からの再実行を行わない。

streamは3秒ごとにバッチ送信し、Slackによる切り捨て前に本文を分割する
（1バッチ4,000 Unicode文字）。待機中バッファを制限し、scannerエラーを伝播させ、
通常EOFでは残りをflushする。最初の送信失敗で非ゼロ終了。
キャンセル時は未送信データが残ることを報告し、送信成功と扱わない。
`--thread`、書式、unfurl指定は全バッチに配線する。JSON/blocks/payloadのstream入力は非対応。

#### ファイル送受信のワナと受入条件

ナレッジのredirect記事とconfig-and-ioのファイル確定に関する知見を適用し、
制御・バイト転送・確定を別の段階として扱う。GCSのCloseの挙動をSlackに仮定せず、
Slackの明示的な完了APIを公式仕様で確認する。

1. upload割当前に宛先・親thread timestamp・入力を検証する。
   通常ファイルを一度だけ開くかstdinから読み、サイズ制限付きコピーでprivate一時ディレクトリ内の
   0600 snapshotへ固定する。その完成snapshotから `length` を取得し、stat後に元のパスを
   開き直さない。1 GiBのupload全体をメモリに読み込まない。
2. filenameと正確なバイト長で `files.getUploadURLExternal` へPOSTし、`ok=true`、
   file ID、HTTPS upload URLを確認する。raw bytesはPUTでなく **POST**、実際のContent-Lengthと
   `application/octet-stream` で送る。署名付きURLをログに出さず、Bearer/Cookieを付けない。
   hostは `slack.com` またはラベル境界付き `.slack.com` に限定し、userinfo・非標準port・
   upload redirectは拒否する。追加hostは検証してから許可し、任意URL対応にしない。
3. バイト転送成功後だけ `files.completeUploadExternal` を呼ぶ。file ID、明示的な
   `channel_id`、任意のcomment、**親**の `thread_ts` を渡す。
   制御APIはHTTP 200でも `ok=true` が必要。バイト転送先の応答は別の非JSON仕様として扱う。
   copy/network失敗後の後始末で完了APIを呼ばない。送信成功でなく、確定成功をupload成功とする。
4. Slackの公式手順では完了処理は一度だけ。429を含む汎用再試行・auto-join後再送から除外し、
   file IDと失敗段階を報告する。完了応答が不確定なら結果不明とし、自動で別ファイルを割り当てない。
   `channel_id` なしの完了や、確定前の成功ID出力を行わない。
5. ファイル共有のbroadcastフラグを提供せず、完了APIへ `reply_broadcast` を送らない。
   過去のworkspace調査では無視されると記録され、現行公式資料にも引数がない。
   求められていない2件目のメッセージで代替しない。
6. downloadはファイル本体とAPIエラー・未認証HTMLを区別し、正規のHTML添付は保持する。
   copy・サイズ超過・close失敗では一時ファイルだけを削除し、既存の保存先を保持する。
   ファイル上限は宣言サイズと実際の受信バイトの両方に適用し、Content-Length欠落・偽装で迂回させない。
   診断はhost・段階を示し、署名付きURLを含めない。

fixtureには、binary往復のバイト一致、通常投稿と親threadへの添付、許可された再送の本文一致、
バイト送信失敗後の確定禁止、宛先不明時の割当禁止、HTTP 200の `ok=false`、
確定timeout/429の再送禁止、兄弟hostへ認証を伴うdownload、外部hostへ認証を送らないredirect、
トークン非送信時とBearer送信済み・`files:read` 拒否時それぞれのHTTP 200ログインHTML、
判別不能なHTML応答、正規HTML添付、中断・サイズ超過を含める。
実Slackでの往復確認は明示的に許可されたリリース前検証とし、mock成功を実配信の証拠にしない。

### 6. APIとスコープ

公式資料の確認日は2026-09-22。利用する操作のscopeだけを要求する。
下表の会話種別ごとのscopeは選択肢であり、すべてを必須にする意味ではない。

| API・用途 | Bot scope |
|-----------|-----------|
| [auth.test](https://docs.slack.dev/reference/methods/auth.test/) | 追加scope不要。外部操作前にbot/workspaceの同一性を検証 |
| [chat.postMessage](https://docs.slack.dev/reference/methods/chat.postMessage/) | `chat:write`。名前・アイコン変更は追加で `chat:write.customize` |
| [files.getUploadURLExternal](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/)、[files.completeUploadExternal](https://docs.slack.dev/reference/methods/files.completeUploadExternal/) | `files:write`。返されたupload URLへのバイト送信にはAPIトークンを転送しない |
| [conversations.open](https://docs.slack.dev/reference/methods/conversations.open/) | ボットとユーザーのDMは `im:write` |
| [conversations.list](https://docs.slack.dev/reference/methods/conversations.list/)、[conversations.info](https://docs.slack.dev/reference/methods/conversations.info/) | `channels:read` / `groups:read`。DM等を対象にする場合だけ `im:read` / `mpim:read` |
| [conversations.history](https://docs.slack.dev/reference/methods/conversations.history/)、[conversations.replies](https://docs.slack.dev/reference/methods/conversations.replies/) | 種別に応じ `channels:history` / `groups:history` / `im:history` / `mpim:history` |
| [users.list](https://docs.slack.dev/reference/methods/users.list/)、[users.info](https://docs.slack.dev/reference/methods/users.info/) | `users:read`（`users.read` ではない） |
| [usergroups.list](https://docs.slack.dev/reference/methods/usergroups.list/)、[usergroups.users.list](https://docs.slack.dev/reference/methods/usergroups.users.list/) | `usergroups:read`。グループ名・所属メンバー展開時だけ |
| [conversations.join](https://docs.slack.dev/reference/methods/conversations.join/) | `channels:join` |
| [conversations.create](https://docs.slack.dev/reference/methods/conversations.create/)、[conversations.setTopic](https://docs.slack.dev/reference/methods/conversations.setTopic/)、[conversations.setPurpose](https://docs.slack.dev/reference/methods/conversations.setPurpose/) | publicは `channels:manage`、privateは `groups:write` |
| [conversations.invite](https://docs.slack.dev/reference/methods/conversations.invite/) | publicは `channels:manage` または `channels:write.invites`。privateは `groups:write` または `groups:write.invites` |
| 認証付きprivate fileダウンロード | `files:read` と対象ファイルへのアクセス権 |

現行replies資料は、DMに加えてchannel用bot scopeも列挙する。過去の
「botはDMの返信しか取得できない」という前提を固定しない。
資料確認と実際の機能確認は別であり、リリース前にインストール済みbotの実権限で確認する。
実行主体の検証や必須メッセージ取得の拒否は、method・code・必要scope付きのエラーにする。
任意の表示名補完は `users:read` 拒否も含め警告とID保持、ファイル保存は `files:read` 拒否も含め
警告とメタデータ保持を許す。これらの警告は `--quiet` でも表示する。
必須のメッセージ取得範囲を黙って縮めない。
内部アプリと商用配布アプリでhistory/repliesのレート制限は異なり得る。
ページ件数から取得終了を判断しない。Slackの投稿名義変更には適切なユーザーの許可が必要で、
フラグを残すことは無人サービスでユーザーになりすましてよいという意味ではない。

### 7. 実装構造と移行

```text
main.go                 終了コードとsignal context
cmd/                    引数、選択済み設定、stdout/stderr、依存注入
internal/config/        profile・server環境変数（サービスregistryなし）
internal/slack/         Slackモデル、通信、名前解決、操作
internal/export/        scli互換exportモデルと表示
internal/input/         ブロックするローカル入力のキャンセル
testdata/export/        合成API入力とscli互換の期待JSON
```

開発用ブランチでprovider境界を段階的に置き換え、履歴と有用なfixture・テストを保持する。
利用側が定義する小さなinterfaceやRoundTripper注入で、実行時mock providerを置き換える。
scli/swrite実行ファイルには依存せず、別リポジトリの `internal` をimportしない。
採用した動作と参照commitを記録し、scat内のfixtureで整合を検証する。
共通ライブラリ化は、将来の複数リポジトリに関する判断で必要性が示された場合に検討する。

| 既存インターフェース | v2への移行 |
|----------------------|------------|
| `export log -c X` | `channel export X` |
| `--start-time` / `--end-time` | `--start` / `--end` |
| `--output-files DIR` | `--save-dir DIR`。`auto` でなく保存先を明示 |
| `--output-format` | export `--format` |
| `--silent`、`--iconemoji`、`--noop` | `--quiet`、`--icon-emoji`、`--dry-run` |
| upload `--filetype` | 削除。現行Slack upload実装でも使用されていない |
| `provider`、`endpoint`、`SCAT_PROVIDER`、`--provider` | 削除。Slack botプロファイルだけを残す |
| 全プロファイルのchannel/user一覧 | workspaceごとに `--profile` を指定して実行 |
| exportのメンション置換・全体時系列順 | 原文・scliのスレッド単位の順序 |

恒久的な互換aliasは作らない。削除フラグは移行方法を示して拒否する。
削除済みフィールドがある旧設定は、バックアップして編集する案内とともにエラーにする。
mockプロファイルを実botと解釈し直したり、読み込み時に資格情報を書き換えたりしない。
空でない `SCAT_PROVIDER` も削除案内付きで拒否する。
他のprofile情報と入力制限は維持できる。既存exportファイルは書き換えない。import機能はない。

### 8. 開発計画と受入条件

1. **設計:** 本記録の独立レビューを行い、コマンド・export期間・撤去・失敗時の仕様を
   承認してから実装する。
2. **仕様fixture:** 通常投稿・bot投稿・空履歴・空files・リッチコンテンツ・親子・
   保存結果についてscli出力を固定する。`local_path` 等の仕様書と実出力の差を記録する。
3. **基盤と設定:** Slack専用config/client、依存注入、必要時の名前解決、キャンセル、
   レート制限、資格情報保護と単体テスト。
4. **操作:** post/upload/stream・channel/user管理とテスト。既存のbot動作を保持し、
   swrite互換の結果出力を追加する。
5. **Export:** スレッド単位の出力、ページング、broadcast重複排除、atomic出力、
   添付保存・失敗テスト。時計・保存先以外の差を隠さずscli互換項目をgolden testする。
6. **移行・リリース:** 日英の利用者向け文書とsetupを同時更新し、代替テストが揃ってから
   旧providerコード・テストを撤去。CONTRIBUTING、AGENTS、CHANGELOG、旧roadmapも更新。
   独立コードレビュー、`go test ./...`、`make check`、`make build`、4環境ビルド、
   脆弱性検査を通す。専用のfixture workspaceでbot実権限を確認する。
   投稿・作成・招待を行う実環境テストには、その利用の明示的な許可が必要。
   その後、組織のrelease・署名・`verify-release`・Homebrew・umbrella・カタログ・
   `check-org.sh` の手順を実施する。公開は実装ブランチとは別のgateとする。

重点テスト: 短い非最終ページを含む複数ページ、cursor再出現・欠落、broadcast重複、
期間外の親、期間外の返信、bot/user ID、メンション原文、権限失敗、保存失敗でも情報保持、
redirectのtoken範囲、パストラバーサル・symlink、中断時の既存出力保持、429上限・キャンセル、
結果不明な変更操作を再送しないこと、stream読み取り・送信失敗、dry-runや不要な名前解決で
通信しないこと、profile分離、旧設定拒否。

## Consequences

サービスは単一のbot実行主体と予測可能なSlack専用CLIを得る。
export利用者はリッチ情報・添付を含むscliのデータモデルを使える。
v2ではスクリプトと設定の移行が必要になる。botとuserの可視範囲の差は残る。
メモリ使用量と親の選択期間という意味は、scli互換exportの明示的な制約として残す。
文書を複製するだけでなく、テストで基準との整合を守る。

## Alternatives considered

- **汎用provider維持:** 未対応サービス用の複雑さを持ち越す。
- **scli丸ごとコピー:** user認証の責務と、仕様を意味しない実装上の問題も持ち込む。
- **パッケージ名だけ変更:** 初期化・部分失敗・データ差が残る。
- **即座に共通ライブラリ化:** 仕様の検証前に、scat刷新を全ツール移行へ拡大する。
- **全返信を期間で絞る:** scliの親選択の意味が変わる。期間限定のhistoryだけでは、
  古い親へのすべての期間内返信は発見できない。
- **全旧フラグ・schema維持:** 破壊的変更を許容する合意にもかかわらず、複雑さをv2へ持ち越す。

## References

- [scliのdownload調査記録](https://github.com/nlink-jp/scli/blob/854e6a0/internal/slack/redirect.go): `files:read` 不足によるHTTP 200のログインHTML。トークン送信だけでは成功と判断できない。
- [組織規約](https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md): プロジェクト単位ADR、実装前設計、独立検証。
- [scli export仕様](https://github.com/nlink-jp/scli/blob/854e6a0/docs/en/EXPORT_FORMAT.md) と [出力型](https://github.com/nlink-jp/scli/blob/854e6a0/internal/slack/types.go): 今回の基準。`local_path` は説明との不一致があるため出力型に合わせる。
- [開発プロセスの知見](https://github.com/nlink-jp/knowledge/blob/main/docs/ja/development-process.md): 動作仕様と有用な期待値を保持し、繰り返す差の発生源を再構成する。
- [設定・I/Oの知見](https://github.com/nlink-jp/knowledge/blob/main/docs/ja/config-and-io.md): 名前変更では保存済み設定を考慮する。今回は別の認証先への暗黙の正規化を避け、明示的な移行エラーを選ぶ。
- [セキュリティの知見](https://github.com/nlink-jp/knowledge/blob/main/docs/ja/security.md): ダウンロードの資格情報転送範囲を維持。
- [API再試行の知見](https://github.com/nlink-jp/knowledge/blob/main/docs/ja/mcp-server-design.md): timeoutした変更操作を安全に繰り返せると仮定しない。
- [ファイル確定の知見](https://github.com/nlink-jp/knowledge/blob/main/docs/ja/config-and-io.md): 後始末と確定を分け、計測と転送の間で入力を安定させる。
- [Slack upload手順](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/) と [完了処理](https://docs.slack.dev/reference/methods/files.completeUploadExternal/): 3段階、バイトのPOST、明示的宛先、親timestamp、一度だけの確定。

workspaceの指示とmemoryの `project_export_format`、`project_scli_bot_mode_paused`、
`reference_go_authorization_across_redirects`、`feedback_slack_api_quirks`、`project_scli_mcp`、
`project_slack_mcp_extender` のファイル転送に関する知見を確認した。
旧stail基準と中断中のscli-bot案より、今回のscli-export基準・scat分離という明示的な決定を優先する。
項目とダウンロードに関する有用な知見は保持する。memoryのローカルパスや私的な値は公開文書に含めない。
実装テストで確認した境界時刻の丸めとファイル応答判定の知見をknowledgeへ還元する。実Slackでの新しい実測を主張しない。
