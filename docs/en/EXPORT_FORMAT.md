# Export Log Data Format

The `scat export log` command outputs channel message history in a structured JSON format. Each entry in the `messages` array represents a single message and contains the following fields:

- `user_id` (string): The ID of the user or bot who posted the message.
  - For messages from human users, this is the Slack User ID (e.g., `U12345ABC`).
  - For messages from bots, this is the Slack Bot ID (e.g., `B012345DEF`).
- `user_name` (string, optional): The display name of the user or bot.
  - For human users, this is typically their display name or real name.
  - For bot messages, this is usually the bot's configured username.
- `post_type` (string): Indicates the type of poster.
  - `"user"`: The message was posted by a human user.
  - `"bot"`: The message was posted by a bot.
- `timestamp` (string): The message's timestamp in RFC3339 format (e.g., `2025-08-15T10:30:00Z`).
- `timestamp_unix` (string): The message's timestamp in Unix epoch format (e.g., `1755255897.650199`). This is the raw timestamp provided by Slack.
- `text` (string): The content of the message.
- `files` (array of objects, optional): An array of file objects if the message includes attachments.
  - `id` (string): The ID of the file.
  - `name` (string): The original name of the file.
  - `mimetype` (string): The MIME type of the file (e.g., `image/jpeg`, `text/plain`).
  - `local_path` (string, optional): The local path where the file was downloaded, if `--output-files` was specified during export.
- `thread_timestamp_unix` (string, optional): If the message is a reply, this is the Unix timestamp of the parent message in the thread.
- `is_reply` (bool): `true` if the message is a reply within a thread, otherwise `false`.

## Example JSON Output

This example includes a regular message, a message with a file, a message that starts a thread, and a reply to that thread.

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
      "files": [],
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
      "files": [],
      "is_reply": false
    },
    {
      "user_id": "U12345ABC",
      "user_name": "John Doe",
      "post_type": "user",
      "timestamp": "2025-08-14T10:12:00Z",
      "timestamp_unix": "1755168720.000000",
      "text": "This is a reply to Jane's message.",
      "files": [],
      "thread_timestamp_unix": "1755168600.000000",
      "is_reply": true
    }
  ]
}
```
