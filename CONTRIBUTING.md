# Contributing to scat

First off, thank you for considering contributing to `scat`! Whether it's a bug report, a new feature, or an improvement to the documentation, your help is greatly appreciated.

## How to Contribute

There are several ways you can contribute to this project:

-   **Reporting Bugs**: If you find a bug, please open an issue and provide as much detail as possible.
-   **Suggesting Enhancements**: If you have an idea for a new feature or an improvement to an existing one, open an issue to discuss it.
-   **Submitting Pull Requests**: If you want to contribute code, please submit a pull request.

## Reporting Bugs

When opening an issue for a bug, please include the following:

-   **`scat` version**: Run `scat --version`.
-   **Operating System**: e.g., macOS 14.2, Ubuntu 22.04.
-   **What you did**: The exact command and arguments you used.
-   **What you expected to happen**: A clear description of what you thought would happen.
-   **What actually happened**: A description of the error, including any output or logs. Use `scat --debug ...` to get more detailed logs.

## Submitting Pull Requests

1.  **Fork the repository** and create your branch from `main`.
2.  **Make your changes**. Please ensure your code is formatted with `gofmt`.
3.  **Write clear commit messages**. We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. This helps in automating changelogs and makes the project history easier to read.
    -   `feat:` for new features.
    -   `fix:` for bug fixes.
    -   `docs:` for documentation changes.
    -   `refactor:` for code changes that neither fix a bug nor add a feature.
    -   `test:` for adding or improving tests.
4.  **Write tests for your changes**. All new features and bug fixes should be accompanied by appropriate unit tests. Ensure existing tests continue to pass.
    -   Run `make test` to execute the test suite.
    -   For CLI commands, consider using the `testprovider` (configured in your `config.json` with `"provider": "test"`) to mock external service interactions. This provider logs method calls to `stderr`, allowing you to verify command behavior.
5.  **Update the documentation** (`README.md`, etc.) if your changes affect it.
6.  **Submit the pull request**. Provide a clear description of the problem and your solution.

## Development Setup

For instructions on how to build the project, run tests, and other development tasks, please see [BUILD.md](./docs/BUILD.md).

To run tests, simply execute:
```bash
make test
```

### Code Structure Overview

-   `main.go`: The main entry point of the application.
-   `cmd/`: Contains all the command-line interface logic, using the `cobra` library. Each command and subcommand has its own file.
    -   `root.go`: The root command. Detects the operational mode (`SCAT_MODE`) at startup and resolves the configuration (from file or environment variables) once, storing it in `appcontext.Context`. Individual commands consume this pre-resolved config rather than performing their own file I/O.
    -   `providers.go`: The provider factory (`GetProvider`) and shared helpers including `requireCLIMode`, which guards commands unavailable in server mode.
-   `internal/appcontext/`: Defines `Context`, the application-wide settings struct passed through cobra's context. It holds the resolved `*config.Config` alongside flags like `Debug`, `Silent`, and `ServerMode`.
-   `internal/config/`: Handles loading and saving the configuration file, plus `DetectServerMode` (reads `SCAT_MODE`) and `BuildConfigFromEnv` (constructs a virtual `*Config` from environment variables for server mode).
-   `internal/provider/`: Defines the `provider.Interface` and contains the specific implementations for different services.
    -   `provider.go`: Defines the core `Interface` that all providers must implement.
    -   `types.go`: Defines shared data structures used in the provider interfaces (e.g., `PostMessageOptions`).
    -   `mock/`: A mock provider implementation, useful for testing.
    -   `slack/`: The Slack provider implementation, with its logic split into multiple files based on responsibility (`post.go`, `upload.go`, `exporter.go`, etc.).
-   `internal/export/`: Contains the data structures for the exported log format (`ExportedLog`, etc.).
-   `internal/util/`: Contains generic helper functions (e.g., `ToRFC3339`).