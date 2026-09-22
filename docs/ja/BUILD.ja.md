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

cacheは `.cache/` 配下です。テストではHTTP transport、時刻、I/Oを注入し、
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

`make package` はrelease archiveを作成してmacOS notarizationを要求し、
`make verify-release` は公開前にnotarized archiveとversionを検証します。
`make brew` はHomebrew formulaを生成します。組織のrelease・署名規約に従ってください。
ローカルのad-hocビルド成功はnotarized releaseを意味しません。
実Slack検証と公開は別のgateで、この刷新ブランチではv2を公開しません。

静的検査の `gosec ./...` は、Slack兄弟hostに限定したAuthorization再付与にG119を報告します。
ナレッジの指示に従い、この警告は抑制せず、該当箇所に理由と許可・拒否両側のテスト名を記載しています。
