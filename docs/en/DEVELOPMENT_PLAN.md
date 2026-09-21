# Development Plan

This document outlines the development status and future roadmap for `scat`.

---

## Completed Milestones

### v1.5.0 (In Progress)

-   **Comprehensive Test Suite**: Implemented a robust testing framework, significantly increasing test coverage and project stability.
    -   Adopted a black-box testing approach using a dedicated `test_provider` to verify command logic without actual API calls.
    -   Added unit tests for all major commands: `post`, `upload`, `export log`, `channel list`, and `config init`.
    -   Added unit tests for the provider factory (`GetProvider`) and the `test_provider` itself.
-   **Code Refinements**: Improved code consistency and clarity.
    -   Refactored provider source files to use a consistent `_provider.go` naming convention.

### v1.4.0

-   **Configurable Config Path**: Implemented the `--config` global option to specify an alternative path for the configuration file, overriding the default `~/.config/scat/config.json`.

### v1.3.0

-   **Provider Interface Refactoring**: Adopted the "Options Struct" pattern for `PostMessage` and `PostFile` methods to improve consistency and extensibility across the provider interface.

### v1.2.0

-   **Log Export Feature**: Implemented the `scat export log` command.
-   **Major Refactoring**: Significantly improved the internal architecture for better maintainability.

### Pre-v1.2.0

-   **v0.1.9**: Added comprehensive documentation.
-   **v0.1.8**: Refactored the Slack provider.
-   **v0.1.7**: Implemented a new provider registration system and `config init` command.
-   **v0.1.6**: Implemented channel list caching in the provider.
-   **v0.1.5**: Centralized global flag handling.

---

## Future Roadmap

### 1. Testing Framework (Completed)

-   **Status**: A comprehensive suite of automated tests has been implemented for all major commands and internal logic, ensuring stability and preventing regressions. The framework uses a dedicated `test_provider` for black-box testing of command-line interfaces.

### 2. New Providers (Priority: Low)

-   With the improved provider registration system, adding new providers is now much easier. Potential candidates include:
    -   Discord
    -   Microsoft Teams

### 3. Advanced Features (Priority: Medium)

#### 3.1. Block Kit Support for `post` command (Completed)

-   **Goal**: Extended the `scat post` command to support posting rich messages using Slack's Block Kit framework.
-   **Approach**: Introduced a `--format blocks` flag to specify that the message content is a Block Kit JSON payload.
-   **Specification Details**:
    -   **`--format` Flag Behavior**:
        -   A new `string` flag `--format` will be added to the `post` command.
        -   Allowed values: `"text"` (default) and `"blocks"`.
        -   If `--format` is not specified, it defaults to `"text"`, treating the message content as plain text.
        -   If `--format blocks` is specified, the message content (from argument, `--from-file`, or stdin) will be parsed as a JSON string representing a Block Kit payload.
    -   **JSON Parsing and Error Handling**:
        -   When `--format blocks` is used, the command will attempt to `json.Unmarshal` the message content into a Go data structure (e.g., `[]map[string]interface{}`).
        -   If the JSON parsing fails, the `post` command will immediately return an error, providing early feedback to the user.
    -   **Exclusive Flag Handling**:
        -   `--format blocks` cannot be used simultaneously with `--stream`. The `--stream` flag is designed for continuous text input, which is incompatible with a single, structured Block Kit JSON payload.
        -   `--format blocks` cannot be used simultaneously with file upload-related flags (`--file`, `--filename`, `--filetype`, `--comment`). Block Kit is a message formatting mechanism, not a file upload mechanism.
-   **Implementation Phases**:
    -   **Phase 1: Command-Line Interface Extension (`cmd/post.go`)**
        -   Add the `--format` flag.
        -   Implement logic to read message content based on source (arg, file, stdin).
        -   Implement JSON parsing for `"blocks"` format and error handling for invalid JSON.
        -   Implement exclusive flag handling logic.
        -   Pass the parsed Block Kit `[]byte` (or `json.RawMessage`) to the `provider.PostMessageOptions`.
    -   **Phase 2: Provider Interface Extension (`internal/provider/types.go`)**
        -   Add a `Blocks []byte` field to the `provider.PostMessageOptions` struct.
        -   Define behavior: If `Blocks` is non-nil/non-empty, the `Text` field in `PostMessageOptions` should be ignored by provider implementations (as Slack API prioritizes `blocks` over `text`).
    -   **Phase 3: Provider Implementation Update**
        -   **`internal/provider/slack/slack_provider.go`**: Modify the `PostMessage` method to include the `blocks` parameter in the Slack API `chat.postMessage` call when `opts.Blocks` is present.
        -   **`internal/provider/mock/mock_provider.go`**: Update the `PostMessage` method to log the `Blocks` content when present.
        -   **`internal/provider/test_provider/test_provider.go`**: Update the `PostMessage` method to log the `Blocks` content when present, facilitating black-box testing.
    -   **Phase 4: Test Addition (`cmd/post_test.go` and provider tests)**
        -   Add new test cases for `cmd/post_test.go` to cover:
            -   Successful posting of Block Kit JSON from argument, file, and stdin using `--format blocks`.
            -   Error handling for invalid Block Kit JSON.
            -   Error handling for exclusive flag combinations (`--format blocks` with `--stream` or file upload flags).
        -   Add test cases to `mock_provider_test.go` and `test_provider_test.go` to verify that `PostMessage` correctly handles and logs the `Blocks` field.
-   **Considerations**:
    -   Ensure robust error handling for all parsing and API interactions.
    -   Maintain backward compatibility with existing plain text message posting functionality.

#### 3.2. Persistent Caching

-   **Goal**: Implement a file-based caching mechanism for data like channel lists. This would persist the cache between `scat` command invocations, further reducing API calls for users who run the command frequently.
