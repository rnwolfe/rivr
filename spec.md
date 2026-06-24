# spec.md — rivr

> The build spec for an agent-focused CLI. Written by `cli-plan`; consumed by `cli-scaffold`,
> `cli-implement`, and `cli-publish`. Keep it current — it is the single source of truth.

`rivr` — an agent-focused CLI for **search and data retrieval against Amazon Shopping**
(consumer retail catalog): product search, product detail, live offers/pricing, customer
reviews, and product variations. **Read-only by design** — no cart/order/mutation surface.

## Interaction model (why read-only)
`rivr` is a **read → decide → hand-off** tool. Every command's terminal output is structured
product data plus a **canonical deep link** (`url` → `https://www.amazon.com/dp/<ASIN>`). The
agent surfaces the link; the human completes the purchase in a browser, where their
logged-in Amazon session, payment, shipping address, and Prime already live. The CLI never
touches cart/checkout.

Read-only is not merely a safety posture — **none of the backends expose a cart/checkout
surface** (third-party providers and the official Creators API are search/catalog/affiliate
APIs). There is no mutating operation to gate; the deep link *is* the action boundary. The
contract's `--allow-mutations`/`--dry-run`/etc. ship for uniformity but are inert.

**Associate/partner tag (monetization + attribution).** The official path is affiliate-
oriented: deep links may carry `?tag=<associate-id>` for attribution. rivr decorates every
emitted product URL (`search` items, `item get`, `variations`) with an Associates tag, with
a **disclosed, opt-out-able built-in default** (the "sponsorware" model):

- **Default** — a built-in project tag (`DefaultAssociateTag`, currently the placeholder
  `rivr-20`; replace with the real registered tag before publishing). If a referred link
  leads to a purchase, Amazon pays the project a small referral fee **at no extra cost to the
  buyer**. This funds rivr's development *and* helps the project meet the Creators API's
  qualified-sales minimum (which gates official-API access) — a virtuous loop tying the
  default backend's monetization to keeping the optional official backend alive.
- **Replaceable** — `--associate-tag <id>` / `RIVR_ASSOCIATE_TAG` substitutes the user's own
  tag (no default applied).
- **Opt-out** — `--no-associate-tag` / `RIVR_NO_ASSOCIATE_TAG` emits undecorated `/dp/<ASIN>`
  links and prints a single, non-pushy stderr notice explaining what was turned off.
- **Disclosed** (the ethics bar): the active tag is visible in every URL, in `rivr doctor`,
  in `schema --json` (`safety.associate_tag {mode,tag}`), and in `SKILL.md`/README. Not a
  secret — a plain config value.
- **Compliance note**: the Amazon Associates Operating Agreement requires the *associate*
  (the maintainer) to disclose the affiliate relationship to end users; the README/SKILL.md
  disclosure satisfies this. cli-implement should default the official Creators backend's
  partner tag from the same value rather than configuring it twice.

## Target
- **Service**: Amazon Shopping product catalog (consumer retail). Surfaced through a
  **pluggable provider backend**, not a single upstream.
- **Surface**: Three provider classes behind one normalized interface —
  1. **Third-party data providers (DEFAULT)** — SerpApi, Rainforest API, Oxylabs, Bright
     Data, etc. Plain JSON-over-HTTPS REST + API key. This is the feasible, agent-completable
     path and the **only** source of customer-review *text*.
  2. **Official Amazon Creators API (optional)** — `affiliate-program.amazon.com/creatorsapi`.
     Successor to PA-API 5.0. Four operations (search/getItems/getBrowseNodes/getVariations),
     OAuth 2.0 client-credentials auth. **Gated** (see ToS/risk).
  3. **DIY scraping of amazon.com (advanced opt-in, NOT default)** — documented as a
     last-resort provider; high breakage + bot-detection + ToS risk. Off unless explicitly
     selected.
- **Rate limits / pagination**:
  - *Third-party*: per-provider, credit/quota based (e.g. SerpApi free 250/mo, $25/mo →
    1,000; Rainforest ~$23/mo → 500 credits). API returns quota headers; surface them.
  - *Creators API (official)*: starts **1 TPS / 8,640 req/day**, scales with trailing-30-day
    shipped revenue (~1 TPS per $4,320, cap 10 TPS). Search hard cap **10 pages × 10 items =
    100 results/query** (`ItemPage` 1–10, `ItemCount` 1–10). No deep pagination.
  - Normalize: emit `nextCursor` only when the active provider can actually page further;
    omit it (signal end) at the provider's ceiling. Truncate loudly (contract §6).
