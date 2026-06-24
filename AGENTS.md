# AGENTS.md — amzn

Agent-focused CLI for **Amazon Shopping search and data retrieval**. Read-only; pluggable
provider backend. Built to the agent-CLI contract (see `spec.md`).

## Build / test / run
```bash
go build ./...                       # build
go vet ./...                         # vet
go test ./...                        # contract tests (incl. schema-snapshot gate)
go run ./cmd/amzn search "usb-c cable" --json
AMZN_UPDATE_GOLDEN=1 go test ./internal/cli/...   # regen schema golden after an INTENDED change
```

## Layout
- `cmd/amzn/main.go` — only `os.Exit(cli.Run(...))`; all logic in `internal/cli`.
- `internal/cli/` — kong grammar (`root.go`), one file per noun (`search.go`, `item.go`,
  `reviews.go`, `variations.go`, `browse.go`, `provider.go`), `misc.go` (auth/doctor/schema/
  agent/version), `suggest.go` ("did you mean").
- `internal/provider/` — **the normalized output schema + the pluggable backend interface**.
  `stub.go` is a placeholder; `cli-implement` adds real backends (serpapi/rainforest/creators)
  and wires keyring auth, then points `DefaultName()` at `serpapi`.
- `internal/output/` — output contract (stdout/stderr split, `--format`, `--select`, `--limit`).
- `internal/errs/` — stable exit-code table + structured `CLIError`.
- `internal/fence/` — untrusted-text fencing (contract §8).
- `internal/skill/SKILL.md` — embedded agent contract (printed by `amzn agent`).

## Conventions
- **Read-only**: no command mutates. The `--allow-mutations`/`--dry-run`/`--yes`/`--force`
  flags exist for contract uniformity but are no-ops; `Runtime.Guard` stays default-deny.
- **stdout = data, stderr = chatter.** Never `fmt.Println` to stdout — use `output.Writer`.
- **Output schema is append-only** (`internal/provider` types). Field rename/removal = a
  reviewed schema-golden diff. The snapshot test is a required CI gate.
- **Secrets via stdin/env, never argv.** Persist to the OS keyring (cli-implement).
- **Fence free text** (titles, descriptions, features, review bodies) by default.
- Version is injected via ldflags; `var version = "dev"` must stay a plain literal (go#64246).

## Next step
`cli-implement` replaces `internal/provider/stub.go` with real backends + keyring auth.
