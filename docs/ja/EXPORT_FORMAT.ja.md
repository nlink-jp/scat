# ログエクスポートのデータ形式

`scat export log` コマンドは、チャンネルのメッセージ履歴を構造化されたJSON形式で出力します。

## 最上位

- `export_timestamp` (string): エクスポートした時刻（UTC、RFC3339形式、例: `2025-08-15T11:03:53Z`）。
- `channel_name` (string): `--channel` に渡したチャンネル名そのもの（例: `#example-channel`）。
- `messages` (array of objects): メッセージの配列（1件につき1エントリ）。

## メッセージ

`messages` 配列の各エントリは1件のメッセージを表し、以下のフィールドを含みます。

- `user_id` (string): メッセージを投稿したユーザーまたはボットのID。
  - 人間のユーザーによるメッセージの場合、これはSlackのユーザーID（例: `U12345ABC`）です。
  - ボットによるメッセージの場合、これはSlackのボットID（例: `B012345DEF`）です。
- `user_name` (string, optional): ユーザーまたはボットの表示名。
  - 人間のユーザーの場合、通常は表示名または本名です。
  - ボットのメッセージの場合、通常はボットに設定されたユーザー名です。
- `post_type` (string): 投稿者の種別を示します。
  - `"user"`: メッセージは人間のユーザーによって投稿されました。
  - `"bot"`: メッセージはボットによって投稿されました。
- `timestamp` (string): メッセージのタイムスタンプ（RFC3339形式、例: `2025-08-15T10:30:00Z`）。
- `timestamp_unix` (string): メッセージのタイムスタンプ（Unixエポック形式、例: `1755255897.650199`）。これはSlackから提供される生のタイムスタンプです。
- `text` (string): メッセージの本文。
- `files` (array of objects, optional): メッセージに添付されたファイルの配列。キーはファイルがあるときだけ現れ、空の配列になることはありません。
  - `id` (string): ファイルのID。
  - `name` (string): ファイルの元の名前。
  - `mimetype` (string): ファイルのMIMEタイプ（例: `image/jpeg`、`text/plain`）。
  - `local_path` (string, optional): エクスポート時に `--output-files` が指定された場合、ファイルがダウンロードされたローカルパス。
- `thread_timestamp_unix` (string, optional): メッセージがスレッドに属するときに現れます。返信ではスレッドの親メッセージのUnixタイムスタンプ、親メッセージ自身では自分の `timestamp_unix` と同じ値です。
- `is_reply` (bool): メッセージがスレッド内の返信である場合は `true`、そうでない場合は `false`（スレッドの親は `false`）。

stail と scli が出力する `attachments`（旧形式のリッチ添付）と `blocks`（Block Kit の JSON）は、scat は出力しません。

## JSON出力の例

この例には、通常のメッセージ、ファイルを含むメッセージ、スレッドを開始するメッセージ、およびそのスレッドへの返信が含まれています。

```json
{
  "export_timestamp": "2025-08-15T11:03:53Z",
  "channel_name": "#example-channel",
  "messages": [
    {
      "user_id": "U12345ABC",
      "user_name": "John Doe",
      "post_type": "user",
      "timestamp": "2025-08-14T10:00:00Z",
      "timestamp_unix": "1755168000.000000",
      "text": "Hello, world!",
      "is_reply": false
    },
    {
      "user_id": "B012345DEF",
      "user_name": "MyBot",
      "post_type": "bot",
      "timestamp": "2025-08-14T10:05:00Z",
      "timestamp_unix": "1755168300.000000",
      "text": "This is a bot message.",
      "files": [
        {
          "id": "F98765XYZ",
          "name": "report.pdf",
          "mimetype": "application/pdf",
          "local_path": "./scat-export-example-channel-20250815T110353Z/F98765XYZ_report.pdf"
        }
      ],
      "is_reply": false
    },
    {
      "user_id": "U67890GHI",
      "user_name": "Jane Smith",
      "post_type": "user",
      "timestamp": "2025-08-14T10:10:00Z",
      "timestamp_unix": "1755168600.000000",
      "text": "Let's start a thread here. This is the parent message.",
      "thread_timestamp_unix": "1755168600.000000",
      "is_reply": false
    },
    {
      "user_id": "U12345ABC",
      "user_name": "John Doe",
      "post_type": "user",
      "timestamp": "2025-08-14T10:12:00Z",
      "timestamp_unix": "1755168720.000000",
      "text": "This is a reply to Jane's message.",
      "thread_timestamp_unix": "1755168600.000000",
      "is_reply": true
    }
  ]
}
```
