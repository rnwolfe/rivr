# rivr

> An agent-friendly CLI for **Amazon Shopping search and data retrieval**. Read-only by
> design, structured JSON everywhere, self-describing, and built for LLM agents to drive
> safely.

`rivr` searches Amazon's catalog and retrieves product detail, live offers, customer
reviews, and variations through a **pluggable provider backend** — a third-party data
provider (default) or the official Amazon Creators API. It implements the
[agent-CLI contract](./spec.md): read-only, `schema --json`, structured errors with stable
exit codes, bounded output, an embedded agent guide, and **prompt-injection fencing on by
default**.

> [!NOTE]
> Scaffolded skeleton. The real provider integrations and keyring auth are wired by
> `cli-implement`; today it ships a `stub` backend so the contract surface is runnable and
> tested offline.

## Why a CLI (not an MCP server)

No maintained general-purpose Amazon-product-search CLI exists — and the official Product
Advertising API (PA-API 5.0) was retired in May 2026. `rivr` fills that gap with a tool an
agent can drive in a hot loop at near-zero token cost.

## Install

```bash
go install github.com/rnwolfe/rivr/cmd/rivr@latest
# or, once published:
brew install rnwolfe/tap/rivr
```

## Quick start

```bash
# configure a backend (key read from stdin, never argv)
printf %s "$SERPAPI_KEY" | rivr auth login --provider serpapi

rivr search "usb-c cable" --json
rivr item get B0XXXXXXX --detailed --json
rivr item offers B0XXXXXXX --json
rivr reviews B0XXXXXXX --json          # third-party backends only
rivr variations B0XXXXXXX --json
rivr provider list --json
```

## Providers & authentication

`rivr` normalizes one schema across backends; pick with `--provider` (or `RIVR_PROVIDER`).
Secrets are read from **stdin** (never argv) and stored in the OS keyring, with a `0600` file
fallback (`RIVR_KEYRING=file` forces it on headless boxes).

| Provider | Get a credential | Notes |
|---|---|---|
| **serpapi** (default) | sign up → [serpapi.com/manage-api-key](https://serpapi.com/manage-api-key) → `printf %s "$KEY" \| rivr auth login --provider serpapi` | renewing free tier (~250/mo); reviews are a page **sample** |
| **rainforest** | [Traject Data](https://trajectdata.com/) dashboard → `printf %s "$KEY" \| rivr auth login --provider rainforest` | **full** paginated reviews + real offers |
| **creators** (official) | Associates Central → Tools → Creators API → `printf '%s\n%s' "$ID" "$SECRET" \| rivr auth login --provider creators` | needs an approved Associate w/ ≥10 sales/30d, else `ASSOCIATE_NOT_ELIGIBLE`; no review text |
| **scrape** (keyless) | `RIVR_SCRAPE_ENABLE=1` (opt-in) | for **home/residential** use only — modest throttled requests; ToS risk + fragile. Don't run from cloud/hosted IPs (→ use `creators`). |

`rivr auth status --json` tests the active provider; `rivr doctor --json` runs full
diagnostics. A missing credential returns `AUTH_REQUIRED` (exit 4) naming the login command.

## Agent-facing surface

- `rivr schema` — machine-readable command tree, exit codes, providers, live safety state.
- `rivr agent` — prints the embedded `SKILL.md` (the agent contract; no repo/network needed).
- `rivr doctor --json` — setup diagnostics.

## Safety

- **Read-only.** No cart/order/mutation surface. Mutation flags exist for contract
  uniformity but are no-ops.
- **Untrusted text is fenced.** Titles, descriptions, features, and review bodies — all
  attacker-controllable — are wrapped `‹untrusted›…‹/untrusted›` by default. Disable with
  `--no-wrap-untrusted`.
- **Secrets** are read from stdin/env and stored in the OS keyring, never passed as flags.

## Affiliate attribution (how rivr is funded)

Product deep links are tagged with an Amazon Associates ID by default. If a referred link
leads to a purchase, Amazon pays the project a small referral fee — **the buyer pays nothing
extra**. This funds development and helps the project meet the Amazon Creators API's
qualified-sales minimum that keeps official-API access alive.

It is fully transparent and in your control:

- The active tag is visible in every `url`, in `rivr doctor`, and in `rivr schema`.
- Use your own: `--associate-tag <your-id>` (or `RIVR_ASSOCIATE_TAG`).
- Turn it off: `--no-associate-tag` (or `RIVR_NO_ASSOCIATE_TAG`).

_As an Amazon Associate the maintainer earns from qualifying purchases._

## Exit codes

`0` ok · `2` usage · `3` empty results · `4` auth required · `5` not found ·
`6` permission / `ASSOCIATE_NOT_ELIGIBLE` · `7` rate limited · `8` retryable ·
`10` config · `11` unsupported by provider · `13` input required. Full table: `rivr schema`.

## Development

See [AGENTS.md](./AGENTS.md). Build/test: `go build ./... && go test ./...`.

## License

MIT
