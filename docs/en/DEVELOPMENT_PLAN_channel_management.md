# Development Plan: Channel Management Features

> Historical v1 plan. Current behavior is specified in [ADR-0001](adr/0001-slack-bot-renewal.md) and [README](../../README.md).

This document outlines the development plan for implementing new channel creation and user invitation features in `scat`.

## 1. Features

- **`scat channel create`**: A new command to create a public or private Slack channel.
  - It supports setting the description and topic.
  - It supports inviting users and user groups upon creation.
- User specification supports both User IDs (e.g., `U12345`) and mention names (e.g., `@username`).
  - Note: A separate `scat channel invite` command was not implemented; invitation is handled directly by `scat channel create`.

## 2. Design Policy

- All parameters for provider methods, including required ones, will be passed via a single `Options` struct to maintain consistency with existing patterns (`Post`, `ExportLog`).
- The implementation will be layered, separating low-level API calls from high-level provider logic and CLI command logic.

## 3. Development Steps

### Step 1: Add Capabilities Definitions
- **File:** `internal/provider/provider.go`
- **Action:** Add `CanCreateChannel` boolean flag to the `Capabilities` struct. (Note: `CanInviteUsers` was not added as a separate capability, as invitation is handled within `CreateChannelOptions`.)

### Step 2: Define Interfaces and Types
- **File:** `internal/provider/types.go`
- **Action:** Define `CreateChannelOptions` struct. (Note: `InviteToChannelOptions` was not defined as a separate struct.)
- **File:** `internal/provider/provider.go`
- **Action:** Add the following method to the `provider.Interface`:
  - `CreateChannel(opts CreateChannelOptions) (string, error)` (Note: Returns channel ID string, not a `Channel` struct. `InviteToChannel` was not added as a separate method.)

### Step 3: Update All Provider Implementations
- **`internal/provider/slack/`**:
  - **`api.go`**: Added new low-level functions to call the `conversations.create` and `conversations.invite` Slack APIs.
  - **`types.go`**: Added structs to unmarshal the responses from the new APIs.
  - **`slack_provider.go`**: Updated the `Capabilities()` method to return `true` for the new features.
  - **`channel.go` (or similar)**: Implemented the `CreateChannel` method, calling the new functions in `api.go`. (Note: `InviteToChannel` method was not implemented separately.)
- **`internal/provider/mock/mock_provider.go`**: Added dummy implementations for the new interface methods.
- **`internal/provider/testprovider/test_provider.go`**: Added stateful, in-memory fake implementations for the new interface methods.

### Step 4: Implement CLI Commands
- **Files:** Created `cmd/channel_create.go`. (Note: `cmd/channel_invite.go` was not created as a separate command; invitation is handled by `scat channel create`.)
- **File:** `cmd/channel.go`
- **Action:**
  - Registered the new commands.
  - The command logic:
    1. Checks the provider's capabilities first.
    2. Resolves user mention names to IDs using `ResolveUserID` and `ResolveUserGroupID` functionality.
    3. Builds the appropriate `Options` struct from command-line arguments and flags.
    4. Calls the corresponding provider method.

### Step 5: Implement Tests
- **Provider Tests (`internal/provider/slack/*_test.go`):**
  - Used `net/http/httptest` to create a mock HTTP server.
  - Wrote unit tests to verify that the provider methods send the correct requests to the Slack API and handle responses correctly.
- **CMD Tests (`cmd/*_test.go`):**
  - Used the `test_provider` to test the CLI commands.
  - Verified that command flags are parsed correctly and that the `test_provider`'s in-memory state is updated as expected.

### Step 6: Update Documentation
- **`README.md`, `README.ja.md`**: Add usage instructions for the new commands. (Note: This step is pending and needs to be done separately.)
- **`docs/en/SLACK_SETUP.md`, `docs/ja/SLACK_SETUP.ja.md`**: Added the newly required OAuth scopes to the setup instructions:
  - `channels:manage`
  - `groups:write`
  - `usergroups:read` (Note: This scope was added during implementation.)
