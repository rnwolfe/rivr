# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/rnwolfe/rivr/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/rnwolfe/rivr/releases/tag/v0.1.3
[0.1.0]: https://github.com/rnwolfe/rivr/releases/tag/v0.1.0
