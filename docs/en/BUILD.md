# Building and testing scat

[日本語](../ja/BUILD.ja.md)

Use Go **1.25 or later** and Make. The minimum supports anchored `os.Root`
rename operations for downloaded attachments. Build through Make, not a direct
`go build`, so binaries stay in `dist/` and version/signing flags are applied.

```sh
make fmt          # go fmt ./... with project-local caches
make check        # go vet ./... + go test ./...
make test GOFLAGS=-race
make build        # dist/scat
make build-all    # darwin/arm64, linux/amd64, linux/arm64, windows/amd64
make vulncheck    # requires govulncheck; accesses the vulnerability database
```

Caches are under `.cache/`. Tests inject HTTP transports, time and I/O; they need
no network listeners, Slack tokens or live mutations. Synthetic export fixtures
are tracked in `testdata/export/`. `internal/input` tests blocking-read cancellation.
The command suite covers retained v1 behavior (argument/file/stdin input, profile
lifecycle, service mode, rich posts, uploads, channel operations) using v2 outputs.
Legacy mock-provider log assertions are replaced by checks of API requests and
results. Transport tests cover both permitted and forbidden credential redirects,
HTML-at-200 errors, real HTML/JSON files, upload staging, one-shot completion,
pagination, rate limits, ambiguous failures and atomic output.

For local use, copy `dist/scat` into a directory on your PATH. There is no
`make install`, universal macOS binary or `bin/` output directory.

## Repository and release

This repository is the `scat` submodule of `nlink-jp/chatops-series`. Commit scat
changes here. After pushing, update the umbrella gitlink in a separate commit.
Do not replace the submodule with a directory or point the umbrella at an
unpublished commit. Keep other submodules untouched.

`make package` builds release archives and requests macOS notarization;
`make verify-release` checks the notarized archive and version before publication.
`make brew` generates the Homebrew formula. Follow the org release/signing policy;
a successful local ad-hoc build is not a notarized release. Live Slack checks and
publication remain separate gates; this renewal branch does not publish v2.

The static scan `gosec ./...` reports G119 on scoped Authorization re-attachment
to Slack sibling hosts. Following the knowledge entry, this finding is retained
with its justification and both allowed/forbidden regression test names in place.
