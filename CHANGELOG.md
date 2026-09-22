# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `make clean` no longer fails partway. The module cache inside the repository
  holds read-only files, so `rm -rf .cache` stopped with "Permission denied" and
  left a half-deleted cache; `go clean -modcache` removes it first.

### Internal

- `make verify-release`'s Linux-archive check could not see AppleDouble members.
  macOS `tar` folds `._` members out of a plain listing, so an archive built
  with `--no-xattrs` but without `COPYFILE_DISABLE=1` passed with stray `._`
  files inside; the check now lists with `--options 'tar:!mac-ext'`. The entry
  list is compared in the C locale, so a correct archive is not refused when the
  gate runs under a UTF-8 locale, and a refusal states the expected list. The
  v2.1.1 archives were built with both settings; only the check was blind.

## [2.1.1] - 2026-09-23

### Fixed

- Actually keep macOS extended attributes out of the Linux archives. v2.1.0 set
  `COPYFILE_DISABLE=1`, which does not stop macOS `tar` from writing them as
  `LIBARCHIVE.xattr.*` / `SCHILY.xattr.*` pax headers — the archives built for
  v2.1.0 still carried a Dropbox attribute and `com.apple.provenance`.
  `tar --no-xattrs` removes them. v2.1.0 published no archives.

### Added

- `make verify-release` now also judges each Linux archive: no macOS metadata
  entries, no extended attributes as pax headers, and exactly the canonical
  binary, `README.md` and `LICENSE`. The claim is checked, not asserted.

## [2.1.0] - 2026-09-23

### Added

- Add a bot-only Slack App Manifest (`slack-app-manifest.json`) whose scopes cover
  the existing CLI features, and document manifest-based installation and app
  updates in English and Japanese. Tests pin the scope set in both directions and
  reject user scopes, credentials and unknown manifest fields.

### Fixed

- Keep macOS extended attributes out of the Linux release archives. `tar` on macOS
  stored them as `LIBARCHIVE.xattr.*` / `SCHILY.xattr.*` pax headers, which GNU tar
  reports as unknown keywords on extraction; `COPYFILE_DISABLE=1` suppresses them.

### Docs

- Document `channel create --private`, the `profile add` flags and the `upload`
  short forms. The v2 rewrite dropped them from the READMEs while the flags stayed.
- Record in [ADR-0001](docs/en/adr/0001-slack-bot-renewal.md) why the manifest
  requests the union of the scopes rather than one set per enabled operation.

### Internal

- Cover `channel create --private`, which had no test.
- Remove the v1 `.gemini/GEMINI.md` agent guide. It described the removed provider
  interface and a workflow that contradicts AGENTS.md and CONTRIBUTING.md.

## [2.0.0] - 2026-09-22

### Removed

- **Breaking:** Remove runtime provider registration, capabilities, mock/test
  providers, endpoint fields and `SCAT_PROVIDER`; scat is Slack-only and
  bot-authenticated. Named bot profiles are preserved; legacy configuration is
  rejected with explicit migration guidance.

### Changed

- Replace startup-wide resolution with invocation-scoped dependencies, lazy ID/name
  resolution, bot identity verification and environment-only service configuration.
- Align export with scli: rich attachments/blocks, raw text, explicit empty arrays,
  always-present local_path, parent/reply grouping, cursor pagination and broadcast
  deduplication. Required retrieval failures abort; optional failures retain metadata.
- Rename export and flags, add result JSON and timestamps, wire thread/unfurl fields,
  preserve stdin streaming with bounded batches and propagate failures/cancellation.
- Stage uploads with bounded memory, verify membership before allocation and never
  replay completion. Confine download credentials, validate HTML/JSON responses,
  enforce byte limits and atomically replace anchored attachment targets.
- Write credential files atomically at 0600 and preserve zero-as-unlimited limits.
  Require Go 1.25+ for anchored rename. Add synthetic export fixtures, injected
  transport/command tests and make fmt / make vulncheck targets.

### Added

