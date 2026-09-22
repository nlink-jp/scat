# AGENTS.md — scat

## Project

scat is the service-facing, bot-authenticated Slack tool in chatops-series.
The v2 implementation removes the generic provider layer and follows accepted
[ADR-0001](docs/en/adr/0001-slack-bot-renewal.md)
([日本語](docs/ja/adr/0001-slack-bot-renewal.ja.md)). scli remains the user-authenticated sibling. Release status comes from Git tags,
not prose banners.

Follow the [organization conventions](https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md)
and [CONTRIBUTING.md](CONTRIBUTING.md). The renewal's product direction is agreed;
its detailed implementation contract was approved on 2026-09-22.

## Repository boundary

This checkout is the `scat` submodule of `nlink-jp/chatops-series`, declared in
the umbrella's `.gitmodules`. Check out `main` before changing the submodule,
as required by the organization. Commit scat source and documentation in this
repository. After pushing the scat commit, update and commit the `scat` gitlink
in the umbrella separately; never record an unpublished commit there. Preserve
the submodule registration and verify both repositories' status before and after
integration. Run the organization check after the pointer update.

## Build and validation

- `make build`: current-platform binary in `dist/scat`; never invoke `go build`
  directly. The Makefile owns version injection and signing.
- `make check`: `go vet ./...` and `go test ./...`.
- `make test`: full Go suite; `make test GOFLAGS=-race`: race detector. Tests
  inject `http.RoundTripper` and require no listeners or real Slack credentials.
- `make e2e`: builds the CLI and runs uncached live Slack round trips; requires
  `SCAT_E2E_CONFIG` and `SCAT_E2E_CHANNEL` for a dedicated bot/test channel.
  Missing settings fail, and cleanup deletes only this run's messages/files.
- `make fmt`: format with project-local caches.
- `make vulncheck`: reachable-vulnerability scan; needs govulncheck and network.
- Go 1.25+ is required for `os.Root.Rename` (anchored attachment replacement).
- `make build-all`: darwin/arm64, linux/amd64, linux/arm64, windows/amd64.
- `make package`, `make verify-release`, `make brew`: packaging, public macOS
  signing/notarization gate, and Homebrew publication; follow the org checklist.
- Caches live in `.cache/`; build artifacts and caches are ignored by Git.

## Current structure

- `main.go`: entry point; module `github.com/nlink-jp/scat`.
- `cmd/`: Cobra commands and invocation-scoped dependencies/configuration.
- `internal/config/`: bot profiles, limits and environment-only service mode.
- `internal/slack/`: bot identity, API calls, lazy resolution, posting and files.
- `internal/export/`: scli-compatible model, selection, rendering, atomic output.
- `internal/input/`: cancelable blocking input adapter; the CLI owns one invocation.
- `e2e/`: opt-in real Slack binary tests (`-tags=e2e`), including cleanup.
- `testdata/export/`: synthetic reference JSON and golden output.
- `docs/en/`, `docs/ja/`: paired documentation; `docs/{en,ja}/adr/`: design decisions.
- `scripts/`: vendored release tooling.

## Gotchas and renewal constraints

- Do not remove multi-workspace bot profiles when removing multiple providers.
- For renewal export use scli `854e6a0` as the recorded baseline, including parent
  followed by replies, original text, rich fields and file metadata. Read the ADR
  for the parent-selection interval and intended corrections such as deduplication.
- scli's serialized `local_path` is always present; its prose says optional.
  The renewal follows the actual output type and pins the distinction in fixtures.
- Keep user credentials out of scat. No fallback to scli config or generic
  `SLACK_TOKEN`; remote work must verify the bot identity as the ADR specifies.
- Preserve API permission errors; don't report incomplete message retrieval as
  successful export. Optional enrichment/download failures have separate rules.
- Preserve authenticated-download redirect containment. A copied HTTP client
  alone is not proof that Authorization stays inside the allowed domain.
- Live Slack can label HTML/JSON metadata as text/plain and serve force-download.
  MIME mismatch acceptance requires authenticated Slack attachment identity and
  recorded-size verification; preserve login and foreign-host refusal tests.
- File upload has separate allocation, byte-transfer and completion stages.
  Stage a bounded snapshot; never finalize a failed transfer or replay one-shot
  completion. See the ADR's file-transfer acceptance cases and knowledge links.
- HTTP 200 can contain an unauthenticated HTML login page. Test this alongside
  legitimate HTML attachments and authenticated Slack sibling-host redirects.
- Runtime providers, capabilities and appcontext are removed. Retained command
  behaviors are tested through actual requests/results rather than provider logs.
- End bounds with sub-microsecond fractions round upward; start bounds round
  downward. Exact microsecond boundaries remain exclusive.
- Missing membership/privacy fields are unknown, not evidence of a public channel.
- Gosec G119 at the download redirect is retained and justified in place; tests
  cover both authenticated sibling hosts and credential-free outside hosts.
- Blocking input cancellation returns promptly. The source owner must close an
  abandoned read, or the one-shot CLI exits; never reuse such stdin in a REPL.
- Read reference projects' CLAUDE/AGENTS and mechanism-specific knowledge before
  porting. Independently review design and implementation as required by the org.
- Keep README.md / README.ja.md, related docs, CHANGELOG and this file in sync.
- Live verification uses an authorized dedicated test channel and bot configuration.
  Do not treat offline success or skipped live tests as completed E2E.
  Keep workspace IDs, tokens and raw live exports out of tracked artifacts.
