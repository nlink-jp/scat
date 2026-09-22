# ADR-0001: Renew scat as a Slack CLI for services

| Field | Value |
|-------|-------|
| Status | **Accepted** — maintainer approved implementation on 2026-09-22 |
| Date | 2026-09-22 |
| Binds | scat |
| Decision makers | nlink-jp maintainers |
| Triggered by | Unused provider abstraction and observable differences from the sibling Slack tools |

Japanese: [日本語](../../ja/adr/0001-slack-bot-renewal.ja.md).

## Context

scli is used by people with user permissions. scat is used by services with
bot permissions. The renewal keeps that distinction and makes scat Slack-only.
The maintainer has agreed to prioritize consistent behavior over backwards
compatibility. This record was approved before implementation. The implementation follows
the submodule main workflow; publication and live Slack verification are separate gates.

The comparison baseline is scat `24cfd70`, scli `854e6a0`, swrite `c47d9e3`, and
stail `1323ef2`. scli's export is the reference because it covers threads, rich
attachments, Block Kit, and downloaded files. swrite is the reference for bot
posting and pipeline use. Neither reference is copied blindly.

Current scat registers only Slack plus two testing providers. Its capability
flags, deprecated endpoint field, provider selection, and provider-dependent
commands therefore add complexity without a second service. Construction also
fetches channels, users, and user groups before knowing which are needed.
Export loses rich content and can omit a whole thread or a file's metadata on
partial failure. These are structural reasons for replacing the affected
boundaries, not for discarding every existing tested behavior.

## Decision

### 1. Product boundary

- Keep the `scat` name, repository, and chatops-series membership. Target the
  breaking release as **v2.0.0**, since the current release is already 1.x.
- Keep posting, DM delivery, uploads, stdin streaming, export, channel/user
  listing, channel creation, user/group invitations, and named bot profiles.
- Remove provider registration, capability negotiation, `endpoint`,
  `SCAT_PROVIDER`, `--provider`, and public `mock`/`test` modes.
- Multiple bot profiles remain useful; multiple services do not.
- Accept bot credentials only. Reject user/app-level tokens with a diagnostic;
  never borrow scli credentials or fall back to another token. Do not introduce
  user OAuth, search/unread state, Socket Mode, Discord, or Teams in this renewal.
  Before the first remote operation in each invocation, call `auth.test` and
  require a nonempty `bot_id` and workspace identity; token prefix alone is not
  proof. Cache that result only within the invocation. Local configuration,
  help/version, and dry-run do not call it; dry-run does not claim authenticated
  identity validation.
- Keep stail and swrite independent. Their retirement or a fleet-wide shared
  library would be a separate decision; this ADR binds only scat.

### 2. Command and output contract

| Operation | Renewed interface |
|-----------|-------------------|
| Post | `scat post [text] -c <channel>` or `--user <user>`; argument, `--from-file`, stdin precedence |
| Rich post | `--format text|blocks|payload`, following swrite; payload preserves text, blocks, attachments and unfurl settings |
| Thread post | `post --thread <ts>`; explicit Slack timestamp string |
| Upload | `scat upload --file <path|-> -c <channel>`; `--filename`, `--comment`, `--thread`; DM alternative via `--user` |
| Export | `scat channel export <channel> --output <path|-> --start <RFC3339> --end <RFC3339> --save-dir <dir>` |
| Lists | `scat channel list`, `scat user list`, optional `--json`; selected profile only |
| Management | `channel create <name>` with `--private`, `--topic`, `--description`, `--invite`; `channel invite <channel> <user-or-group>...` |
| Local configuration | `config init`; `profile add/list/use/set/remove`; `cache clear` |

Global flags: `--config`, `--profile/-p`, `--quiet/-q`, `--debug`, `--json`, and
`--version/-V`. `--quiet` suppresses informational stderr only, never data,
warnings, or errors. Diagnostics never contain credentials or full request bodies.

