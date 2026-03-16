# scat: 汎用コマンドラインコンテンツ投稿ツール

`scat` is a versatile command-line interface for sending content from files or standard input to a configured destination, such as Slack. It is inspired by `slackcat` but is designed to be more generic and extensible.

---

## 主な機能

- **テキストメッセージの投稿**: 引数、ファイル、標準入力からコンテンツを送信します。
- **ファイルのアップロード**: 指定したパスや標準入力からファイルをアップロードします。
- **コンテンツのストリーミング**: 標準入力を継続的に監視し、定期的にメッセージを投稿します。
- **チャネルログのエクスポート**: チャネルのメッセージ履歴を構造化されたJSONファイルや標準出力に出力します。
- **プロファイル管理**: 複数の宛先を設定し、簡単に切り替えることができます。
- **拡張可能なプロバイダ**: 現在、Slackとテスト用のモックプロバイダをサポートしています。

## インストール

[リリースページ](https://github.com/magifd2/scat/releases)から、お使いのシステム用の最新のバイナリをダウンロードしてください。

または、ソースからビルドすることも可能です:

```bash
make build
```

## 初期セットアップ

投稿を開始する前に、設定ファイルを作成する必要があります。

1.  **設定ファイルの初期化**:

    以下のコマンドを実行して、デフォルトの場所に設定ファイル (`~/.config/scat/config.json`) を作成します:

    ```bash
    scat config init
    ```

    **重要**: この設定ファイルには、Slackトークンなどの機密情報が含まれます。セキュリティのため、ファイルパーミッションを `600` (所有者のみ読み書き可能) に設定することを強く推奨します。

2.  **プロファイルの設定**:

    デフォルトのプロファイルは、テストに便利なモックプロバイダを使用します。Slackのような実際のサービスに投稿するには、新しいプロファイルを追加する必要があります。

    Slackプロファイル設定の詳細な手順については、**[Slackセットアップガイド](./docs/SLACK_SETUP.md)** を参照してください。

    以下に、新しいSlackプロファイルを簡単に追加する例を示します:

    ```bash
    # このコマンドを実行すると、Slack Botトークンを安全に入力するよう求められます。
    scat profile add my-slack-workspace --provider slack --channel "#general"
    ```

3.  **アクティブプロファイルの設定**:

    `scat` がデフォルトで使用するプロファイルを指定します:

    ```bash
    scat profile use my-slack-workspace
    ```

## 使用例

`scat` の一般的な使い方をいくつか紹介します。

### テキストメッセージの投稿 (`post`)

-   **引数から投稿**:
    `scat post "コマンドラインからこんにちは！"`

-   **標準入力から (パイプ)**:
    `echo "このメッセージはパイプされました。" | scat post`

### Block Kit メッセージの投稿 (`post` と `--format blocks`)

-   **引数から (JSON文字列)**:
    `scat post --format blocks '[{"type": "section", "text": {"type": "mrkdwn", "text": "引数からBlock Kit！"}}]'`

-   **ファイルから (JSONファイル)**:
    (Block Kit JSONコンテンツを含む `blocks.json` というファイルを作成してください)
    `scat post --format blocks --from-file ./blocks.json`

-   **標準入力から (JSONパイプ)**:
    `echo '[{"type": "section", "text": {"type": "mrkdwn", "text": "標準入力からBlock Kit！"}}]' | scat post --format blocks`

### ファイルのアップロード (`upload`)

-   **パスを指定してファイルをアップロード**:
    `scat upload --file ./report.pdf`

-   **コメント付きでアップロード**:
    `scat upload --file ./screenshot.png -m "こちらがご依頼のスクリーンショットです。"`

### チャネルログのエクスポート (`export log`)

-   **標準出力にエクスポートし、`jq`にパイプする**:
    `scat export log --channel "#random" | jq .`

-   **指定したファイルにエクスポートする**:
    `scat export log -c "#random" --output "my-export.json"`

-   **添付ファイルを自動生成されたディレクトリに保存する**:
    `scat export log -c "#random" --output-files auto`

-   **ログは標準出力、添付ファイルは指定ディレクトリに保存する**:
    `scat export log -c "#random" --output - --output-files "./attachments"`

## コマンドリファレンス

### グローバルフラグ

| フラグ             | 説明                                           |
| ------------------ | ---------------------------------------------- |
| `--config <path>`  | 設定ファイルの代替パスを指定します。サーバーモードでは使用できません。 |
| `--profile <name>` | コマンドの実行に特定のプロファイルを使用します。 |
| `--debug`          | 詳細なデバッグログを有効にします。               |
| `--silent`         | 成功メッセージを抑制します。                   |
| `--noop`           | コンテンツを送信しないドライランを実行します。   |

### メインコマンド

| コマンド        | 説明                                           |
| --------------- | ---------------------------------------------- |
| `scat post`     | テキストメッセージを投稿します。                 |
| `scat upload`   | ファイルをアップロードします。                   |
| `scat export`   | チャネルログなどのデータをエクスポートします。   |
| `scat profile`  | 設定プロファイルを管理します。                   |
| `scat config`   | 設定ファイル自体を管理します。                   |
| `scat channel`  | 対応プロバイダのチャンネルを管理します。         |

### `post` コマンドのフラグ

| フラグ          | 短縮形 | 説明                                           |
| --------------- | ------ | ---------------------------------------------- |
| `--channel`   | `-c`   | この投稿の宛先チャンネル (IDまたは名前) を上書きします。 |
| `--from-file` |        | メッセージ本文をファイルから読み込みます。       |
| `--stream`    | `-s`   | 標準入力からメッセージを継続的にストリームします。|
| `--tee`       | `-t`   | 投稿前に標準入力の内容を画面に出力します。     |
| `--username`  | `-u`   | この投稿のユーザー名を上書きします。             |
| `--iconemoji` | `-i`   | 使用するアイコン絵文字 (Slackプロバイダのみ)。   |
| `--format`    |        | メッセージのフォーマット (`text` または `blocks`)。デフォルトは `text`。 |

### `upload` コマンドのフラグ

| フラグ        | 短縮形 | 説明                                                     |
| ----------- | ------ | -------------------------------------------------------- |
| `--channel` | `-c`   | このアップロードの宛先チャンネル (IDまたは名前) を上書きします。 |
| `--file`    | `-f`   | **必須。** アップロードするファイルのパス、または `-` で標準入力。|
| `--filename`| `-n`   | アップロード時のファイル名。                             |
| `--filetype`|        | 構文ハイライト用のファイルタイプ (例: `go`)。            |
| `--comment` | `-m`   | ファイルと一緒に投稿するコメント。                       |

### `export log` コマンドのフラグ

| フラグ            | 短縮形 | 説明                                                     |
| --------------- | ------ | -------------------------------------------------------- |
| `--channel`     | `-c`   | **必須。** エクスポート元のチャネル。                    |
| `--output`      |        | ログの出力ファイルパス。`-`で標準出力（デフォルト）。     |
| `--output-files`|        | 添付ファイルの保存先。`auto`でディレクトリを自動生成。未指定時はダウンロードしない。 |
| `--output-format` |      | 出力フォーマット (`json` または `text`)。                |
| `--start-time`  |        | 時間範囲の開始 (RFC3339フォーマット)。                   |
| `--end-time`    |        | 時間範囲の終了 (RFC3339フォーマット)。                   |

### `profile` サブコマンド

| サブコマンド | 説明                                           |
| ---------- | ---------------------------------------------- |
| `list`     | 利用可能なすべてのプロファイルを表示します。     |
| `use`      | アクティブなプロファイルを切り替えます。         |
| `add`      | 新しいプロファイルを追加します。                 |
| `set`      | 現在のプロファイルの設定値を変更します。         |
| `remove`   | プロファイルを削除します。                       |

### `channel` サブコマンド

| サブコマンド | 説明                                                     |
| ---------- | -------------------------------------------------------- |
| `list`     | `slack` プロファイルで利用可能なチャンネルを一覧表示します。|
| `create`   | `slack` プロファイルに新しいチャンネルを作成します。       |

### `config` サブコマンド

| コマンド             | 説明                                           |
| ------------------- | ---------------------------------------------- |
| `config init`       | 新しいデフォルト設定ファイルを作成します。       |

---

## サーバーモード（コンテナ / CI デプロイ）

サーバーサイドやコンテナ化された環境向けに、`scat` は**サーバーモード**をサポートしています。このモードでは、設定ファイルを使用せず、すべての設定を環境変数から読み込みます。

### サーバーモードの有効化

`SCAT_MODE=server` 環境変数を設定します。プロファイルの設定は以下の環境変数で指定します:

| 変数 | 必須 | 説明 |
| --- | --- | --- |
| `SCAT_MODE` | はい | `server` に設定するとサーバーモードが有効になります。 |
| `SCAT_PROVIDER` | はい | プロバイダ名 (例: `slack`)。 |
| `SCAT_TOKEN` | はい | 認証トークン。 |
| `SCAT_CHANNEL` | いいえ | デフォルトの送信先チャンネル。 |
| `SCAT_USERNAME` | いいえ | デフォルトの表示名。 |
| `SCAT_MAX_FILE_SIZE` | いいえ | アップロードファイルの最大サイズ（バイト、デフォルト: 1073741824 = 1 GB）。 |
| `SCAT_MAX_STDIN_SIZE` | いいえ | 標準入力の最大読み込みサイズ（バイト、デフォルト: 10485760 = 10 MB）。 |

### 使用例

```bash
export SCAT_MODE=server
export SCAT_PROVIDER=slack
export SCAT_TOKEN=xoxb-xxxxxxxxxxxx
export SCAT_CHANNEL="#deploy-notify"

echo "Deployed v1.2.0" | scat post
```

### Kubernetes での使用例

Kubernetes Secret からトークンを注入することで、設定ファイルもボリュームマウントも不要です:

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

### サーバーモードでの制約

サーバーモードでは以下の操作は利用できず、エラーが返されます:

- `--config` フラグ（設定ファイルは完全に無視されます）
- `--profile` フラグ（環境変数で設定されたプロファイルのみ使用されます）
- すべての `profile` サブコマンド（`add`, `use`, `list`, `set`, `remove`）
- `config init`

---

## 謝辞 (Acknowledgements)

このプロジェクトは、[bcicen/slackcat](https://github.com/bcicen/slackcat) のコンセプトに強くインスパイアされ、またその影響を受けています。ファイルや標準入力のストリーミングを処理し投稿するコアロジックは、オリジナルの `slackcat` のコードベースを参考に再実装されました。`slackcat` も同じくMITライセンスで配布されています。

## ライセンス (License)

このプロジェクトはMITライセンスの下で公開されています。詳細は [LICENSE](LICENSE) ファイルをご覧ください。