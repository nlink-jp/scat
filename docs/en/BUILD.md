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

Caches are under `.cache/`. Default tests inject HTTP transports, time and I/O; they need
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

`make package` builds the release archives — the canonical binary, `README.md`
and `LICENSE`, per the org Release Archive Standard — and requests macOS notarization;
`make verify-release` checks the notarized archive and version before publication.
`make brew` generates the Homebrew formula. Follow the org release/signing policy;
a successful local ad-hoc build is not a notarized release. Live Slack checks and
publication follow the separate release gates.

The static scan `gosec ./...` reports G119 on scoped Authorization re-attachment
to Slack sibling hosts. Following the knowledge entry, this finding is retained
with its justification and both allowed/forbidden regression test names in place.

## Live Slack E2E

Use a dedicated channel that the test bot has already joined and a v2 config whose
current profile holds that bot. Do not use a production channel. The test requires
`chat:write`, `channels:read`, `channels:history`, `files:write` and `files:read`
for a public channel (use the corresponding private-channel scopes when needed).
A configured custom username also needs `chat:write.customize`.

```sh
SCAT_E2E_CONFIG=/absolute/path/to/test-config.json \
SCAT_E2E_CHANNEL=C0123456789 make e2e
```

`make e2e` first builds `dist/scat`, then runs `go test -tags=e2e -count=1`.
It fails when credentials, channel or binary are missing; it does not silently
skip live validation. It creates uniquely marked root/reply/stream messages and
binary, HTML and JSON files, verifies export schema and exact downloaded bytes,
checks the exclusive parent interval retains later replies, exercises server-mode
tee and invalid credentials, and deletes this run's files/messages. Cleanup
failures fail the test. Configuration is read only; tokens are never arguments.
Do not commit raw live exports, credentials or workspace-specific reports.

This suite covers the post/upload/export round trip. Channel creation, invitations,
DMs and artificial rate-limit/redirect/transfer failures remain separate cases;
the default suite covers their injected requests and failure handling.
