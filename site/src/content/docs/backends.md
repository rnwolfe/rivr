---
title: Backends — capability, cost & risk
description: Choose a rivr backend by what it can do, what it costs, and what it risks. One schema across all of them.
---

`rivr` normalizes one schema across backends. Pick one with `--provider <name>` (or
`RIVR_PROVIDER`); the default is `serpapi`. This page mirrors the machine-readable profiles
from `rivr provider list --json` — that command is the source of truth, so an agent can choose
programmatically.

> Pricing/eligibility is **indicative** (2026) and changes — verify on each provider's site.

## Capabilities

| capability | serpapi (default) | rainforest | creators (official) | scrape (opt-in) | stub |
|---|:--:|:--:|:--:|:--:|:--:|
| search | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| item get | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| offers | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| reviews | ✓ sample | ✓ full | ✗ | ✗ (walled) | ✓ (fake) |
| variations | ✓ | ✓ | ✓ | ✗ | ✓ (fake) |
| browse nodes | ✗ | ✗ | ✓ | ✗ | ✓ (fake) |

A capability a backend lacks returns `UNSUPPORTED_BY_PROVIDER` (exit 11). Reviews carry a
`scope`: `full` (paginated) vs `sample` (a product-page sample — not the whole corpus).

## Auth, cost & deployment

| backend | keyless | auth | hosted/cloud-safe | rough cost |
|---|:--:|---|:--:|---|
| **serpapi** | ✗ | API key (stdin → keyring) | ✓ | Free tier ~250/mo (renewing); paid metered beyond |
| **rainforest** | ✗ | API key (stdin → keyring) | ✓ | Paid, credit-based; plans from ~$25/mo |
| **creators** | ✗ | OAuth2 client-credentials | ✓ | Free API — needs an approved Associate w/ ≥10 qualifying sales / 30 days |
| **scrape** | ✓ | none (`RIVR_SCRAPE_ENABLE=1`) | ✗ residential only | Free; cost is your bandwidth/IP + selector upkeep |
| **stub** | ✓ | none | ✓ | Free (fixtures) |

## Risks

- **serpapi** — third-party dependency; paid beyond free tier; reviews are a page sample; no browse.
- **rainforest** — third-party dependency; paid per request.
- **creators** — eligibility wall (`ASSOCIATE_NOT_ELIGIBLE`, exit 6); **no review text**; portal access-gated (endpoints env-overridable via `RIVR_CREATORS_*`).
- **scrape** — Amazon ToS; bot-detection/blocking (→ `BLOCKED` + a persisted cooldown); fragile DOM (→ `SCHEMA_DRIFT`). **Do not run from cloud/hosted IPs.**
- **stub** — not real data; offline/testing only.

## Which backend should I use?

- **Agent or you at home (residential):** `scrape` (keyless, free) for modest throttled use, or
  `serpapi` (free tier) if you'd rather not scrape. This is the primary use case.
- **Hosted / cloud / datacenter:** never `scrape`. Use `creators` if you're an eligible
  Associate, otherwise `serpapi`/`rainforest`.
- **Need review text:** `rainforest` (full) or `serpapi` (sample); `creators` and `scrape` return none.
- **Need browse-node trees:** `creators` only.

## Choosing programmatically

```bash
# keyless and usable from any host:
rivr provider list --json | jq '[.[] | select(.keyless and .hostedSafe) | .name]'
# backends that return full review text:
rivr provider list --json | jq '[.[] | select(.reviewsScope == "full") | .name]'
```