- **ToS / risk** — STATE LOUDLY:
  - **PA-API 5.0 is DEAD** (deprecated 2026-04-30, endpoint off 2026-05-15). Do not build
    against it. OffersV1 retired 2026-01-31. The official path is the **Creators API** only.
  - **Official Creators API eligibility is a hard wall**: requires an *approved Amazon
    Associates account* + **≥10 qualified sales in the trailing 30 days** to get AND KEEP
    access (`AssociateNotEligible` / HTTP 403 on a sales drought; auto-restores ~2 days after
    referred sales ship). Chicken-and-egg for non-affiliates → effectively unobtainable for a
    hobbyist/MVP. This is *why* the official backend is optional, not default.
  - **Third-party providers**: each carries its own ToS; credentials are paid API keys
    (treat as sensitive). Verify review-pagination depth per provider — common gap.
  - **DIY scraping**: Amazon Conditions of Use prohibit robots/scrapers; 2026 bot detection
    (AWS WAF Bot Control + TLS/JA3 fingerprinting + a March-2026 BSA update reportedly
    targeting agentic crawlers). "Robot Check" pages return HTTP 200 (silent garbage). High
    breakage (per-session A/B DOMs). Advanced opt-in only; loud warnings.
  - **Official API data gap**: NO customer-review *text* and no reliable star-rating/review-
    count return (never re-added after PA-API 4→5). Review text/ratings ⇒ third-party only.