`post` prints the resulting timestamp by default, as swrite does; `--json`
prints `{"ts":"...","channel":"..."}`, as scli does. Streaming prints one
result per successful batch. `--tee` deliberately makes stdout a copy of stdin:
it suppresses result output and is incompatible with `--json`. Upload prints
file IDs (one per line), or `{"files":[{"id":"..."}],"channel":"..."}` with
`--json`. Channel creation prints its ID or `{"id":"...","name":"..."}`;
invitations have diagnostics only, or `{"channel":"...","users":["..."]}`
after success with `--json`. List JSON is an array, never a profile-keyed map.
Configuration commands retain their human-facing output; reject `--json` there
until an explicit configuration-output contract exists.

Export defaults to JSON independently of `--json`. Retain the useful text
renderer via export `--format text` (reject with `--json`); it is a view of the
same export model, not another retrieval path. Empty JSON collections are `[]`.

`--dry-run` replaces `--noop` for post/upload/create/invite: validate local input
and print a redacted operation summary to stderr, without API calls, writes to
Slack, or success IDs. Reject it for reads and `--stream`. It cannot confirm
remote permissions or resolve names. No runtime testing provider is required.

### 3. Bot configuration and service execution

Keep `~/.config/scat/config.json` with `current_profile` and `profiles`. Profile
fields are `token`, `channel`, `username`, and `limits`. Write directories with
0700 and files with 0600; warn about insecure existing credential permissions.
Do not put tokens in command arguments. Secure prompting is confined to explicit
profile setup with a terminal; service commands never prompt.

`SCAT_MODE=server` reads only `SCAT_TOKEN`, `SCAT_CHANNEL`, `SCAT_USERNAME`,
`SCAT_MAX_FILE_SIZE`, `SCAT_MAX_STDIN_SIZE`, and optional `SCAT_CACHE_DIR`.
It does not read or change profiles/keychains. Reject `--config`, `--profile`,
and configuration-management commands in that mode. Invalid modes fail.
Preserve the existing 1 GiB file and 10 MiB stdin defaults and the explicit
zero-means-unlimited option. Validate size values before reading input.

Select configuration once per invocation and inject it, rather than storing it
in package globals or obtaining it through unchecked context type assertions.
Channel/user/group resolution is lazy; explicit IDs avoid listing APIs. Name
ambiguity fails rather than selecting the first match. Persistent caches are
optional in server mode, scoped to the bot/workspace identity and invalidated
when credentials change; no tokens are stored in cache content or path names.

### 4. Export: scli is the reference

Use the exact scli envelope: `export_timestamp`, `channel_name`, `messages`.
Do not introduce a new envelope/schema version as part of this renewal.

| Field | Contract |
|-------|----------|
| `export_timestamp` | UTC RFC3339; injected clock for tests |
| `channel_name` | Resolved name with `#`; if lookup fails, `#<channel ID>` and a warning, matching scli's fallback |
| `user_id` | API `user` when present, otherwise `bot_id` |
| `user_name` | Resolved display name; omitted if empty |
| `post_type` | `bot` when `bot_id` is present, otherwise `user` |
| `timestamp` / `timestamp_unix` | RFC3339 UTC / unmodified Slack timestamp string; never float64 identifiers |
| `text` | Original API text; do not rewrite mentions into display names |
| `files` | Always an array, including `[]`; each file has `id`, `name`, `mimetype`, `local_path` |
| `local_path` | Always a string: absolute path after a successful save, `""` otherwise; follows scli's actual output type |
| `attachments` | scli's documented legacy attachment fields; omitted when absent |
| `blocks` | Raw JSON array; omitted when absent |
| `thread_timestamp_unix` | API `thread_ts`, omitted when absent; parent may equal its own timestamp |
| `is_reply` | `thread_ts` exists and differs from `ts` |

