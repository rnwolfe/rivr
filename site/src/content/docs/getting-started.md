---
title: Getting started
description: Install rivr and reach a first result in under a minute — offline, then against real Amazon data.
---

`rivr` is a **read-only** Amazon Shopping CLI for AI agents and humans. It searches the
catalog and returns product detail, offers, reviews, and variations as structured JSON through
a pluggable backend. It never adds to cart or buys — the terminal output is data plus a
canonical `/dp/<ASIN>` deep link.

## Install

```bash
# curl (macOS / Linux) — verifies the release SHA-256 before installing
curl -fsSL https://rivr.sh/install.sh | sh

# Homebrew (macOS / Linux)
brew install rnwolfe/tap/rivr

# Go (any platform; static binary, no CGO)
go install github.com/rnwolfe/rivr/cmd/rivr@latest
```

Or download a prebuilt binary from [Releases](https://github.com/rnwolfe/rivr/releases)
(checksums + SBOM + provenance included).

## First result in <60s (no account)

The `stub` backend returns fixtures, so you can explore the shape with no key and no network:

```bash
RIVR_PROVIDER=stub rivr search "usb-c cable" --json
RIVR_PROVIDER=stub rivr item get B0CXYZ123 --detailed --json
rivr schema | jq '{read_only: .safety.read_only, exit_codes}'
```

## Against real data

The lowest-friction real backend is **SerpApi** (renewing free tier). The key is read from
**stdin**, never as a flag:

```bash
printf %s "$SERPAPI_KEY" | rivr auth login --provider serpapi
rivr auth status --provider serpapi --json   # actively tests the key
rivr search "mechanical keyboard" --prime --min-rating 4 --json
```

No key at all? On a **home/residential** connection you can use the keyless scraper (opt-in;
see [Backends](/backends) for the Amazon-ToS caveats):

```bash
RIVR_SCRAPE_ENABLE=1 rivr --provider scrape search "usb-c cable" --json
```

## Next steps

- [Backends](/backends) — choose by capability, cost, and risk.
- [Authentication & security](/auth) — credential storage, `doctor`, threat model.
- [Commands](/commands) — the full command/flag reference.
- [For agents](/agents) — the contract an LLM agent relies on.