- **Prior art / competitive landscape**:
  - **No maintained general-purpose Amazon-product-search CLI exists** in any language —
    genuine whitespace. Everything below is a *library*, not an `amazon search …` binary.
  - **Python** (strongest SDK gravity): `python-amazon-paapi` (sergioteula, ~277★, Creators-
    ready) + official `amazon-creatorsapi-python-sdk` (v1.0.0, Feb 2026). *Gaps vs contract:*
    library not a CLI — no read-only gate, no `schema --json`, no structured errors/exit
    codes, no output bounding, no prompt-injection fencing, no keyring.
  - **Node/TS**: `amazon-paapi` (jorgerosal, ~91★) healthiest; TS-native weak. Same contract
    gaps (it's a wrapper lib).
  - **Go**: `goark/pa-api` (~43★, single-maintainer, already pivoted to Creators/OAuth2).
    Library only; same gaps.
  - **Third-party provider SDKs/clients** exist but are generic REST clients — no agent
    affordances.
- **Build verdict**: **BUILD.** No agent-engineered CLI exists for this target; the libraries
  miss every contract pillar an agent needs. Concrete differentiators:
  1. **Structurally read-only** — search/retrieval only; no mutation surface to misfire.
  2. **Pluggable provider abstraction with a single normalized output schema** — survives the
     PA-API→Creators churn and lets an agent switch backends without relearning fields.
  3. **Prompt-injection fencing ON by default** — product titles, bullet features,
     descriptions, and especially *review text* are attacker-controlled free text; fence them
     (contract §8). No existing tool does this for Amazon data.
  4. **Full agent self-description + token discipline** — `schema --json`, embedded `agent`
     SKILL.md, structured errors + stable exit codes (incl. distinct `RATE_LIMITED` and
     `AUTH_REQUIRED`/eligibility codes), `--limit`/`--select` bounding.
  - **Mine for mechanics**: `goark/pa-api` (Creators OAuth2 client-credentials dance + the
    four-operation request shapes); SerpApi/Rainforest docs for normalization field mapping.

## Language & framework
- **Language**: **Go**
- **Rationale (SDK gravity > distribution > performance)**: SDK gravity is *neutral* here —
  the default backend (third-party providers) is plain REST+API-key needing no SDK, and even
  the official Creators backend is 4 REST ops behind OAuth client-credentials (direct HTTP is
  fine). With no SDK forcing a language, the deciding axis is the **agent hot-loop**: a
  search/retrieval tool is invoked repeatedly, so **single static binary + lowest cold start**
  wins → Go (factory default). *(TS/Bun was considered for an `npx` zero-install human trial;
  cold-start for the agent loop outweighs it. Python's richer official-API SDK doesn't pull,
  since that backend isn't the default and is trivially direct-HTTP.)*
- **Framework**: **kong** (typed-grammar-as-data → `schema --json` is a reflection walk).
- **SDK/library used**: **Direct HTTP** for all backends (per-provider request builders +
  one normalizer). Optionally vendor `goark/pa-api` request shapes for the Creators backend.
- **Blueprint**: references/research/blueprint-go.md
- **Language-specific gotchas to honor**: GoReleaser + `homebrew_casks`; embed `SKILL.md`
  via `//go:embed`; keep the provider interface small so `schema --json` stays backend-stable.

## Auth
- **Model**: **Per-provider API key / bearer** (third-party, DEFAULT) · **OAuth 2.0
  client-credentials** (official Creators backend) · none for DIY-scrape (opt-in).
- **Provider constraints**:
  - Third-party: long-lived API key/bearer; no browser, no redirect, no user login. Quota is
    credit-based, not time-based.
  - Creators API: confidential client → needs client-id + client-secret; exchanges for a
    short-lived bearer (~1h TTL) → must refresh. Plus the **Associate eligibility wall** above
    (an auth-shaped failure that is really an account-state failure → distinct exit code +
    remediation pointing at the Associates dashboard, not "log in again").
- **Feasible path to usability (end-to-end)** — fully agent-completable, no browser:
  1. **Primary / default (third-party)**: `rivr auth login --provider serpapi` reads the API
     key from **stdin** (`--token-stdin`, never argv) → stored in OS keyring. Fully headless
     thereafter; `rivr search "usb-c cable"` works immediately. User obtains the key once from
     the provider dashboard (one-time, out of band).
  2. **Optional (official Creators)**: `rivr auth login --provider creators` reads
     client-id + client-secret from stdin → tool runs the **client-credentials grant**
     itself, caches the bearer, and `auth refresh` re-mints on expiry. No redirect URI, no
     callback, no cert. *Precondition the tool checks in `doctor`/`auth status`: caller is an
     eligible Associate; if `AssociateNotEligible`, emit the eligibility exit code with a
     dashboard remediation.*
  - Never browser-only as the sole path (contract §7) — satisfied: every backend is stdin +
    headless.
- **Secret storage**: OS keyring + 0600 XDG fallback; warn on insecure perms (contract §7).
  Keyed per provider so multiple backends coexist.
- **Subcommands**: `auth login | status | logout | refresh`; `doctor`.

## Command surface (noun-verb)
All commands are **reads**. Service-namespaced; `--provider` selects the backend (flag >
config default). Mutations: **none** — the `--allow-mutations` gate is present (scaffold) but
no command consumes it; the tool is structurally read-only.

| Command | Read/Mutation | Description | Key output fields |
|---|---|---|---|
| `rivr search <query>` | read | Keyword/category product search (filters: `--category`, `--min-rating`, `--prime`, `--min-price`/`--max-price`, `--sort`). | `items[]{asin,title*,price,currency,rating,reviewCount,prime,url,image}`, `nextCursor`, `provider` |
| `rivr item get <asin...>` | read | Full product detail for one or more ASINs (`--detailed` adds bullets/specs). | `asin,title*,brand,price,currency,offers,features*[],description*,images[],rating,reviewCount,salesRank,url` |
| `rivr item offers <asin>` | read | Live offers / buybox / price / availability (OffersV2-style). | `offers[]{price,currency,condition,merchant,prime,availability}`, `buyboxPrice` |
| `rivr reviews <asin>` | read | Customer reviews (text + rating). **Third-party providers only**; on official/scrape backends returns a structured `UNSUPPORTED_BY_PROVIDER` error. | `reviews[]{rating,title*,body*,author,date,verified}`, `nextCursor` |
| `rivr variations <asin>` | read | Size/color/style variations of a parent product. | `parentAsin,variations[]{asin,attributes,price,url}` |
| `rivr browse <node-id>` | read | Browse-node (category) tree metadata. *(Official Creators backend; degrade gracefully on providers without it.)* | `nodeId,name,ancestors[],children[]` |
| `rivr provider list` | read | Configured providers + which is default + auth status of each. | `providers[]{name,configured,default,capabilities}` |

`*` = free-text field from the target → **fenced as untrusted by default in agent mode**
(contract §8).

## Exit codes
Start from contract §4; target-specific additions in **bold**.
```
0   ok                      5  not found (bad/unknown ASIN or node)
1   generic error           6  permission denied / **ELIGIBILITY (AssociateNotEligible)**
2   usage/parse             7  **RATE_LIMITED / quota exhausted (provider or official)**
3   empty results (search   8  retryable/transient (upstream 5xx, network)
    returned 0 matches)     10 config error (no provider configured / bad default)
4   auth required           **11 UNSUPPORTED_BY_PROVIDER (e.g. reviews on official backend)**
   (missing/invalid key)    13 input required (--no-input hit a prompt)
                            130 cancelled (SIGINT)
```
Note: code **6** carries two distinct `code` strings in the JSON error body —
`PERMISSION_DENIED` and `ASSOCIATE_NOT_ELIGIBLE` (latter remediation → Associates dashboard).
`3` (empty results) is a success-adjacent signal, never collapsed into `5`/`1`.

## Output schema
Single **provider-normalized** shape; `schemaVersion` field; append-only. Free-text fields
carry the untrusted-fence wrapper in agent mode.

```jsonc
// rivr search
{
  "schemaVersion": "1",
  "provider": "serpapi",
  "query": "usb-c cable",
  "items": [
    { "asin": "B0xxxxxxx", "title": "<untrusted>…</untrusted>", "price": 12.99,
      "currency": "USD", "rating": 4.6, "reviewCount": 21034, "prime": true,
      "url": "https://www.amazon.com/dp/B0xxxxxxx", "image": "https://…" }
  ],
  "nextCursor": "page:2",      // omitted at provider ceiling = end of results
  "count": 1, "limit": 50
}

// rivr item get
{ "schemaVersion": "1", "provider": "...", "asin": "...", "title": "...", "brand": "...",
  "price": 0.0, "currency": "USD", "offers": [ … ], "features": [ … ], "description": "…",
  "images": [ … ], "rating": 4.6, "reviewCount": 21034, "salesRank": 142, "url": "…" }

// rivr reviews
{ "schemaVersion": "1", "provider": "...", "asin": "...",
  "reviews": [ { "rating": 5, "title": "…", "body": "…", "author": "…",
                 "date": "2026-05-01", "verified": true } ],
  "nextCursor": null }
```
Normalization mapping (SerpApi / Rainforest / Creators → this shape) is documented per
provider in `cli-implement`. Missing-but-not-erroring fields are `null`, never dropped
(append-only stability).

## Universal contract surface (provided by scaffold — confirm no conflicts)
`--format json|plain|tsv` · `--allow-mutations` (present but unused — read-only tool) ·
`--dry-run` · `--yes`/`--force` · `--no-input` · `--limit` (default 50) · `--select` ·
`--concise`/`--detailed` · `schema --json` · `agent`. Plus global `--provider`.
No conflicts: `--provider` is the only tool-specific global.

## Distribution
- **Targets**: `go install` + Homebrew tap (GoReleaser `homebrew_casks`) + release binaries
  (linux/macos, amd64/arm64).
- **Trial path**: `brew install <tap>/rivr` or download a release binary → `rivr auth login
  --provider serpapi` (renewing free tier) → `rivr search …`. No build toolchain needed.
- **Agent hot-loop path**: the single static Go binary (lowest cold start; invoked in loops).

## Publish
- **Flag**: **full** (portfolio-bound).
- **If full**: docs site (starlight-docs) · doc content (harvest-docs) · release (release
  skill) · README + VHS demo · hygiene files · discoverability (Show HN / awesome-lists /
  the agent-CLI cluster). Landing page emphasizing the read-only + injection-fenced +
  provider-pluggable story.

## Prompt-injection surface
**High — load-bearing.** All seller/buyer-authored free text is attacker-controllable:
- `search` → item **titles**
- `item get` → **titles, bullet features, descriptions**
- `reviews` → **review titles + bodies** (the sharpest vector — anyone can post a review)
- `variations`/`browse` → names/attributes
These fields are wrapped as untrusted by default in agent mode (`--wrap-untrusted` on; contract
§8). Numeric/ID fields (asin, price, rating, url) are not fenced. A malicious listing or
review attempting "ignore previous instructions…" must reach the agent already fenced.
