# Contributing to rivr

Thanks for your interest! rivr is a Go CLI built to the [Agent CLI
Guidelines](https://aclig.dev/) — contributions should preserve that contract (read-only by
default, structured errors + stable exit codes, bounded JSON output, prompt-injection fencing,
stdin/keyring secrets).

## Setup

```bash
git clone https://github.com/rnwolfe/rivr && cd rivr
go build ./...        # build
go vet ./...          # vet
go test ./...         # tests (incl. the schema-snapshot gate)
go run ./cmd/rivr --help
```

Go 1.25+. No CGO required (`CGO_ENABLED=0` builds statically).

## Tests

- Contract tests + provider fixtures live alongside the code; run `go test ./...`.
- The **schema snapshot is a CI gate** — if you intentionally change the command tree, flags,
  exit codes, or providers, regenerate it:
  `RIVR_UPDATE_GOLDEN=1 go test ./internal/cli/...` and review the diff.
- Provider changes should include an `httptest` fixture (see `internal/provider/*_test.go`).

## Conventional Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) — they drive the changelog
and version bumps. Examples: `feat(providers): …`, `fix(auth): …`, `docs: …`, `chore: …`.

Sign off your commits (DCO): `git commit -s`. There is no CLA.

## Pull requests

- Keep the contract intact; if you add a command/flag, update `schema`, the embedded
  `SKILL.md`, the docs, and the landing copy in the **same PR** (see `AGENTS.md`).
- Output fields are **append-only** — renames/removals are breaking and need discussion.
- `go build ./... && go vet ./... && go test ./...` must pass.
- Don't commit build artifacts, `dist/`, `node_modules/`, or `.vercel/`.

## Adding a provider backend

Implement the `provider.Provider` interface (+ optionally `Validator`, `Describable`,
`TagAware`), register it in `internal/provider/registry.go`, and add a `Describe()` profile so
it shows up in `rivr provider list` and the backend matrix. Add an `httptest` fixture.
