# Slack bot setup

[日本語](../ja/SLACK_SETUP.ja.md)

## Create an app from the manifest

Use [slack-app-manifest.json](../../slack-app-manifest.json) to configure the bot
and its scopes in one import. Release archives hold only the binary, README and
license, so take this file from the repository. It contains no tokens, workspace
IDs or callback URLs and can be reused across workspaces.

1. Open [Your Apps](https://api.slack.com/apps), choose **Create New App → From a manifest**,
   and select the workspace where the bot will run.
2. Choose **JSON**, paste the complete file, and continue to the permissions review.
   Review the scopes and create the app. Its default app/bot name is `scat`;
   change the display names in the manifest first if needed.
3. Choose **Install to Workspace** (or request your workspace administrator's approval).
   After installation, copy **Bot User OAuth Token** from **OAuth & Permissions**.
4. Run the commands below and paste that token at the hidden prompt. Use the bot
   token (`xoxb-`), not a user or app-level token.

```sh
scat config init
scat profile set token
scat profile set channel C0123456789
```

Add the bot to the channels it should access, particularly private channels.
The manifest grants scopes, not membership or access to every workspace conversation.
For services, inject the same bot token as `SCAT_TOKEN` with `SCAT_MODE=server`;
see [README](../../README.md). Tokens do not belong in the manifest or command arguments.

The template covers the existing bot CLI features: text/rich posts and custom
name/icon, file upload/download, channel/thread export, channel creation and
invitations to existing channels, and user/group lookup. `channels:manage` and
`groups:write` cover channel creation and invitations, so separate invite-only
scopes are unnecessary in this template. It does not request user scopes.

scat calls the Web API and needs no Events API, Socket Mode, incoming webhook,
public server or OAuth redirect URL. Token rotation is disabled in the template
because scat does not refresh expiring OAuth tokens. The app-level token and
Socket Mode settings used by stail are not needed here.

## Update an existing app

For an existing scat app, save a copy of its current **App Manifest**, then use
its JSON editor to replace `oauth_config.scopes.bot` with the template's entire
bot-scope list in one paste, keeping other app settings intact. Save the change
and **Reinstall to Workspace** to grant newly added scopes. Updating the manifest
alone does not update an already installed token's permissions. Keep the
non-rotating bot-token configuration required by scat.

## Scope reference and customization

The supplied template covers the full feature set, rather than a posting-only
minimum. For a limited deployment, remove unused scopes from a copy before
importing it; this table explains which operations will then be unavailable.
No manual per-scope assignment is needed when using the unmodified manifest.

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

Explicit channel/user IDs
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

Slack may classify uploaded HTML/JSON as `text/plain` and serve the original bytes
as `application/force-download`. scat checks authenticated Slack origin, matching
`Content-Disposition` filename and recorded size before accepting this MIME mismatch.
A filename extension or HTTP 200 alone is insufficient; recognized login HTML is
still rejected. The real round-trip suite is documented in [BUILD](BUILD.md#live-slack-e2e).

Live post/upload/download tests require an explicitly authorized fixture workspace
and channel. Offline tests do not establish real Slack permissions or delivery.

References: [auth.test](https://docs.slack.dev/reference/methods/auth.test/),
[replies](https://docs.slack.dev/reference/methods/conversations.replies/),
[upload allocation](https://docs.slack.dev/reference/methods/files.getUploadURLExternal/),
[upload completion](https://docs.slack.dev/reference/methods/files.completeUploadExternal/),
[ADR scope inventory](adr/0001-slack-bot-renewal.md).

Manifest references: [configuration guide](https://docs.slack.dev/app-manifests/configuring-apps-with-app-manifests/),
[field reference](https://docs.slack.dev/reference/app-manifest/).
The repository test checks JSON structure, supported bot scopes and disabled
unsupported auth modes; it does not create or install an app in Slack.