Keep parent messages in chronological order; place each parent's replies
immediately after it in chronological order. This is **thread-grouped**, not a
global timestamp sort. Deduplicate on `(channel ID, ts)`, including broadcast
replies seen in both history and replies; attach them to their selected parent.
Preserve a history-returned broadcast reply once if its parent was not selected.

`--start` and `--end` select history messages using exclusive bounds, then expand
the selected parents' entire threads, following scli. Therefore replies outside
the interval can appear. Replies to an older unselected parent are not exhaustively
discovered. This is a **parent-selection interval**, not an all-message activity
window; do not advertise the latter. Accept RFC3339 with offsets/fractions,
preserve Slack microsecond precision in API bounds, and reject start >= end.
When input has finer precision, floor the lower bound and ceil the upper bound;
exact microsecond bounds remain exclusive.

Follow cursors even when a page is shorter than the requested limit. Use
`has_more` plus the cursor consistently, detect a repeated cursor, and fail on
`has_more=true` without a usable cursor instead of silently completing or looping.
History/thread retrieval errors fail the export. Resolve-name failures warn and
retain IDs/text. Download failures warn and retain file metadata with empty
`local_path`; they do not invalidate a complete message export.

As in scli, construct a complete export before emitting it. Write output files
through a temporary file in the same directory and rename after success; retrieval
failure leaves an existing output untouched and emits no JSON to stdout. A broken
stdout pipe returns an error; stdout bytes already written cannot be rolled back.
Successful downloads may remain if a later stage fails; never delete the user's
directory as rollback. Whole-export memory use remains proportional to history,
as in scli; a bounded-memory export mode is outside this renewal.

Download into `<fileID>_<safe basename>` beneath the selected directory. Reject
unsafe components, refuse existing symlink destinations, and anchor writes to an
opened directory (`os.Root`) so parent links cannot escape it. Use temporary files
plus atomic replacement so interrupted downloads do not become completed files.
Accept an initial authenticated download URL only over HTTPS, without userinfo or
nonstandard ports, at `slack.com` or a hostname ending in `.slack.com` (label
boundary required). An unrecognized initial host fails without sending a token;
add a new host only after verifying Slack ownership and a real download need.
Preserve scli's redirect protection: judge every hop against the original URL,
not the preceding hop. Never downgrade HTTPS; abort such redirects. On an
outside-domain HTTPS hop explicitly remove Authorization and Cookie, including
headers the standard client might have copied, and use no cookie jar. Reject an
HTML login response reached with the credential withheld. Sending Bearer is not
proof of authentication: scli recorded the same HTTP 200 login page when
`files:read` was missing. Check known login/error responses against expected file
metadata (including mimetype), even on the original or an allowed sibling host.
Do not reject every HTML file; preserve legitimate HTML attachments. If the
response cannot be distinguished from an authentication error, warn, preserve
metadata with empty `local_path`, and do not replace the destination.
Private download URLs never enter the export.

### 5. Slack transport and partial failure

Use `context.Context` throughout; cancellation reaches HTTP, retry waits and the
stream loop. Inject the HTTP client, clock/timer, filesystem boundary and I/O.
The constructor does no network I/O. Use 30 seconds for control API requests and
30 minutes for upload/download byte transfers, both bounded by caller cancellation;
the latter avoids applying a small-JSON timeout to the retained 1 GiB file limit.
Honor cancellation during `Retry-After`. Allow at most three attempts for an
explicit HTTP 429, except the one-shot upload completion described below;
absent/malformed Retry-After uses one second. Reject negative
or overflowing durations. Do not automatically repeat a mutation after a timeout,
connection loss, or 5xx with an uncertain outcome; report the uncertainty.
Every permitted retry reconstructs its request body. Buffer bounded text input;
stage uploads once in a private temporary file and rewind the same opened
snapshot for an allowed retry, enforcing the size limit and cleaning up on every exit. Never
reuse an exhausted request body or restart an uncertain whole upload sequence.

