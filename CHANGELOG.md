# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.1] - 2026-06-25

### Added
- `install.sh` (served at `https://rivr.sh/install.sh`): `curl -fsSL https://rivr.sh/install.sh | sh`
  — detects os/arch, downloads the release tarball, **verifies its SHA-256** against
  `checksums.txt`, installs to `~/.local/bin` (or `$RIVR_INSTALL_DIR`). Function-wrapped for
  truncation safety. Now the primary install one-liner on the site/README.

### Changed
- **Homebrew is now a cross-platform formula** (`Formula/rivr.rb`) instead of a macOS-only
  cask, so `brew install rnwolfe/tap/rivr` works on **Linux and macOS**.

## [0.3.0] - 2026-06-25

### Added
- `rivr version --check` — a pull-based release check returning
  `{current, latest, updateAvailable, upgrade}`.
- A passive, cached (24h) upgrade notice on the **human TTY path** (stderr) — **silent for
  agents** (`--json`/non-TTY/`--no-input`); disable with `RIVR_NO_UPDATE_CHECK=1`.
- `doctor` reports update status.

### Notes
- rivr **never auto-updates** and never instructs an agent to update its own binary; it only
  surfaces the upgrade command to a human (mutating tooling mid-task breaks determinism).

## [0.2.0] - 2026-06-25

### Added
- `item compare <ASIN>...` — fetch multiple ASINs and return them with a best-of summary
  (`cheapest` / `highestRated` / `mostReviewed`), so agents don't hand-assemble comparisons.
- `badges` field on search results (e.g. "Amazon's Choice", "Best Seller") — a trust signal
  for "find the best." Population is provider-dependent (best-effort).

### Changed
- Embedded `SKILL.md` + docs now instruct driving agents to surface the item `url` **verbatim**
  (so the affiliate tag survives instead of being dropped by a hand-rebuilt link), and surface
  the search filters (`--sort`/`--min-rating`/`--prime`) more prominently.

## [0.1.3] - 2026-06-25

### Fixed
- **scrape**: search results always include a product `url` (synthesized from the ASIN when
  Amazon's DOM omits the link) and a more robust `reviewCount` (multiple selector fallbacks).
- **scrape**: `reviews` now correctly reports `UNSUPPORTED_BY_PROVIDER` (Amazon walls the
  reviews page) instead of returning a silent empty list; capability matrix updated.
- **doctor** no longer fails when the default provider is unconfigured but the keyless
  `scrape` backend is enabled — it reports the keyless path as usable.

### Added
- `--version` flag (the common probe; complements the `version` subcommand).
- `truncated` field on `search`/`reviews` JSON when results exceed `--limit` (in-band signal
  for agents piping straight to `jq`).
- `search` results are de-duplicated by ASIN.
- `AUTH_REQUIRED` suggests `--provider scrape` when `RIVR_SCRAPE_ENABLE=1` is set.

## [0.1.0] - 2026-06-24

### Added

- Read-only Amazon Shopping CLI: `search`, `item get`/`item offers`, `reviews`,
  `variations`, `browse`, `provider list`.
- Pluggable backend with one normalized schema: **serpapi** (default), **rainforest**,
  official **creators** (OAuth2 client-credentials), keyless **scrape** (opt-in,
  residential), and a **stub** offline backend. `rivr provider list --json` emits a full
  capability/cost/risk descriptor per backend.
- Agent-CLI contract: `--json`/`--format`, `--select`, `--limit`, structured
  `{error,code,remediation,retryAfterSeconds}` errors, stable exit-code table, example-led
  `--help`, `schema --json`, embedded `agent` SKILL.md, `doctor`.
- Safety: read-only (no mutations/purchases), prompt-injection fencing on by default,
  secrets via stdin → OS keyring (`0600` file fallback), persistent cross-process throttle
  with fail-fast cooldowns.
- Disclosed, opt-out built-in Amazon Associates tag on product deep links
  (`--associate-tag` / `--no-associate-tag`).

[Unreleased]: https://github.com/rnwolfe/rivr/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/rnwolfe/rivr/releases/tag/v0.3.1
[0.3.0]: https://github.com/rnwolfe/rivr/releases/tag/v0.3.0
[0.2.0]: https://github.com/rnwolfe/rivr/releases/tag/v0.2.0
[0.1.3]: https://github.com/rnwolfe/rivr/releases/tag/v0.1.3
[0.1.0]: https://github.com/rnwolfe/rivr/releases/tag/v0.1.0
