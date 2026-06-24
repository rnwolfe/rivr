# Backends — capability, cost & risk matrix

`rivr` normalizes one schema across several backends. Pick one with `--provider <name>` (or
`RIVR_PROVIDER`); the default is `serpapi`.

This page is the human view of the machine-readable profiles emitted by
`rivr provider list --json` — that command is the single source of truth, so an agent can
choose a backend programmatically (e.g. `rivr provider list --json --select name,keyless,hostedSafe,reviewsScope`).

> Pricing/eligibility below is **indicative** (2026) and changes — verify on each provider's
> site before relying on it.

## Capabilities

| capability | serpapi (default) | rainforest | creators (official) | scrape (opt-in) | stub |
|---|:--:|:--:|:--:|:--:|:--:|
| search | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| item get | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| offers | ✓ | ✓ | ✓ | ✓ | ✓ (fake) |
| reviews | ✓ sample | ✓ full | ✗ | ✓ full | ✓ (fake) |
| variations | ✓ | ✓ | ✓ | ✗ | ✓ (fake) |
| browse nodes | ✗ | ✗ | ✓ | ✗ | ✓ (fake) |

A capability a backend lacks returns `UNSUPPORTED_BY_PROVIDER` (exit 11). Review `scope` is in
every reviews response: `full` (paginated) vs `sample` (a product-page sample — not the whole
corpus).

## Auth, cost & deployment

| backend | keyless | auth | hosted/cloud-safe | rough cost |
|---|:--:|---|:--:|---|
| **serpapi** | ✗ | API key (stdin → keyring) | ✓ | Free tier ~250 searches/mo (renewing); paid metered beyond |
| **rainforest** | ✗ | API key (stdin → keyring) | ✓ | Paid, credit-based (no free tier; small trial); plans from ~$25/mo |
| **creators** | ✗ | OAuth2 client-credentials | ✓ | Free API — but needs an approved Associate w/ ≥10 qualifying sales / 30 days |
| **scrape** | ✓ | none (`RIVR_SCRAPE_ENABLE=1`) | ✗ residential only | Free; cost is your bandwidth/IP + selector upkeep |
| **stub** | ✓ | none | ✓ | Free (fixtures) |

## Risks

| backend | headline risk |
|---|---|
| **serpapi** | third-party dependency; paid beyond free tier; reviews are a page sample; no browse |
| **rainforest** | third-party dependency; paid per request |
| **creators** | eligibility wall (`ASSOCIATE_NOT_ELIGIBLE`, exit 6); **no review text**; portal is access-gated (endpoints env-overridable) |
| **scrape** | Amazon ToS; bot-detection/blocking (→ `BLOCKED` + cooldown); fragile DOM (→ `SCHEMA_DRIFT`); **do not run from cloud/hosted IPs** |
| **stub** | not real data — offline/testing only |

## Which backend should I use?

- **Agent or you at home (residential):** `scrape` (keyless, free) is plausible for modest,
  throttled use — or `serpapi` (free tier) if you'd rather not scrape. This is the primary
  use case.
- **Hosted / cloud / datacenter:** never `scrape` (it gets blocked). Use `creators` if you're
  an eligible Associate, otherwise `serpapi`/`rainforest`.
- **Need full review text:** `rainforest` (or `scrape`); `serpapi` gives only a sample;
  `creators` gives none.
- **Need browse-node trees:** `creators` only.
- **Lowest friction to try:** `serpapi` free key, or `scrape` keyless at home.
- **Cleanest data licensing:** `creators` (first-party), if eligible.

## Choosing programmatically

```bash
# backends that work without a key, usable from any host:
rivr provider list --json | jq '[.[] | select(.keyless and .hostedSafe) | .name]'

# backends that return full review text:
rivr provider list --json | jq '[.[] | select(.reviewsScope == "full") | .name]'
```
