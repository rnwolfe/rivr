# AGENTS.md — rivr

Agent-focused CLI for **Amazon Shopping search and data retrieval**. Read-only; pluggable
provider backend. Built to the agent-CLI contract (see `spec.md`).

## Build / test / run
```bash
go build ./...                       # build
go vet ./...                         # vet
go test ./...                        # contract tests (incl. schema-snapshot gate)
go run ./cmd/rivr search "usb-c cable" --json
RIVR_UPDATE_GOLDEN=1 go test ./internal/cli/...   # regen schema golden after an INTENDED change
```

## Layout
- `cmd/rivr/main.go` — only `os.Exit(cli.Run(...))`; all logic in `internal/cli`.
- `internal/cli/` — kong grammar (`root.go`), one file per noun (`search.go`, `item.go`,
  `reviews.go`, `variations.go`, `browse.go`, `provider.go`), `misc.go` (auth/doctor/schema/
  agent/version), `suggest.go` ("did you mean").
- `internal/provider/` — **the normalized output schema + the pluggable backend interface**.
  `stub.go` is a placeholder; `cli-implement` adds real backends (serpapi/rainforest/creators)
  and wires keyring auth, then points `DefaultName()` at `serpapi`.
- `internal/output/` — output contract (stdout/stderr split, `--format`, `--select`, `--limit`).
- `internal/errs/` — stable exit-code table + structured `CLIError`.
- `internal/fence/` — untrusted-text fencing (contract §8).
- `internal/skill/SKILL.md` — embedded agent contract (printed by `rivr agent`).

## Conventions
- **Read-only**: no command mutates. The `--allow-mutations`/`--dry-run`/`--yes`/`--force`
  flags exist for contract uniformity but are no-ops; `Runtime.Guard` stays default-deny.
- **stdout = data, stderr = chatter.** Never `fmt.Println` to stdout — use `output.Writer`.
- **Output schema is append-only** (`internal/provider` types). Field rename/removal = a
  reviewed schema-golden diff. The snapshot test is a required CI gate.
- **Secrets via stdin/env, never argv.** Persist to the OS keyring (cli-implement).
- **Fence free text** (titles, descriptions, features, review bodies) by default.
- Version is injected via ldflags; `var version = "dev"` must stay a plain literal (go#64246).

## Web presence freshness (keep docs + landing + cards in sync)

The site lives in `site/` (Astro + Starlight). **One shared token source** —
`site/src/styles/tokens.css` — styles BOTH the landing (`site/src/pages/index.astro`) and the
docs (via `customCss` in `site/astro.config.mjs`). Never hand-copy tokens.

When the **value proposition, command surface, flags, exit codes, providers, or brand**
change, in the **same PR**:

1. Update the affected docs page under `site/src/content/docs/` (and the repo `docs/backends.md` if the matrix changed).
2. Update the **landing** copy/examples in `site/src/pages/index.astro` (hero, comparisons, backend matrix) and the README's mirrored matrix.
3. **Regenerate OG cards** if any page title/headline changed: `cd site && node scripts/gen-og.mjs` (keep the page list in `scripts/gen-og.mjs` and the `known` list in `src/components/Head.astro` in sync). If the brand/tagline/headline command changed, also regenerate the GitHub social-preview card: `cd site && node scripts/gen-social.mjs` → `.github/social-preview.png` (re-upload in repo Settings → Social preview).
4. Rebuild (`cd site && pnpm build`) so `llms.txt` regenerates, and keep the embedded `internal/skill/SKILL.md` aligned (it's the agent contract shipped in the binary).

Build/preview the site: `cd site && pnpm build` / `pnpm dev` (binds 0.0.0.0). Deploy is Vercel.

## Stages
`cli-plan` → `cli-scaffold` → `cli-implement` (done) → `cli-publish` (this web presence + hygiene).