- Add `make e2e`: uncached tests driving the built CLI against a dedicated real
  Slack channel, including post/reply/stream, service-mode tee, binary/HTML/JSON
  file round trips, export schema/parent bounds, invalid credentials and cleanup.
  Missing live configuration fails instead of silently skipping validation.
- Add `make verify-release`, which refuses to release a darwin zip that is
  un-notarized, stale, does not unpack, does not run, or holds a build from
  another tag. Every step fails on its own; only the `spctl` line is
  informational. Matches the org template (CONVENTIONS.md §Code Signing →
  Verifying a release).
- Add `make brew`, which generates the Homebrew formula for the tap from the
  built darwin-arm64 zip.

### Fixed

- Accept authenticated Slack forced-download attachments when their filename and
  recorded size match, even when Slack labels HTML/JSON as `text/plain`. Real
  round-trip tests exposed the MIME mismatch; login responses and untrusted or
  mismatched attachments remain rejected.
- Give cancellation priority over stream EOF and check it before committing a
  downloaded attachment; retain the previous destination on cancellation.

### Docs

- Move the English documents under `docs/en/` and add the Japanese counterparts
  under `docs/ja/`, per the org documentation layout.
- Label historical development plans as v1 records and explicitly document v2
  invitations to existing channels, separately from live verification coverage.
- Clarify the organization main/submodule workflow; independent review does not
  require pull requests. Remove mutable release-status prose from the READMEs.
- Record accepted [ADR-0001](docs/en/adr/0001-slack-bot-renewal.md) and its
  Japanese mirror; replace the v1 setup/build/export guides with v2 contracts,
  migration steps and project-specific agent guidance.
- Replace the old export documentation with the scli-compatible v2 schema,
  including always-present file arrays and local_path, attachments and blocks.

## [1.15.0] - 2026-07-12

### Removed

- **darwin/amd64 (Intel) pre-built binary.** macOS releases now ship
  **arm64 only**, per the org-wide policy (darwin is Apple-Silicon only; no
  universal binaries). Intel Mac users can build from source.

### Changed

- **Linux release archives are now `.tar.gz`** (darwin/windows remain `.zip`),
  per `nlink-jp/.github` CONVENTIONS.md §Release Archive Standard.
- **`LICENSE` is now bundled** in every release archive alongside `README.md`.
- **darwin code-signature identifier** is now the canonical `scat`
  (was `scat-darwin-arm64`), set via `codesign -i` so it stays stable after
  the archived binary is renamed to its canonical name.
- **Dropped the `-s -w` linker strip flags**, aligning `LDFLAGS` with the
  org-standard form; also avoids a false-positive antivirus quarantine of
  the stripped Windows binary during cross-build.

No change to the binary's behaviour — a packaging / build-config release.

## [1.14.1] - 2026-05-22

### Changed

- **Darwin releases are now Developer ID signed and Apple-notarized.**
  `scat-v1.14.1-darwin-{amd64,arm64}.zip` carry full Apple Developer
  ID Application signatures and notarization tickets from Apple. End
  users on macOS no longer need to bypass Gatekeeper with right-click
  → Open or `xattr -d com.apple.quarantine` on first launch; local
  users who place `scat` under Dropbox-synced (or any other
  FileProvider-managed) paths are no longer killed by macOS's
  ad-hoc + provenance distrust policy. Pipeline:
  `scripts/codesign-darwin.sh` + `scripts/notarize-darwin.sh`, driven
  by `make package`. Adopts the org-wide convention in
  `nlink-jp/.github` CONVENTIONS.md §Code Signing.
- **Release zips now include README.md** alongside the binary,
  aligning with the chatops-series CLAUDE.md release convention.

No behaviour change to the binary itself — feature-wise this is
identical to v1.14.0.

## [1.14.0] - 2026-03-28

### Changed
- Migrated to nlink-jp organisation — module path updated to `github.com/nlink-jp/scat`
- Build output directory changed from `bin/` to `dist/`

## [1.13.0] - 2026-03-18

### Features

