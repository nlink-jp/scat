# 開発計画: チャンネル管理機能

> 過去のv1計画です。現在の仕様は [ADR-0001](adr/0001-slack-bot-renewal.ja.md) と [README](../../README.ja.md) を参照してください。

このドキュメントでは、`scat` に新しいチャンネル作成機能とユーザー招待機能を実装するための開発計画を概説します。

## 1. 機能

- **`scat channel create`**: パブリックまたはプライベートのSlackチャンネルを作成する新しいコマンド。
  - 説明（description）とトピックの設定をサポートします。
  - 作成時のユーザーおよびユーザーグループの招待をサポートします。
- ユーザーの指定は、ユーザーID（例: `U12345`）とメンション名（例: `@username`）の両方をサポートします。
  - 注: `scat channel invite` という独立したコマンドは実装しませんでした。招待は `scat channel create` が直接処理します。

## 2. 設計方針

- 既存のパターン（`Post`、`ExportLog`）との一貫性を保つため、プロバイダーメソッドのパラメータは必須のものも含めてすべて、単一の `Options` 構造体を介して渡します。
- 実装は階層化し、低レベルなAPI呼び出しを高レベルなプロバイダーロジックおよびCLIコマンドのロジックから分離します。

## 3. 開発手順

### ステップ1: Capabilities定義の追加
- **ファイル:** `internal/provider/provider.go`
- **作業:** `Capabilities` 構造体に `CanCreateChannel` の真偽値フラグを追加する。（注: 招待は `CreateChannelOptions` の中で処理されるため、`CanInviteUsers` は独立したcapabilityとしては追加しませんでした。）

### ステップ2: インターフェースと型の定義
- **ファイル:** `internal/provider/types.go`
- **作業:** `CreateChannelOptions` 構造体を定義する。（注: `InviteToChannelOptions` は独立した構造体としては定義しませんでした。）
- **ファイル:** `internal/provider/provider.go`
- **作業:** `provider.Interface` に以下のメソッドを追加する。
  - `CreateChannel(opts CreateChannelOptions) (string, error)`（注: `Channel` 構造体ではなく、チャンネルIDの文字列を返します。`InviteToChannel` は独立したメソッドとしては追加しませんでした。）

### ステップ3: すべてのプロバイダー実装の更新
- **`internal/provider/slack/`**:
  - **`api.go`**: Slack APIの `conversations.create` と `conversations.invite` を呼び出す、新しい低レベル関数を追加しました。
  - **`types.go`**: 新しいAPIのレスポンスをアンマーシャルするための構造体を追加しました。
  - **`slack_provider.go`**: 新機能について `true` を返すよう `Capabilities()` メソッドを更新しました。
  - **`channel.go`（または類似のファイル）**: `api.go` の新しい関数を呼び出す `CreateChannel` メソッドを実装しました。（注: `InviteToChannel` メソッドは個別には実装しませんでした。）
- **`internal/provider/mock/mock_provider.go`**: 新しいインターフェースメソッドのダミー実装を追加しました。
- **`internal/provider/testprovider/test_provider.go`**: 新しいインターフェースメソッドについて、状態を保持するインメモリの偽実装を追加しました。

### ステップ4: CLIコマンドの実装
- **ファイル:** `cmd/channel_create.go` を作成しました。（注: `cmd/channel_invite.go` は独立したコマンドとしては作成していません。招待は `scat channel create` が処理します。）
- **ファイル:** `cmd/channel.go`
- **作業:**
  - 新しいコマンドを登録しました。
  - コマンドのロジック:
    1. 最初にプロバイダーのcapabilitiesを確認する。
    2. `ResolveUserID` および `ResolveUserGroupID` の機能を用いて、ユーザーのメンション名をIDに解決する。
    3. コマンドライン引数とフラグから適切な `Options` 構造体を構築する。
    4. 対応するプロバイダーメソッドを呼び出す。

### ステップ5: テストの実装
- **プロバイダーのテスト（`internal/provider/slack/*_test.go`）:**
  - `net/http/httptest` を使用してモックHTTPサーバーを作成しました。
  - プロバイダーのメソッドがSlack APIへ正しいリクエストを送信し、レスポンスを正しく処理することを検証するユニットテストを記述しました。
- **CMDのテスト（`cmd/*_test.go`）:**
  - `test_provider` を使用してCLIコマンドをテストしました。
  - コマンドのフラグが正しく解析されること、および `test_provider` のインメモリ状態が期待どおりに更新されることを検証しました。

### ステップ6: ドキュメントの更新
- **`README.md`、`README.ja.md`**: 新しいコマンドの使用方法を追記する。（注: この手順は未完了であり、別途対応が必要です。）
- **`docs/en/SLACK_SETUP.md`、`docs/ja/SLACK_SETUP.ja.md`**: 新たに必要となったOAuthスコープをセットアップ手順に追加しました。
  - `channels:manage`
  - `groups:write`
  - `usergroups:read`（注: このスコープは実装中に追加されました。）
