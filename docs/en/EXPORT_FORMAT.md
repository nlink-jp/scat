# Export data format (v2)

[日本語](../ja/EXPORT_FORMAT.ja.md)

`scat channel export <channel>` follows scli `854e6a0`'s JSON model. The golden
fixtures in `testdata/export/` pin the contract and the deliberate correction of
broadcast duplicates. Authentication is bot-only, so visibility can differ from scli.

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

| Field | Meaning |
|-------|---------|
| `export_timestamp` | UTC RFC3339 export time |
| `channel_name` | `#name`, falling back to `#ID` with a warning |
| `messages` | Array, including `[]` for empty history |
| `user_id` | API `user`; otherwise `bot_id` |
| `user_name` | Bot username or resolved user name; ID fallback on user lookup failure; omitted if empty |
| `post_type` | `bot` when `bot_id` exists, otherwise `user` |
| `timestamp` | UTC RFC3339 (seconds) |
| `timestamp_unix` | Original Slack timestamp string, preserving microseconds |
| `text` | Original API text, including mentions |
| `files` | Always an array |
| `thread_timestamp_unix` | Original `thread_ts`; omitted if absent; a parent can reference itself |
| `is_reply` | `thread_ts` exists and differs from message `ts` |
| `attachments` | Legacy rich attachments, omitted if absent |
| `blocks` | Raw JSON array; explicit empty arrays remain arrays |

Each file contains `id`, `name`, `mimetype`, and **always** `local_path`.
`local_path` is an absolute path after successful saving, and `""` otherwise.
Private download URLs and credentials never enter the export.

Attachments preserve `fallback`, `color`, `pretext`, `title`, `title_link`, `text`,
`fields`, `footer`, `image_url`. Empty optional values are omitted. Each field
contains `title`, `value`, `short`.

## Selection and ordering

`--start` / `--end` are exclusive RFC3339 parent-selection bounds. Offsets and
fractional seconds are accepted. Start must precede end. For Slack's microsecond
granularity, start rounds down and end rounds up only when necessary; exact
microsecond bounds remain exclusive. This avoids dropping messages below a
nanosecond end bound.

History parents appear oldest first, each immediately followed by its replies,
also oldest first. Replies are fetched without the parent-selection bounds.
The result may therefore contain replies outside the interval. New activity under
an old, unselected parent is not exhaustively discovered. A broadcast reply is
exported once: under its selected parent, or as a history item if that parent was
not selected. Pagination follows cursors even on short pages; invalid/repeated
cursors fail rather than silently truncating.

## Failures and saved files

History/reply failures abort before JSON output; existing output files remain
untouched. Names may fall back to IDs with warnings. Failed attachment downloads
retain metadata and empty `local_path`. `--quiet` does not suppress these warnings.

`--save-dir` stores files as `<fileID>_<basename>` using an anchored directory and
atomic replacement. Traversal names and existing symlinks are refused. Size limits
cover both metadata and actual transferred bytes. Interrupted saves remove only
the temporary file. Earlier completed downloads can remain after a later failure.
Credential redirects, login HTML at HTTP 200, legitimate HTML/JSON attachments,
and ambiguous responses are checked separately; see the ADR's file-transfer cases.

The complete export is collected in memory before rendering; memory use grows
with history size. `--output -` writes stdout; a broken pipe returns an error.
`--output <path>` replaces the file atomically after rendering succeeds.
`--format text` displays the same model for human reading; JSON is the full data
interchange format and default.
