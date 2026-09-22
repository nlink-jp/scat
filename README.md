# scat

[日本語](README.ja.md)

A Slack CLI for services using **bot credentials**. `scli` is the sibling CLI for
people using user credentials. scat retains multiple workspace profiles while
removing the unused multi-service provider layer.

See [ADR-0001](docs/en/adr/0001-slack-bot-renewal.md) for the renewal contract
and [migration](#migration-from-v1) for breaking interface changes.

## Setup

Requires Go 1.25+ to build. Download published versions from
[Releases](https://github.com/nlink-jp/scat/releases), or build this checkout:

```sh
make build                  # dist/scat
make check                  # go vet ./... + go test ./...
scat config init
scat profile set token      # hidden terminal prompt; never pass a token argument
scat profile set channel C0123456789
```

[Slack setup and scopes](docs/en/SLACK_SETUP.md) · [Build and test](docs/en/BUILD.md)

Run `make e2e` against a dedicated Slack channel before declaring live behavior verified.
It builds the CLI and checks real posts, threads, streams, file round trips and export.
See [live E2E setup](docs/en/BUILD.md#live-slack-e2e); missing configuration fails the run.

Configuration is `~/.config/scat/config.json`. New directories use 0700 and
credential files use 0600. Existing broad file permissions produce a warning.
The default profile starts without credentials; it cannot silently post anywhere.
`auth.test` verifies bot and workspace identity before remote work. User/app-level
tokens are rejected; scat never reads scli credentials or `SLACK_TOKEN`.

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

`profile set` accepts `channel`, `username`, `token`, and the two limit keys above.
`--profile` also selects the target of `profile set`. Default/active profiles cannot
be removed. Limits must be nonnegative; **0 remains unlimited after save/load**.
Lookups are cached only within an invocation; `cache clear` reports that no
persistent cache exists. Configuration commands reject `--json`.

## Service mode

Set `SCAT_MODE=server` and inject `SCAT_TOKEN` through your service's secret
mechanism. Optional variables: `SCAT_CHANNEL`, `SCAT_USERNAME`,
`SCAT_MAX_FILE_SIZE`, `SCAT_MAX_STDIN_SIZE`. Defaults are 1 GiB for files and
10 MiB for stdin, in bytes. `SCAT_CACHE_DIR` is reserved for optional persistent
caching; this implementation does not write a persistent cache.

Server mode reads no profile file, never prompts, and rejects `--config`,
`--profile`, and local configuration commands. Invalid modes and legacy
`SCAT_PROVIDER` values fail explicitly.

## Commands

Global flags: `--config`, `--profile/-p`, `--quiet/-q`, `--debug`, `--json`,
`--version/-V`. Quiet suppresses informational stderr, not results, warnings or
errors. Debug prints configuration-selection diagnostics without tokens or bodies.

```sh
scat post 'Hello' -c '#general'
printf 'Hello\n' | scat post
scat post --from-file message.txt --user U0123456789
scat post 'Reply' -c C0123456789 --thread 1704067200.123456
scat post --format blocks '[{"type":"divider"}]'
scat post --format payload --from-file message.json
scat post 'Preview' --dry-run
```

Input precedence is argument, `--from-file`, stdin. `--format` accepts `text`,
`blocks` (an array or an object containing `blocks`), and `payload`. Payload fields
are `text`, `blocks`, `attachments`, `unfurl_links`, `unfurl_media`, `mrkdwn`;
routing and identity come from CLI/config, not the JSON. Optional flags:
`--username`, `--icon-emoji`, `--thread`, `--unfurl-links`, `--unfurl-media`,
`--mrkdwn`. Explicit CLI booleans override payload booleans. Custom name/icon
requires the appropriate Slack permission and authorization.

Post prints the timestamp; `--json` prints `{"ts":"...","channel":"..."}`.
`--user` opens a DM and cannot be combined with explicit `--channel`.

```sh
tail -f service.log | scat post --stream -c C0123456789
printf 'pipeline text\n' | scat post --tee
```

Stream batches every three seconds or 4,000 Unicode characters, flushes at EOF,
and fails on input/delivery errors or cancellation. The stdin limit applies to
the invocation, including streams. Thread and formatting flags apply to every
batch. `--tee` copies text stdin to stdout and suppresses result IDs; it rejects
`--json`, argument/file input and structured formats. Stream accepts only text
stdin and rejects `--dry-run`.

```sh
scat upload --file report.pdf -c C0123456789 --comment 'Report'
cat report.pdf | scat upload --file - --filename report.pdf --user U0123456789
scat upload --file report.pdf -c C0123456789 --thread 1704067200.123456
```

Upload also accepts `--dry-run`; stdin requires `--filename`. File inputs must be
regular files. Inputs are staged in a private temporary snapshot with bounded
memory. Upload checks destination membership before allocation, transfers bytes,
and completes sharing with an explicit channel and optional **parent** timestamp.
Only successful completion prints file IDs, or
`{"files":[{"id":"..."}],"channel":"..."}` with `--json`.
Completion is never automatically replayed, even on 429; an uncertain result
reports file ID and stage. File-share broadcast is not supported.

```sh
scat channel list --json
scat user list --json
scat channel create alerts --topic 'Service alerts' --description 'Automation' --invite U0123456789
scat channel invite C0123456789 U0123456789 @on-call
```

Lists concern only the selected profile and return JSON arrays. IDs avoid name
listing; ambiguous names fail. Invitations accept users or user groups; group IDs
avoid user-name lookup. Creation prints its ID or `{"id":"...","name":"..."}`.
Invitation JSON is `{"channel":"...","users":["..."]}`; plain mode uses stderr.
`channel invite` adds users to an existing channel, independently of creation-time
`channel create --invite`.

Create/invite support `--dry-run` and do not resolve names in that mode.
If setting topic/purpose or inviting fails after creation, the error includes the
created ID; scat does not recreate or delete it automatically.

## Export

```sh
scat channel export C0123456789 --output messages.json --save-dir attachments
scat channel export '#general' --start 2024-01-01T00:00:00Z --end 2024-02-01T00:00:00Z
scat channel export C0123456789 --format text
```

Export follows [scli's data model](docs/en/EXPORT_FORMAT.md), including rich
attachments, raw blocks/text and file metadata. JSON is the default; `--output -`
uses stdout. `--format text` renders the same retrieved data and rejects `--json`.

**Time bounds select parents, exclusively; each selected thread is then fetched
in full.** Replies outside the interval can appear. This does not exhaustively
find new replies to old, unselected parents. Output is parent followed by its
replies, not global time order. Broadcast duplicates are removed.

Message/thread retrieval errors fail the whole export before output. Name lookup
or file-download failure warns and preserves IDs/metadata, with `local_path:""`
for unsaved files. Warnings remain visible under `--quiet`. Output files and
attachments are replaced only after successful writes. Earlier successful file
downloads may remain after a later export failure. Export memory grows with the
retrieved history, matching scli.

Downloads protect token-bearing redirects, distinguish login HTML at HTTP 200
from attachments, and retain legitimate HTML/JSON files using response metadata.
Unclear responses fail the file save with a warning. Attachment names cannot escape
the selected directory; existing symlink targets are refused. API calls use a
30-second timeout, file transfers 30 minutes, with cancellation. HTTP 429 retries
are bounded; ambiguous mutations and upload completion are not replayed.

## Migration from v1

Back up your config and scripts before upgrading. Remove `provider` and `endpoint`
from **every** profile, and unset `SCAT_PROVIDER`. Old fields fail with migration
guidance; scat never converts a mock profile into a live bot silently. Supply bot
credentials through the prompt or service environment. Existing exports are untouched.

| v1 | v2 |
|----|----|
| `export log --channel X` | `channel export X` |
| `--start-time`, `--end-time` | `--start`, `--end` |
| `--output-files DIR` | `--save-dir DIR` (explicit directory; no auto mode) |
| `--output-format` | export `--format` |
| `--silent`, `--iconemoji`, `--noop` | `--quiet`, `--icon-emoji`, `--dry-run` |
| `--provider`, upload `--filetype` | Removed |
| List every profile at once | Select each with `--profile` |
| Resolved mention text / global export sort | Original text / thread grouping |

There are no permanent compatibility aliases or runtime mock providers. Tests
inject HTTP and I/O instead. Dry runs validate local inputs, emit a redacted stderr
summary and no success IDs; they do not verify bot identity or remote permissions.