- **`channel invite` command**: New subcommand to invite one or more users or user groups to an existing channel. Users can be specified by display name or `@`-prefixed name; user groups by handle. Resolves names to IDs internally.
- **`user list` command**: New `scat user` command group with a `list` subcommand that lists all workspace users with their display names and IDs. Supports `--json` for structured output.
- **`channel list` now includes IDs**: The output of `scat channel list` has been updated to display both channel name and channel ID in a table format. JSON output now returns `{"name": "...", "id": "..."}` objects instead of plain strings.

### Provider Interface

- `ListChannels()` now returns `[]provider.Channel` (with `ID` and `Name` fields) instead of `[]string`.
- Added `ListUsers() ([]provider.UserInfo, error)` to the provider interface.
- Added `InviteToChannel(opts InviteToChannelOptions) error` to the provider interface.
- Added `CanListUsers` and `CanInviteToChannel` capability flags.

### Documentation

- **README.md / README.ja.md**: Added documentation for `channel invite`, `user list`, `channel list --json`, `channel create` flags, `profile add` flags, and `profile set` keys. Fixed `--profile` flag placement (per-command, not global). Added DM usage examples to Japanese README.

## [1.12.0] - 2026-03-16

### Features

- **Configurable DoS protection limits in server mode**: Added `SCAT_MAX_FILE_SIZE` and `SCAT_MAX_STDIN_SIZE` environment variables to customize upload and stdin size limits when running in server mode (`SCAT_MODE=server`). Defaults remain 1 GB and 10 MB respectively. Invalid values (non-numeric or negative) return an error at startup.

### Bug Fixes

- Fixed `omitempty` tag on the `Limits` struct field in `Profile`, which had no effect since `encoding/json` never treats a zero-value struct as empty.

### Tests

- Added command-level integration tests for server mode covering: mode validation, `post` success, missing required env vars, `--config` flag rejection, `profile`/`config init` command blocking, custom limits, and invalid limit values.

## [1.11.0] - 2026-03-16

### Features

- **Server Mode for container and CI deployments**: Added `SCAT_MODE=server` to run `scat` without a config file, reading all profile settings from environment variables (`SCAT_PROVIDER`, `SCAT_TOKEN`, `SCAT_CHANNEL`, `SCAT_USERNAME`). This enables secure token injection via Kubernetes Secrets or CI environment variables without storing tokens on disk.

### Refactoring

- **Centralized config loading**: The configuration file (or virtual config built from env vars) is now resolved once at startup in `PersistentPreRunE` and stored in `appcontext.Context`. Individual commands no longer perform their own file I/O for config loading.

### Documentation

- **README.md / README.ja.md**: Added "Server Mode" section with usage examples including a Kubernetes deployment example.

## [1.10.0] - 2025-08-20

### Features

- **Channel Creation with User/User Group Invitation**: The `channel create` command now supports inviting users and user groups directly when creating a new channel.
  - User and user group names (handles) are resolved to their respective IDs.

### Documentation

- **Updated Slack Setup Guide**: The `docs/SLACK_SETUP.md` and `docs/SLACK_SETUP.ja.md` files have been updated to include the necessary `usergroups:read` OAuth scope for inviting user groups.
- **Updated Channel Management Development Plan**: The `docs/DEVELOPMENT_PLAN_channel_management.md` has been updated to reflect the actual implementation details of channel creation and user invitation.

## [1.9.0] - 2025-08-17

### Features

- **Thread Export Support**: The `export log` command now correctly fetches and exports threaded conversations.
  - It retrieves all replies for each thread.
  - The exported message format now includes `is_reply` and `thread_timestamp_unix` fields to represent thread relationships.
  - The plain text output format now indents replies for better readability.

## [1.8.0] - 2025-08-16

### Features

- **Direct Message Support**: Added the ability to send direct messages (DMs) via the `post` and `upload` commands using a new `--user` flag.
- **User Resolution**: The `--user` flag can accept both user IDs (e.g., `U123ABCDE`) and mention names (e.g., `@username`). The provider will automatically resolve mention names to user IDs.

### Refactoring

- **Channel Specification**: Refactored the channel specification logic for the `post` and `upload` commands to be more explicit and robust, passing the target channel in `PostOptions` rather than mutating the profile.

 ## [1.7.0] - 2025-08-15

 ### Features

