<div align="center">

# rivr

**A read-only Amazon Shopping CLI built for AI agents.**

Amazon's Product Advertising API was retired in 2026 and no agent-safe replacement existed —
so `rivr` is one. Search products, fetch detail/offers/reviews/variations through **one
normalized schema** over a pluggable backend (SerpApi, Rainforest, the official Amazon
Creators API, or keyless scraping), with the [Agent CLI Guidelines](https://aclig.dev/) baked
in: read-only by default, structured errors + stable exit codes, bounded JSON, prompt-injection
fencing, and stdin/keyring secrets.

[![ci](https://github.com/rnwolfe/rivr/actions/workflows/ci.yml/badge.svg)](https://github.com/rnwolfe/rivr/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/rnwolfe/rivr?sort=semver)](https://github.com/rnwolfe/rivr/releases/latest)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Agent CLI Guidelines: Full](https://aclig.dev/badge/agent-cli-guidelines-full.svg)](https://aclig.dev/conformance/)

![rivr demo](./demo/rivr.gif)

</div>

## Why rivr

- **Read-only by design.** No cart, no checkout, no mutations. Every result is structured data
  plus a canonical `/dp/<ASIN>` deep link — the hand-off point where a human buys in their
  browser. There's no destructive action for an agent to misfire.
- **Built for agents, not just humans.** `--json` everywhere, `schema --json`, an embedded
  `agent` guide, `{error,code,remediation,retryAfterSeconds}` errors, a stable exit-code table,
  and `--limit`/`--select` so responses stay within an agent's context budget.
- **Prompt-injection fenced.** Titles, descriptions, and review text are attacker-controllable;
  rivr wraps them `‹untrusted›…‹/untrusted›` by default.
- **Pluggable backend, one schema.** Swap SerpApi ↔ Rainforest ↔ Creators ↔ scrape without
  relearning fields. `rivr provider list --json` describes each backend's capability/cost/risk.

## Install

```bash
# Homebrew (macOS / Linux)
brew install rnwolfe/tap/rivr

# Go (any platform; static, no CGO)
go install github.com/rnwolfe/rivr/cmd/rivr@latest
```

Or grab a prebuilt binary (linux/macOS/windows, amd64/arm64) from
[Releases](https://github.com/rnwolfe/rivr/releases) — each ships with checksums, an SBOM,
and build-provenance attestation.

## Quickstart (first success in <60s, no account)

```bash
# the offline 'stub' backend returns fixtures — no key, no network
RIVR_PROVIDER=stub rivr search "usb-c cable" --json
RIVR_PROVIDER=stub rivr provider list --json --select name,keyless,hostedSafe,reviewsScope
rivr schema | jq '{read_only: .safety.read_only, exit_codes}'
```

Then point it at real data — the lowest-friction path is SerpApi's free tier:

```bash
# one-time, key read from stdin (never argv)
printf %s "$SERPAPI_KEY" | rivr auth login --provider serpapi
rivr search "mechanical keyboard" --prime --min-rating 4 --json
rivr item get B0XXXXXXX --detailed --json
```

No key at all? On a **home/residential** connection you can use the keyless scraper
(opt-in, Amazon-ToS caveats apply — see the backend matrix): `RIVR_SCRAPE_ENABLE=1 rivr
--provider scrape search "usb-c cable" --json`.

## Providers & authentication

`rivr` normalizes one schema across backends; pick with `--provider` (or `RIVR_PROVIDER`).
Secrets are read from **stdin** (never argv) and stored in the OS keyring, with a `0600` file
fallback (`RIVR_KEYRING=file` forces it on headless boxes).

| Provider | Get a credential | Notes |
|---|---|---|
| **serpapi** (default) | sign up → [serpapi.com/manage-api-key](https://serpapi.com/manage-api-key) → `printf %s "$KEY" \| rivr auth login --provider serpapi` | renewing free tier (~250/mo); reviews are a page **sample** |
| **rainforest** | [Traject Data](https://trajectdata.com/) dashboard → `printf %s "$KEY" \| rivr auth login --provider rainforest` | **full** paginated reviews + real offers |
| **creators** (official) | Associates Central → Tools → Creators API → `printf '%s\n%s' "$ID" "$SECRET" \| rivr auth login --provider creators` | needs an approved Associate w/ ≥10 sales/30d, else `ASSOCIATE_NOT_ELIGIBLE`; no review text |
| **scrape** (keyless) | `RIVR_SCRAPE_ENABLE=1` (opt-in) | **home/residential only** — modest throttled requests; ToS risk + fragile. Don't run from cloud/hosted IPs (→ use `creators`). |

### Capability / cost / risk at a glance

| backend | keyless | hosted-safe | reviews | cost | headline risk |
|---|:--:|:--:|:--:|---|---|
| **serpapi** (default) | ✗ | ✓ | sample | free tier + paid | paid beyond free tier; sample reviews |
| **rainforest** | ✗ | ✓ | full | paid (~$25/mo+) | paid per request |
| **creators** (official) | ✗ | ✓ | none | free* | *needs eligible Associate; no review text |
| **scrape** | ✓ | ✗ residential | none | free | Amazon ToS + blocking; no review text; home use only |
| **stub** | ✓ | ✓ | fake | free | not real data (testing) |

Full matrix + "which should I use?" → **[docs/backends.md](./docs/backends.md)**. Machine-readable
for agents: `rivr provider list --json`.

`rivr auth status --json` tests the active provider; `rivr auth logout` removes **local** creds
only (revoke at the provider to invalidate server-side); `rivr doctor --json` runs full
diagnostics. A missing credential returns `AUTH_REQUIRED` (exit 4) naming the login command.

## Commands

```bash
rivr search <query>          # keyword search (--category --prime --min-rating --min/max-price --sort)
rivr item get <ASIN>...      # product detail (--detailed adds bullets/specs)
rivr item offers <ASIN>      # live offers / buybox / availability
rivr item compare <ASIN>...  # side-by-side + best-of summary (cheapest/highest-rated/most-reviewed)
rivr reviews <ASIN>          # customer reviews (scope: full | sample)
rivr variations <ASIN>       # size/color/style variations
rivr browse <node-id>        # category (browse-node) tree  [creators only]
rivr provider list           # backends + capability/cost/risk
rivr auth login|status|logout|refresh
rivr doctor                  # diagnostics
rivr schema                  # machine-readable command tree, flags, exit codes, safety
rivr agent                   # print the embedded agent guide (SKILL.md)
```

Global flags: `--json`/`--format json|plain|tsv`, `--select a,b.c`, `--limit N`, `--detailed`,
`--provider`, `--associate-tag`/`--no-associate-tag`, `--no-wrap-untrusted`, `--no-input`.

## Exit codes

`0` ok · `2` usage · `3` empty results · `4` auth required · `5` not found ·
`6` permission / `ASSOCIATE_NOT_ELIGIBLE` · `7` rate limited / blocked · `8` retryable ·
`9` upstream / schema drift · `10` config · `11` unsupported by provider · `13` input required.
Full table: `rivr schema`.

## Affiliate attribution (how rivr is funded)

Product deep links are tagged with an Amazon Associates ID by default — if a referred link
leads to a purchase, Amazon pays the project a small referral fee, **the buyer pays nothing
extra**. It's fully transparent and in your control: the active tag is in every `url`, in
`rivr doctor`, and in `rivr schema`. Use your own with `--associate-tag <id>`, or disable with
`--no-associate-tag`.

_As an Amazon Associate the maintainer earns from qualifying purchases._

## Safety

- **Read-only.** No cart/order/mutation surface; the deep link is the action boundary.
- **Untrusted text fenced** by default (`--no-wrap-untrusted` to disable).
- **Secrets** via stdin/env → OS keyring (`0600` file fallback); never argv; redacted in output.

See [SECURITY.md](./SECURITY.md) for the secret-handling threat model.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [AGENTS.md](./AGENTS.md). Build/test:
`go build ./... && go vet ./... && go test ./...`.

## License

[MIT](./LICENSE) © Ryan Wolfe
