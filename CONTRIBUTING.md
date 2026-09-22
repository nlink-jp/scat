# Contributing to scat

Follow the [organization conventions](https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md)
and [AGENTS.md](AGENTS.md). scat is a Slack-only bot CLI; scli owns human/user
credentials. Read [ADR-0001](docs/en/adr/0001-slack-bot-renewal.md) before changing
this boundary or the export contract.

1. For maintainer work in this umbrella submodule, check out `main` before
   changes, following the organization workflow. Do not turn the submodule into
   an ordinary directory. External contributors may submit pull requests;
   independent review does not itself require a PR.
2. Write or update tests with behavior changes. Use `cmd.NewCommand(Dependencies)`
   and injected HTTP RoundTrippers; constructors must not perform I/O. Do not add
   runtime mock providers, global test hooks, or unchecked context assertions.
3. Run `make fmt`, `make check`, `make test GOFLAGS=-race`, `make build`,
   `make build-all`, and `make vulncheck`. See [BUILD](docs/en/BUILD.md).
4. Keep README.md and README.ja.md, paired detailed docs, CHANGELOG and AGENTS in
   sync with the implementation. Pin adopted export behavior with synthetic
   fixtures; never copy real workspace messages or credentials into tests.
5. Use small typed commits (`feat:`, `fix:`, `test:`, `docs:`, `chore:`). Run the
   tests before committing. Obtain an independent implementation review.
6. Push scat before updating the umbrella's `scat` gitlink in a separate commit.
   Feed verified reusable lessons back to knowledge and run `check-org.sh` after
   integration. Follow the separate release checklist when publishing a version.

Report bugs with the version, OS, exact command (secrets removed), expected and
actual behavior. Safe `--debug` output may help; do not post raw tokens, signed
upload URLs or private message bodies in issues.

Live Slack posting/upload/channel/invitation tests require an explicitly
permitted fixture workspace. The offline suite makes no live Slack calls.
