# Contributing to ytcap

Thanks for your interest! This document covers how to set up a dev environment, run tests, and submit changes.

## Development setup

Prerequisites: [mise](https://mise.jdx.dev/) (recommended) or Go 1.24+ manually.

```sh
git clone https://github.com/yetmike/ytcap
cd ytcap
mise install          # installs Go, golangci-lint, goreleaser, node
make build            # builds ./ytcap
```

You also need `yt-dlp` and `summarize` on your `PATH` to actually run the tool — see the README.

## Common commands

```sh
make build            # build the binary
make run ARGS="..."   # run with arguments
make dev ARGS="..."   # run with YTCAP_LOG=debug
make test             # go test ./...
make lint             # golangci-lint run ./...
```

## Submitting changes

1. Open an issue first for anything non-trivial so we can agree on the approach.
2. Fork and create a topic branch.
3. Make your change. Keep commits focused — one logical change per commit.
4. Run `make test` and `make lint` before pushing.
5. Open a PR with a clear description of *why* the change is needed.

## Code style

- Standard Go formatting (`gofmt` / `goimports`).
- `golangci-lint` must pass.
- No new dependencies without a good reason.
- Keep the TUI keyboard-driven; avoid mouse-only interactions.

## Reporting bugs

File an issue with:

- `ytcap version` output
- `yt-dlp --version` and `summarize --version`
- Steps to reproduce
- Relevant lines from `~/.ytcap/ytcap.log` (or rerun with `YTCAP_LOG=debug`)
