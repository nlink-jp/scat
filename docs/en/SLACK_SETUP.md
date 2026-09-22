# Slack bot setup

[日本語](../ja/SLACK_SETUP.ja.md)

Create or configure a Slack app with a bot user, grant scopes for the operations
you need, and install/reinstall it in the workspace. Use its bot token through
`scat profile set token` (hidden terminal input) or the service's `SCAT_TOKEN`.
Do not pass tokens in command arguments. See [README](../../README.md) for config
and server mode; scli user authentication is deliberately separate.

| Operation | Scopes / access |
|-----------|-----------------|
| Post | `chat:write`; `chat:write.customize` only for authorized custom name/icon |
| DM by user ID | `im:write` |
| Resolve channel names / inspect upload membership | `channels:read` / `groups:read`; DM info uses `im:read` or `mpim:read` |
| Resolve user names / list users | `users:read` |
| Resolve groups / expand group members | `usergroups:read` |
| Export history and replies | `channels:history`, `groups:history`, `im:history`, `mpim:history` by conversation type |
| Download files | `files:read` and access to the file |
| Upload files | `files:write`, plus destination resolution/membership access |
| Join public channels for writing | `channels:join` |
| Create / topic / purpose | `channels:manage` (public) / `groups:write` (private) |
| Invite | Public: `channels:manage` or `channels:write.invites`; private: `groups:write` or `groups:write.invites` |

This table is not a requirement to grant all scopes. Explicit channel/user IDs
avoid name-list APIs. Upload membership inspection still requires read access,
even with a channel ID. A bot must be invited to private channels. scat never
joins a channel to export history and never falls back to a user identity.

Each invocation verifies `bot_id` and `team_id` using `auth.test` before remote
operations. Help, local configuration and dry-run make no authentication request.
The current Slack replies reference lists bot scopes for channels and DMs; verify
the installed app's actual access before release instead of assuming historical
DM-only restrictions or assuming documentation proves live access.

## File-transfer pitfalls

Upload uses `files.getUploadURLExternal`, raw-byte POST to the returned HTTPS
URL, then `files.completeUploadExternal`. Byte transfer carries no API Bearer
or cookies; redirects are refused. A channel is explicitly supplied to completion;
a thread upload uses the parent timestamp. Completion is one-shot and not retried.
Missing destination membership fails before allocation unless a public channel can
be joined with the granted scope. File-share broadcast is not available.

Downloads can return a login HTML page with HTTP 200, including with Bearer sent
but `files:read` missing. Go normally strips Authorization on sibling-host
redirects; scat preserves it only within the verified Slack domain, never to
external hosts or on HTTPS downgrade. Do not remove containment checks to solve a
permission problem. Name/file warnings retain export metadata, while mandatory
history/reply failures are fatal.

Live post/upload/download tests require an explicitly authorized fixture workspace
and channel. Offline tests do not establish real Slack permissions or delivery.

References: [auth.test](https://docs.slack.dev/reference/methods/auth.test/),
[replies](https://docs.slack.dev/reference/methods/conversations.replies/),
[upload allocation](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/),
[upload completion](https://docs.slack.dev/reference/methods/files.completeUploadExternal/),
[ADR scope inventory](adr/0001-slack-bot-renewal.md).