For posting, join a public channel once after explicit `not_in_channel`, then retry
the post once. For uploads, resolve the destination and check membership with
`conversations.info` before allocation; join a public channel once at this stage
if needed. This requires the channel-read scope even with an ID. Unknown membership
or private nonmembership fails. DM destinations use `conversations.open`.
Do not replay upload completion to recover membership. Never join to read/export,
bypass private membership, or switch identities on an error.
Creation followed by topic/purpose/invites is not transactional: on later failure
report the created channel ID and failed step, return nonzero, and do not delete
the channel or repeat creation automatically.

Stream batches every three seconds and splits text before Slack can truncate it
(4,000 Unicode characters per batch); bound the pending batch, propagate scanner
errors, flush on normal EOF, and stop on the first failed delivery with nonzero
exit. Cancellation reports unsent buffered data instead of claiming successful
delivery. `--thread`, formatting, and unfurl settings must be wired into every
batch; JSON/blocks/payload streaming remains unsupported.

#### File-transfer traps and acceptance cases

Apply the knowledge-base redirect entry and the file-finalization lesson in
config-and-io to separate control, byte-transfer, and finalization stages.
GCS's Close behavior is not assumed to be Slack's behavior; verify Slack's
explicit completion endpoint against its own specification.

1. Validate destination, parent-thread timestamp, and source before allocation.
   Open a regular-file source once, or read stdin, and stage it with bounded copying
   into a 0600 snapshot in a private temporary directory. Derive `length` from that
   snapshot, not a path reopened after stat. Never buffer a 1 GiB upload in memory.
2. POST `files.getUploadURLExternal` with filename and exact byte length; require
   `ok=true`, file ID, and HTTPS upload URL. Send raw bytes by **POST**, not PUT,
   with actual Content-Length and `application/octet-stream`. Never log the signed
   URL or attach Bearer/cookies to this transfer. Accept only `slack.com` or
   label-boundary `.slack.com` hosts, without userinfo/nonstandard port; refuse
   upload redirects. Additional hosts require verification, not arbitrary-URL support.
3. Only after successful byte transfer call `files.completeUploadExternal` with
   file ID, explicit `channel_id`, optional comment, and the **parent** `thread_ts`.
   API control responses require `ok=true` even at HTTP 200; the byte endpoint has
   a separate non-JSON response contract. Never finalize from cleanup after a
   copy/network error. Success means finalization succeeded, not just bytes sent.
4. Completion is one-shot in Slack's documented workflow. Exclude it from generic
   retry and auto-join replay, including 429 replay: report file ID/stage and fail.
   On ambiguous completion, report outcome unknown and never automatically allocate
   another file. Never complete without `channel_id` or print a success ID early.
5. Do not offer file-share broadcasting or send `reply_broadcast` to completion.
   Prior workspace investigation recorded that it is ignored; the current reference
   has no such argument. Do not emulate it with an unsolicited second message.
6. Downloads distinguish file bytes from API errors and unauthenticated HTML,
   while retaining legitimate HTML attachments. On copy, size-limit, or close
   failure remove only the temporary file, preserving an existing destination.
   Apply the file limit to declared size and actual copied bytes; missing or lying
   Content-Length cannot bypass it. Diagnostics identify host/stage, not signed URLs.

Fixtures cover binary upload/download byte equality, root versus parent-thread
attachments, identical bodies on allowed retries, failed byte transfer never
finalized, missing destination never allocating, completion `ok=false` at HTTP 200,
completion timeout/429 without replay, authenticated sibling-host downloads,
foreign-host redirects without credentials, HTML login at 200 both with a withheld
token and with Bearer sent but `files:read` denied, ambiguous HTML responses,
real HTML files, and interrupted/oversize transfers. Live round-trip validation is an explicitly
authorized release gate; mock success is not proof of actual Slack delivery.

### 6. API and scope inventory

