# Contributing

## Build and test

```sh
go build ./...
go generate ./...    # regenerates the commands and the reference; commit what it changes
go vet ./...
go test -race -count=1 ./...
```

CI (`.github/workflows/test.yml`) runs the same, then `golangci-lint`, a GoReleaser snapshot build and
the install-script tests in `scripts/`. It fails when `go generate ./...` changes, adds or removes a file.

## Where commands come from

Most commands are generated from the API's `openapi.json`, which ships in the pinned
`github.com/CloudAfrica/cloudafrica-go` module (see `go.mod`). `internal/cli/commands.yaml` is the overlay
that shapes them: command names, groups, flags and table columns. `go generate ./...` runs
`internal/gen/cagen`, which writes `internal/cli/commands_gen.go`.

Every operation in `openapi.json` must be one of:

- **generated**: the default, shaped by `operations:` and `columns:` in the overlay;
- **hand-written** (`handwritten:`): a command written in Go, such as `vm create`;
- **excluded** (`exclude:`), with a reason; `ca api` still reaches it.

Generation fails for an operation it cannot generate, so a new API operation can't be missed. To pick
up API changes, bump `cloudafrica-go` and run `go generate ./...`.

`internal/gen/docgen` writes `docs/reference/` from the same command tree; never edit it by hand. The
hand-written pages in `docs/` group commands by domain.

## Writes that must not be retried

The SDK retries a 503. Where an operation's description says a retry after a 5xx can repeat the write
(look for "second", "duplicate" or "5xx"), add its operation id to `unrepeatable` in
`internal/cli/unrepeatable.go` with the hint to print, so `ca` returns the 503 instead of retrying.

## Before you open a PR

Read [`docs/traps.md`](docs/traps.md): mistakes review has caught before.

## Releasing

Push a `v*` tag. `.github/workflows/release.yml` runs GoReleaser, which publishes the archives,
`SHA256SUMS` and its cosign bundle, the install scripts, the Homebrew formula
([homebrew-tap](https://github.com/CloudAfrica/homebrew-tap)) and the Scoop manifest
([scoop-bucket](https://github.com/CloudAfrica/scoop-bucket)). A prerelease tag (`v0.8.0-rc.1`) skips
Homebrew and Scoop.
