# Building scat

This document provides instructions for building the `scat` binary from source and running related development tasks.

---

## Requirements

-   **Go**: Version 1.21 or later.
-   **Make**: A standard `make` utility is used to simplify build tasks.
-   **(macOS only)**: `lipo` and `codesign` are required for creating universal binaries. These are typically available with Xcode Command Line Tools.

## Standard Build

To build the `scat` binary for your current operating system and architecture, run:

```bash
make build
```

The compiled binary will be placed in the `bin/<os>-<arch>/` directory (e.g., `bin/darwin-arm64/scat`).

## Installation

You can install the compiled binary to a directory in your `PATH`.

-   **System-wide installation (requires sudo)**:

    This installs `scat` to `/usr/local/bin`.

    ```bash
    sudo make install
    ```

-   **User-local installation**:

    This installs `scat` to `~/bin`. Make sure `~/bin` is in your shell's `PATH`.

    ```bash
    make install PREFIX=~
    ```

## Development Tasks

The `Makefile` includes several other useful targets for development:

-   **Run tests**:

    ```bash
    make test
    ```

-   **Run linters**:

    This project uses `golangci-lint`.

    ```bash
    make lint
    ```

-   **Check for vulnerabilities**:

    This uses `govulncheck`.

    ```bash
    make vulncheck
    ```

-   **Clean up build artifacts**:

    ```bash
    make clean
    ```

## Cross-Compilation and Release Packaging

To build `scat` for all supported platforms (macOS Universal, Linux/amd64, Windows/amd64) and package them into archives for a release, run:

```bash
make cross-compile
```

This command will:
1.  Build binaries for each target platform.
2.  Create a universal binary for macOS.
3.  Package the binaries into `.tar.gz` (for macOS/Linux) and `.zip` (for Windows) archives in the `bin/` directory.