Official references checked 2026-09-22. Request only scopes for enabled operations;
the following rows are alternatives by conversation type, not one mandatory union.
(For the distributed manifest, see the 2026-09-23 amendment at the end of this record.)

| API / purpose | Bot scope |
|---------------|-----------|
| [auth.test](https://docs.slack.dev/reference/methods/auth.test/) | No additional scope; verifies bot/workspace identity before remote work |
| [chat.postMessage](https://docs.slack.dev/reference/methods/chat.postMessage/) | `chat:write`; name/icon customization additionally `chat:write.customize` |
| [files.getUploadURLExternal](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/), [files.completeUploadExternal](https://docs.slack.dev/reference/methods/files.completeUploadExternal/) | `files:write`; upload bytes to returned URL without forwarding the API token |
| [conversations.open](https://docs.slack.dev/reference/methods/conversations.open/) | `im:write` for a bot-user DM |
| [conversations.list](https://docs.slack.dev/reference/methods/conversations.list/), [conversations.info](https://docs.slack.dev/reference/methods/conversations.info/) | `channels:read` / `groups:read`; `im:read` / `mpim:read` only when addressing those conversation types |
| [conversations.history](https://docs.slack.dev/reference/methods/conversations.history/), [conversations.replies](https://docs.slack.dev/reference/methods/conversations.replies/) | `channels:history` / `groups:history` / `im:history` / `mpim:history` by conversation type |
| [users.list](https://docs.slack.dev/reference/methods/users.list/), [users.info](https://docs.slack.dev/reference/methods/users.info/) | `users:read` (not `users.read`) |
| [usergroups.list](https://docs.slack.dev/reference/methods/usergroups.list/), [usergroups.users.list](https://docs.slack.dev/reference/methods/usergroups.users.list/) | `usergroups:read`; only for group-name/member expansion |
| [conversations.join](https://docs.slack.dev/reference/methods/conversations.join/) | `channels:join` |
| [conversations.create](https://docs.slack.dev/reference/methods/conversations.create/), [conversations.setTopic](https://docs.slack.dev/reference/methods/conversations.setTopic/), [conversations.setPurpose](https://docs.slack.dev/reference/methods/conversations.setPurpose/) | `channels:manage` for public / `groups:write` for private |
| [conversations.invite](https://docs.slack.dev/reference/methods/conversations.invite/) | `channels:manage` or `channels:write.invites` for public; `groups:write` or `groups:write.invites` for private |
| Authenticated private-file download | `files:read`, plus access to the file |

The current replies reference lists bot scopes for channels as well as DMs;
do not hardcode the historical assumption that bots can only fetch DM replies.
Documentation is not a live capability measurement: verify the installed bot's
actual access before release. Refusal of identity validation or required message
retrieval fails the export with method/code/needed scopes. Optional user-name
enrichment may warn and retain IDs, including on `users:read` denial; file saves
may warn and retain metadata, including on `files:read` denial. These warnings
remain visible with `--quiet`. Required message retrieval never silently falls
back to fewer messages. Internal apps and commercially distributed apps can
have different history/replies rate limits; never infer completion from page size.
Slack posting customization requires appropriate user authorization; retaining the
flag is not permission to impersonate a user in an unattended service.

### 7. Implementation structure and migration

```text
main.go                 process exit and signal context
cmd/                    parsing, selected config, stdout/stderr, injected dependencies
internal/config/        profiles and server environment (no service registry)
internal/slack/         Slack models, transport, resolution, operations
internal/export/        scli-compatible export model and rendering
internal/input/         cancellation of blocking local input
testdata/export/        synthetic API input and expected scli-compatible JSON
```

Replace the provider boundary incrementally on main in the existing submodule,
preserving history and useful fixtures/tests. Small consumer-owned interfaces
or injected RoundTrippers replace runtime mock providers. No scli/swrite executable
dependency and no import of another repository's `internal` packages. Record
reference commits with adopted behavior and keep parity fixtures in scat; a shared
library is considered only if a later multi-repository decision justifies it.

| Existing interface | v2 migration |
|--------------------|--------------|
| `export log -c X` | `channel export X` |
| `--start-time` / `--end-time` | `--start` / `--end` |
| `--output-files DIR` | `--save-dir DIR`; explicitly choose a directory instead of `auto` |
| `--output-format` | export `--format` |
| `--silent`, `--iconemoji`, `--noop` | `--quiet`, `--icon-emoji`, `--dry-run` |
| Upload `--filetype` | remove; the current Slack upload implementation does not consume it |
| `provider`, `endpoint`, `SCAT_PROVIDER`, `--provider` | remove; only bot Slack profiles remain |
| All-profile channel/user lists | explicit `--profile` invocation per workspace |
| Export mention substitution/global time sorting | original text/thread-grouped scli output |

No permanent compatibility aliases. Reject removed CLI flags with migration
guidance. Loading an old config containing removed fields fails with instructions
to back up and edit it; do not silently reinterpret a mock profile as a live bot
or rewrite credentials on load. Likewise reject nonempty `SCAT_PROVIDER` with
instructions to remove it. Other profile data and input limits remain usable.
Existing export files are not rewritten; scat has no import command.

### 8. Development plan and acceptance

1. **Design:** review this record independently; approve the detailed CLI,
   export interval, removal and failure contracts before implementation.
2. **Contract fixtures:** pin scli reference outputs for ordinary/bot messages,
   empty history/files, rich content, parents/replies, and file results. Document
   discrepancies between scli prose and its serialized output (`local_path`).
3. **Core and configuration:** Slack-only config/client, dependency injection,
   lazy resolution, cancellation, rate limits, credential hygiene and unit tests.
4. **Operations:** post/upload/stream and channel/user management with tests;
   preserve tested bot behavior and add swrite-compatible result output.
5. **Export:** parent-grouped export, pagination, broadcast deduplication,
   atomic output, file download semantics and failure tests. Golden-test the
   scli-compatible fields without masking differences other than clock/path data.
6. **Migration and release:** replace user docs and setup guide together in both
   languages; remove old provider code/tests once replacements cover behavior;
   update CONTRIBUTING, AGENTS, CHANGELOG and the old roadmap. Independent code
   review, `go test ./...`, `make check`, `make build`, four-platform builds and
   vulnerability scan must pass. Validate bot access with a dedicated fixture
   workspace; tests that post/create/invite require explicit permission to use it.
   Then follow org release/signing/`verify-release`/Homebrew/umbrella/catalog and
   `check-org.sh` gates. Publication follows the separate release gates.

Critical tests: multiple cursor pages including a short non-final page; repeated
or missing cursor; broadcast duplicates; parent outside interval; reply outside
interval; bot/user IDs; unchanged mentions; permission failure; download failure
retains metadata; redirect token containment; path traversal/symlink destination;
interrupted output preserves existing file; 429 exhaustion/cancellation; uncertain
mutation is not replayed; stream read/delivery errors; no API calls for dry-run
or unrelated identity lookups; profile isolation and rejected legacy config.

## Consequences

Service callers obtain one bot identity and a predictable Slack-specific CLI.
Export consumers can use the scli data model, including rich content and files.
Scripts/configurations must migrate for v2. Bot visibility still differs from
user visibility. Memory use and parent-selection time semantics remain explicit
limits of the scli-compatible export. Tests, not repeated prose alone, must keep
the implementation aligned with its reference.

## Alternatives considered

- **Keep the generic provider layer:** perpetuates unsupported-service complexity.
- **Copy scli wholesale:** imports user-authentication responsibilities and known
  implementation edge cases that do not define the intended export contract.
- **Only rename packages:** leaves initialization, partial failure and data drift.
- **Create a shared library immediately:** expands this project-scoped renewal
  into a fleet migration before the concrete contract has been verified.
- **Make the interval filter every reply:** changes scli's parent-selection
  semantics and cannot discover all old-parent replies by a bounded history call.
- **Preserve every old flag/schema:** carries the same complexity into v2 despite
  the maintainer's explicit permission to break compatibility.

## References

- [scli download investigation](https://github.com/nlink-jp/scli/blob/854e6a0/internal/slack/redirect.go): recorded login HTML at HTTP 200 from missing `files:read`; sending a token alone does not establish success.
- [Organization conventions](https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md): project-scoped ADR, design before implementation, independent verification.
- [scli export reference](https://github.com/nlink-jp/scli/blob/854e6a0/docs/en/EXPORT_FORMAT.md) and [serialized types](https://github.com/nlink-jp/scli/blob/854e6a0/internal/slack/types.go): normative baseline for this renewal; `local_path` follows the type rather than the inconsistent prose.
- [Development-process knowledge](https://github.com/nlink-jp/knowledge/blob/main/docs/en/development-process.md): extract behavior and preserve useful expectations; restructure the source of recurring drift.
- [Config/IO knowledge](https://github.com/nlink-jp/knowledge/blob/main/docs/en/config-and-io.md): account for persisted configuration when changing names. Here an explicit migration error is chosen over silent normalization into a different authentication target.
- [Security knowledge](https://github.com/nlink-jp/knowledge/blob/main/docs/en/security.md#a-credential-may-follow-a-redirect-only-within-the-domain-the-request-started-in): preserve credential containment during downloads.
- [API retry knowledge](https://github.com/nlink-jp/knowledge/blob/main/docs/en/mcp-server-design.md#retry-safety-comes-from-a-resource-you-named-not-from-a-token-whose-meaning-you-assumed): do not assume a timed-out mutation can be repeated safely.
- [File-finalization knowledge](https://github.com/nlink-jp/knowledge/blob/main/docs/en/config-and-io.md): cleanup and commit are different; keep input stable between measurement and transfer.
- [Slack upload workflow](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/) and [completion](https://docs.slack.dev/reference/methods/files.completeUploadExternal/): three stages, POST bytes, explicit destination, parent timestamp, one-shot completion.

Workspace instruction files and memories `project_export_format`,
`project_scli_bot_mode_paused`, `reference_go_authorization_across_redirects`, and
`feedback_slack_api_quirks`, `project_scli_mcp`, and the file-transfer lessons in
`project_slack_mcp_extender` were inspected. Their old stail baseline and paused
scli-bot proposal are superseded by this conversation's explicit scli-export /
separate-scat direction. Preserve their relevant field and download lessons.
No local memory paths or private values belong in the committed document.
Synthetic-test findings on timestamp rounding and file-response classification
are fed back to knowledge. No new live Slack measurement is claimed.

### Live verification follow-up — 2026-09-22

The built CLI's dedicated-channel E2E now verifies bot posts, rich replies,
stream threshold/EOF, environment-only tee, binary/HTML/JSON uploads and exact
downloaded bytes, parent-bound export and real invalid-token failure, with cleanup.
It exposed Slack's text/plain metadata plus force-download response for HTML/JSON;
attachment identity and full-size validation correct the false rejection.
See [BUILD](../BUILD.md#live-slack-e2e) for the repeatable command and precise scope.

### Manifest scope amendment — 2026-09-23

The shipped `slack-app-manifest.json` requests the union of the scopes in §6, not
one set per enabled operation. A manifest is imported once, before anyone knows
which operations an installation will use, and Slack applies it as a whole: a
per-operation manifest would mean several manifests, a decision the operator
cannot yet make, and a reinstall for every later feature. The union is the set
this CLI can use, never more, and `manifest_test.go` pins it in both directions.

§6 remains the rule for what the code may call. An operator who wants a narrower
installation deletes scopes from the copy before importing it; the commands whose
scopes were removed then fail with the method, code and needed scope, which is the
failure mode §6 already requires.
