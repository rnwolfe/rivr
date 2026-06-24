# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/rnwolfe/rivr/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/rnwolfe/rivr/releases/tag/v0.1.0
