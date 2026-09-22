# Exportデータ形式（v2）

[English](../en/EXPORT_FORMAT.md)

`scat channel export <channel>` はscli `854e6a0` のJSONモデルに準拠します。
`testdata/export/` のgolden fixtureで仕様と意図的なbroadcast重複除去を固定します。
認証はbot専用なので、閲覧範囲はscliと異なることがあります。

```json
{
  "export_timestamp": "2026-09-22T00:00:00Z",
  "channel_name": "#general",
  "messages": [{
    "user_id": "U0123456789",
    "user_name": "Example",
    "post_type": "user",
    "timestamp": "2024-01-01T00:00:00Z",
    "timestamp_unix": "1704067200.123456",
    "text": "Original <@U0123456789> text",
    "files": [],
    "is_reply": false
  }]
}
```

| フィールド | 意味 |
|------------|------|
| `export_timestamp` | UTC RFC3339のexport時刻 |
| `channel_name` | `#name`。取得失敗時は警告と `#ID` |
| `messages` | 配列。空履歴も `[]` |
| `user_id` | APIの `user`、なければ `bot_id` |
| `user_name` | botのusernameまたはユーザー表示名。補完失敗時はID、空なら省略 |
| `post_type` | `bot_id` があれば `bot`、なければ `user` |
| `timestamp` | UTC RFC3339（秒単位） |
| `timestamp_unix` | 元のSlack timestamp文字列。マイクロ秒を保持 |
| `text` | メンションを含むAPIの原文 |
| `files` | 常に配列 |
| `thread_timestamp_unix` | 元の `thread_ts`。なければ省略。親は自分自身を参照することがある |
| `is_reply` | `thread_ts` が存在し、自分の `ts` と異なる |
| `attachments` | legacy rich attachment。なければ省略 |
| `blocks` | raw JSON配列。明示的な空配列も配列として保持 |

各fileは `id`、`name`、`mimetype` と、**常に存在する** `local_path` を含みます。
`local_path` は保存成功後の絶対パス、それ以外は `""` です。
private download URLと資格情報はexportに含めません。

attachmentsは `fallback`、`color`、`pretext`、`title`、`title_link`、`text`、
`fields`、`footer`、`image_url` を保持します。空の任意項目は省略します。
各fieldは `title`、`value`、`short` を含みます。

## 選択と順序

`--start` / `--end` は親を選ぶ排他的なRFC3339境界です。offset・小数秒も指定できます。
startはendより前である必要があります。Slackのマイクロ秒単位に合わせ、必要な場合だけ
startを切り下げ、endを切り上げます。マイクロ秒ちょうどの境界はそのまま排他的に扱います。
これによりナノ秒単位のend未満のメッセージを落としません。

historyの親を古い順に並べ、その直後に返信を古い順に置きます。
返信には親選択の期間を適用しないため、期間外の返信も含まれます。
選択されない古い親の新しい返信を網羅する機能ではありません。
broadcast返信は選択された親の下、または親が選択されなければhistoryの項目として一度だけ出力します。
短いページでもcursorを追い、不正・循環cursorは黙って打ち切らずエラーにします。

## 失敗と保存ファイル

history/repliesの失敗はJSON出力前にexport全体を中止し、既存の出力ファイルを保持します。
名前補完は警告とIDへのfallbackを許します。添付保存失敗ではmetadataと空の `local_path` を保持します。
`--quiet` でもこれらの警告は消しません。

`--save-dir` は保存先ディレクトリを固定し、`<fileID>_<basename>` にatomicに保存します。
パストラバーサル名と既存symlinkは拒否します。サイズ上限はmetadataと実受信バイトの両方に適用します。
中断時は一時ファイルだけを削除します。後段の失敗前に保存した添付は残ることがあります。
認証redirect、HTTP 200のログインHTML、正当なHTML/JSON添付、判別不能な応答を別々に検査します。
詳細はADRのファイル送受信の受入条件を参照してください。

export全体をメモリに集めてから表示するため、メモリ使用量は履歴量に比例します。
`--output -` はstdoutへ書き、broken pipeはエラーです。
`--output <path>` は表示データの書込成功後にatomicに置き換えます。
`--format text` は同じモデルの人向け表示で、完全なデータ交換形式と既定値はJSONです。
