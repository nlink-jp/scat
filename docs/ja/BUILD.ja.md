# scatのビルドとテスト

[English](../en/BUILD.md)

Go **1.25以降**とMakeを使います。この最低版は、添付保存の `os.Root` による固定ディレクトリ内の
renameをサポートします。直接 `go build` せずMakeを使い、出力先 `dist/` とversion・署名設定を維持します。

```sh
make fmt          # プロジェクト内cacheで go fmt ./...
make check        # go vet ./... + go test ./...
make test GOFLAGS=-race
make build        # dist/scat
make build-all    # darwin/arm64, linux/amd64, linux/arm64, windows/amd64
make vulncheck    # govulncheckが必要。脆弱性DBへアクセス
```

cacheは `.cache/` 配下です。通常テストではHTTP transport、時刻、I/Oを注入し、
network listener・Slack token・実際の変更操作を必要としません。
合成export fixtureを `testdata/export/` に追跡し、`internal/input` では入力待ちのキャンセルを検証します。
CLIテストは維持するv1の操作（引数/file/stdin入力、profile管理、server mode、rich post、upload、channel操作）を
v2の出力で確認します。旧mock providerのログ検査を、API要求と結果の検査へ置き換えました。
通信テストでは認証redirectの許可・拒否、HTML 200エラー、正規HTML/JSON、upload snapshot、
一度だけの完了、pagination、rate limit、結果不明の失敗、atomic出力を検証します。

ローカル利用では `dist/scat` をPATH上のディレクトリへコピーしてください。
`make install`、macOS universal binary、`bin/` 出力はありません。

## リポジトリとリリース

本リポジトリは `nlink-jp/chatops-series` の `scat` submoduleです。
scatの変更はここでコミットし、push後にアンブレラ側のgitlinkを別コミットで更新します。
submoduleを通常ディレクトリに置き換えたり、未公開commitをアンブレラから参照したりしません。
他のsubmoduleは変更しません。

`make package` は全環境の配布archiveにbinary・README・LICENSEと
`slack-app-manifest.json` を同梱し、macOS notarizationを要求します。
`make verify-release` は公開前にnotarized archiveとversionを検証します。
`make brew` はHomebrew formulaを生成します。組織のrelease・署名規約に従ってください。
ローカルのad-hocビルド成功はnotarized releaseを意味しません。
実Slack検証と公開は別のリリースgateに従います。

静的検査の `gosec ./...` は、Slack兄弟hostに限定したAuthorization再付与にG119を報告します。
ナレッジの指示に従い、この警告は抑制せず、該当箇所に理由と許可・拒否両側のテスト名を記載しています。

## 実Slack E2E

検証用botが参加済みの専用チャンネルと、そのbotをcurrent profileに持つv2設定を使います。
本番チャンネルは使いません。public channelでは `chat:write`、`channels:read`、
`channels:history`、`files:write`、`files:read` が必要です。private channelでは対応scopeを使います。
設定にカスタムusernameがある場合は `chat:write.customize` も必要です。

```sh
SCAT_E2E_CONFIG=/absolute/path/to/test-config.json \
SCAT_E2E_CHANNEL=C0123456789 make e2e
```

`make e2e` は `dist/scat` をビルドし、`go test -tags=e2e -count=1` を実行します。
認証情報・対象channel・binaryが不足すれば失敗し、実検証を黙ってskipしません。
一意の識別子で親投稿・返信・stream投稿とbinary/HTML/JSONファイルを作り、export schemaと
ダウンロードバイトの完全一致、親の排他的期間外にある返信の保持、server modeのtee、
無効な認証情報での失敗を確認します。作成したファイル・投稿を削除し、cleanup失敗もテスト失敗にします。
設定は読み取るだけで、tokenを引数に渡しません。実export・認証情報・環境固有の結果をコミットしません。

このsuiteはpost/upload/exportの往復を対象にします。channel作成・招待・DMと、
人工的なrate limit・redirect・転送失敗は別ケースであり、通常suiteの注入テストで要求と失敗処理を検証します。