- **Enhanced `export log` output**: Improved the `scat export log` command output to provide more comprehensive user information.
    - Populated `user_id` with `bot_id` for bot messages, ensuring a consistent identifier for all message types.
    - Introduced a new `post_type` field (`"user"` or `"bot"`) to clearly distinguish between human and bot posts.

 ### Documentation

- **New Export Data Format documentation**: Created `docs/EXPORT_FORMAT.md` to detail the structure of the `export log` output.
- Updated `README.md` to link to the new export data format documentation.

 ## [1.6.0] - 2025-08-14 
 
 ### Features

- **Block Kit Support for `post` command**: Extended the `scat post` command to support posting rich messages using Slack's Block Kit framework.
    - Introduced a `--format blocks` flag.
    - Implemented JSON parsing for Block Kit content, handling both `{"blocks": [...]}` and `[...]` root formats.
    - Added validation for `--format` flag values and exclusive handling with `--stream`.
    - Updated `provider.PostMessageOptions` to include a `Blocks` field.
    - Modified Slack, mock, and test providers to correctly handle and log Block Kit messages.
    - Added comprehensive unit tests for Block Kit posting scenarios and error handling.

## [1.5.0] - 2025-08-14

### Features

- **Comprehensive Test Suite**: Implemented a robust testing framework, significantly increasing test coverage and project stability.
    - Added unit tests for `post`, `upload`, `export log`, `channel list`, and `config init` commands.
    - Added unit tests for the `GetProvider` function (provider factory) and the `test_provider` itself.

### Refactoring

- **Provider File Naming**: Renamed provider source and test files (`mock`, `slack`, `test_provider`) to use a consistent `snake_case` naming convention.
- **Test Helpers**: Moved `setupTest` helper function to `test_helpers.go` for shared use across `cmd` package tests.

### Documentation

- **Development Plan**: Updated `DEVELOPMENT_PLAN.md` to reflect the completed comprehensive testing framework and refactoring efforts.
- **Contributing Guide**: Updated `CONTRIBUTING.md` with guidelines for writing tests, reflecting the new comprehensive test suite.
- **Removed Testing Plan**: Deleted `TESTING_FRAMEWORK_PLAN.md` as its content has been integrated into `DEVELOPMENT_PLAN.md`.

## [1.4.0] - 2025-08-12

### Features

- **Configurable Config Path**: Added `--config` global option to specify an alternative path for the configuration file, overriding the default `~/.config/scat/config.json`.

### Fixes

- **Compilation Errors**: Resolved compilation errors introduced by changes to `config` package function signatures.
- **Linting Issues**: Fixed linting errors related to unchecked error returns and unused imports.

### Refactoring

- **Config Package**: Modified `config.Load()`, `config.Save()`, and `config.GetConfigPath()` to accept and utilize a configurable path.
- **CLI Commands**: Updated all `cmd` package files that load or save configuration to use the new configurable path.
- **Error Handling**: Improved error handling for `MarkFlagRequired` in `cmd/export_log.go` to avoid panics.

## [1.3.0] - 2025-08-12

### Refactoring

- **Provider Interface**: Adopted the "Options Struct" pattern for `PostMessage` and `PostFile` methods to improve consistency and extensibility across the provider interface.

## [1.2.0] - 2025-08-12

### Features

- **Channel Log Export**: Added a new `scat export log` command.
    - Supports JSON and plain text output formats (`--output-format`).
    - Allows exporting the main log to standard output or a file (`--output`).
    - Supports downloading attached files to a specified or auto-generated directory (`--output-files`).
    - Provides time range filtering (`--start-time`, `--end-time`).
    - Automatically resolves user mentions (e.g., `<@U...>` to `@username`).
    - Standardizes timestamps in the output JSON to include both human-readable RFC3339 and original Unix timestamp formats for compatibility and precision.
- **Slack Provider**: Implemented an auto-join feature for the `export log` command, mirroring the behavior of `post`.

### Fixes

