# scat: A General-Purpose Command-Line Content Poster

<a href="https://flatt.tech/oss/gmo/trampoline" target="_blank"><img src="https://flatt.tech/assets/images/badges/gmo-oss.svg" height="24px"/></a>

`scat` is a versatile command-line interface for sending content from files or standard input to a configured destination, such as Slack. It is inspired by `slackcat` but is designed to be more generic and extensible.

---

## Features

- **Post text messages**: Send content from arguments, files, or stdin.
- **Send Direct Messages (DMs)**: Send messages or files directly to users by their ID or mention name.
- **Upload files**: Upload files from a path or stdin.
- **Stream content**: Continuously stream from stdin, posting messages periodically.
- **Export channel logs**: Export message history from a channel to a structured JSON file or stdout.
- **Profile management**: Configure multiple destinations and switch between them easily.
- **Extensible providers**: Currently supports Slack and a mock provider for testing.

## Installation

Download the latest binary for your system from the [Releases](https://github.com/magifd2/scat/releases) page.

Alternatively, you can build from source:

```bash
make build
```

## Initial Setup

Before you can start posting, you need to create a configuration file.

1.  **Initialize the config file**:

    Run the following command to create a configuration file (`~/.config/scat/config.json`) in the default location:

    ```bash
    scat config init
    ```

    **Important**: This configuration file contains sensitive information such as Slack tokens. For security, it is highly recommended to set the file permissions to `600` (read and write only for the owner).


2.  **Configure a Profile**:

    The default profile uses a mock provider, which is useful for testing. To post to a real service like Slack, you need to add a new profile.

    For detailed instructions on setting up a Slack profile, please see the **[Slack Setup Guide](./docs/SLACK_SETUP.md)**.

    Here is a quick example of how to add a new Slack profile:

    ```bash
    # This will prompt you to enter your Slack Bot Token securely.
    scat profile add my-slack-workspace --provider slack --channel "#general"
    ```

3.  **Set the Active Profile**:

    Tell `scat` to use your new profile by default:

    ```bash
    scat profile use my-slack-workspace
    ```

## Usage Examples

Here are some common ways to use `scat`.

### Posting Text Messages (`post`)

-   **From an argument to a channel**:
    `scat post --channel "#random" "Hello from the command line!"`

-   **From standard input (pipe) to the default channel**:
    `echo "This message was piped." | scat post`

-   **As a Direct Message to a user (by mention name)**:
    `scat post --user @someuser "Hello, this is a direct message."`

-   **As a Direct Message to a user (by user ID)**:
    `scat post --user U123ABCDE "You can also use a user ID for DMs."`

### Posting Block Kit Messages (`post` with `--format blocks`)

-   **From an argument (JSON string)**:
    `scat post --format blocks '[{"type": "section", "text": {"type": "mrkdwn", "text": "Hello, Block Kit from argument!"}}]'`

-   **From a file (JSON file)**:
    (Create a file named `blocks.json` with Block Kit JSON content)
    `scat post --format blocks --from-file ./blocks.json`

-   **From standard input (JSON pipe)**:
    `echo '[{"type": "section", "text": {"type": "mrkdwn", "text": "Hello, Block Kit from stdin!"}}]' | scat post --format blocks`

### Uploading Files (`upload`)

-   **Upload a file to a channel**:
    `scat upload --file ./report.pdf --channel "#reports"`

-   **Upload a file as a DM to a user with a comment**:
    `scat upload --file ./screenshot.png --user @someuser -m "Here is the screenshot you requested."`

### Exporting Channel Logs (`export log`)

Exports message history from a channel to a structured JSON file or stdout. It fetches all messages, including replies in threads. For details on the output format, including fields like `user_id`, `user_name`, and `post_type`, please refer to the [Export Data Format documentation](./docs/EXPORT_FORMAT.md).

-   **Export to stdout and pipe to `jq`**:
    `scat export log --channel "#random" | jq .`

-   **Export to a specific file**:
    `scat export log -c "#random" --output "my-export.json"`

-   **Export and download attached files to an auto-generated directory**:
    `scat export log -c "#random" --output-files auto`

-   **Export log to stdout and download files to a specific directory**:
    `scat export log -c "#random" --output - --output-files "./attachments"`

## Command Reference

### Global Flags

| Flag      | Description                                      |
| --------- | ------------------------------------------------ |
| `--config <path>` | Specify an alternative path for the configuration file. Not available in server mode. |
| `--profile <name>` | Use a specific profile for the command.          |
| `--debug`   | Enable verbose debug logging.                    |
| `--silent`  | Suppress success messages.                       |
| `--noop`    | Perform a dry run without sending content.       |

### Main Commands

| Command         | Description                                      |
| --------------- | ------------------------------------------------ |
| `scat post`     | Posts a text message.                            |
| `scat upload`   | Uploads a file.                                  |
| `scat export`   | Exports data, such as channel logs.              |
| `scat profile`  | Manages configuration profiles.                  |
| `scat config`   | Manages the configuration file itself.           |
| `scat channel`  | Manages channels for supported providers.        |

### `post` Command Flags

| Flag          | Shorthand | Description                               |
| ------------- | --------- | ----------------------------------------- |
| `--channel`   | `-c`      | Override destination channel (cannot be used with `--user`). |
| `--user`      |           | Send a direct message to a user by ID or mention name. |
| `--from-file` |           | Read message body from a file.            |
| `--stream`    | `-s`      | Stream messages from stdin continuously.  |
| `--tee`       | `-t`      | Print stdin to screen while posting.      |
| `--username`  | `-u`      | Override the username for this post.      |
| `--iconemoji` | `-i`      | Icon emoji to use (Slack provider only).  |
| `--format`    |           | Message format (`text` or `blocks`). Default is `text`. |

### `upload` Command Flags

| Flag        | Shorthand | Description                                      |
| ----------- | --------- | ------------------------------------------------ |
| `--channel` | `-c`      | Override destination channel (cannot be used with `--user`). |
| `--user`    |           | Send a direct message to a user by ID or mention name. |
| `--file`    | `-f`      | **Required.** Path to the file, or `-` for stdin. |
| `--filename`| `-n`      | Filename for the upload.                         |
| `--filetype`|           | Filetype for syntax highlighting (e.g., `go`).   |
| `--comment` | `-m`      | A comment to post with the file.                 |

### `export log` Command Flags

| Flag            | Shorthand | Description                                      |
| --------------- | --------- | ------------------------------------------------ |
| `--channel`     | `-c`      | **Required.** Channel to export from.            |
| `--output`      |           | Output file path for the log. Use `-` for stdout (default). |
| `--output-files`|           | Directory to save downloaded files. If set to `auto`, a directory is auto-generated. |
| `--output-format` |         | Output format (`json` or `text`).                |
| `--start-time`  |           | Start of time range (RFC3339 format).            |
| `--end-time`    |           | End of time range (RFC3339 format).              |

### `profile` Subcommands

| Subcommand | Description                                      |
| ---------- | ------------------------------------------------ |
| `list`     | List all available profiles.                     |
| `use`      | Set the active profile.                          |
| `add`      | Add a new profile.                               |
| `set`      | Set a value in the current profile.              |
| `remove`   | Remove a profile.                                |

### `channel` Subcommands

| Subcommand | Description                                      |
| ---------- | ------------------------------------------------ |
| `list`     | Lists available channels for `slack` profiles.   |
| `create`   | Creates a new channel for `slack` profiles.      |

### `config` Subcommands

| Command             | Description                                      |
| ------------------- | ------------------------------------------------ |
| `config init`       | Creates a new default configuration file.        |

---

## Server Mode (Container / CI Deployment)

For server-side and containerized deployments, `scat` supports a **server mode** that reads all configuration from environment variables, eliminating the need for a config file on disk.

### Enabling Server Mode

Set the `SCAT_MODE=server` environment variable. All profile settings are then provided via environment variables:

| Variable | Required | Description |
| --- | --- | --- |
| `SCAT_MODE` | yes | Set to `server` to enable server mode. |
| `SCAT_PROVIDER` | yes | Provider name (e.g., `slack`). |
| `SCAT_TOKEN` | yes | Authentication token. |
| `SCAT_CHANNEL` | no | Default destination channel. |
| `SCAT_USERNAME` | no | Default display name. |

### Example

```bash
export SCAT_MODE=server
export SCAT_PROVIDER=slack
export SCAT_TOKEN=xoxb-xxxxxxxxxxxx
export SCAT_CHANNEL="#deploy-notify"

echo "Deployed v1.2.0" | scat post
```

### Kubernetes Example

Inject the token from a Kubernetes Secret — no config file or volume mount required:

```yaml
env:
  - name: SCAT_MODE
    value: "server"
  - name: SCAT_PROVIDER
    value: "slack"
  - name: SCAT_CHANNEL
    value: "#alerts"
  - name: SCAT_TOKEN
    valueFrom:
      secretKeyRef:
        name: slack-credentials
        key: token
```

### Restrictions in Server Mode

The following are not available in server mode and will return an error:

- `--config` flag (config file is ignored entirely)
- `--profile` flag (only the env-var profile is used)
- All `profile` subcommands (`add`, `use`, `list`, `set`, `remove`)
- `config init`

---

## Acknowledgements

This project is heavily inspired by and based on the concepts of [bcicen/slackcat](https://github.com/bcicen/slackcat). The core logic for handling file/stdin streaming and posting was re-implemented with reference to the original `slackcat` codebase. `slackcat` is also distributed under the MIT License.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.