- **CLI Robustness**: Removed ambiguous short-form flags (e.g., `-o`) to prevent misinterpretation of command-line arguments.
- **File Permissions**: Hardened permissions for all created files and directories to `0600` (files) and `0700` (directories) respectively, enhancing security.

### Refactoring

- **Provider Architecture**: Encapsulated all export logic within the provider layer, removing the generic `Exporter` engine in favor of a simpler, more robust design where each provider is fully responsible for its own export process.
- **Slack Provider**: Decomposed the Slack provider's methods into smaller, single-responsibility files (`post.go`, `upload.go`, `exporter.go`, etc.) for improved maintainability.

## [1.1.0] - 2025-08-11

### Features
- **Slack Provider**: Automatically joins the channel if the bot is not in it when attempting to upload a file, then retries the upload. (`205db0b`)

## [1.0.0] - 2025-08-11

This is the first official stable release of `scat`. It includes a wide range of features and improvements implemented since the initial development.

### Features
- **Provider-Based Architecture**: Implemented a flexible provider model, making it easy to add support for new services beyond Slack. (`eb581c6`)
- **Slack Provider**:
    -   Post text messages and upload files. (`55d0516`)
    -   Automatically join public channels when posting if the bot is not already a member. (`deaa521`)
    -   List available channels with `scat channel list`.
    -   Override username and icon emoji when posting messages. (`92bdcd5`)
- **Comprehensive CLI**:
    -   `post` command for sending text messages from arguments, files, or stdin. (`55d0516`)
    -   `upload` command for sending files from a path or stdin. (`55d0516`)
    -   `post --stream` for continuously streaming content, like logs. (`55d0516`)
    -   Full profile management (`add`, `list`, `use`, `set`, `remove`).
    -   `config init` command for explicit and safe configuration initialization. (`5301d87`)
- **Global Flags**:
    -   `--debug` for verbose logging. (`92bdcd5`)
    -   `--noop` for dry runs.
    -   `--silent` to suppress non-error output.

### Fixes
- **Centralized Flag Handling**: Correctly propagated global flags (`--debug`, `--noop`, `--silent`) to all commands. (`8bfe7c7`)
- **Duplicate Error Messages**: Prevented errors from being printed twice by cobra and main. (`0490dc7`)
- **Slack File Upload**:
    -   Migrated from the deprecated `files.upload` API to the modern `files.getUploadURLExternal` and `files.completeUploadExternal` methods. (`59d5faa`)
    -   Reverted an incorrect implementation of file sharing that caused `invalid_blocks` errors. (`8c8ffe1`)
- **Input Limits**: Added configurable limits for file and stdin sizes to prevent excessively large uploads. (`9624a53`)

### Refactoring
- **Provider Registration**: Implemented an explicit provider registry in the `cmd` package, removing direct dependencies from commands to specific provider implementations. (`169279f`)
- **Slack Provider Modularity**: Split the monolithic `slack.go` file into smaller, more focused files (`api.go`, `channel.go`, `types.go`) for improved maintainability. (`08960c4`)
- **Configuration Handling**: Separated the concerns of loading and initializing configuration. (`5301d87`)
- **Application Context**: Introduced an `appcontext` to pass global settings cleanly, reducing coupling. (`d882399`)
- **Channel Caching**: Optimized the Slack provider to cache the channel list on initialization, significantly reducing API calls. (`99c3a66`)

### Documentation
- **Comprehensive READMEs**: `README.md` and `README.ja.md` were completely overhauled with detailed setup instructions, usage examples, and a full command reference. (`d9ce1fe`, `4e2881e`)
- **Added `BUILD.md`**: Provides clear instructions for building the project from source. (`9f3d670`)
- **Added `CONTRIBUTING.md`**: Outlines guidelines for contributing to the project. (`9f3d670`)
- **Added `SLACK_SETUP.md`**: A dedicated guide for configuring the Slack provider. (`3e0e407`)
- **Updated `DEVELOPMENT_PLAN.md`**: The development plan is now up-to-date with completed milestones and a future roadmap. (`54b6d4e`)
- **Removed Build Status Badge**: Removed the non-functional build status badge until a CI/CD pipeline is implemented. (`c600a31`)
