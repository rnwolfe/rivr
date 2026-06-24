---
title: Command reference
description: Every rivr command, its flags, and what it returns. All commands are read-only.
---

All commands are **reads**. Global flags work everywhere; `schema --json` is the
always-current machine-readable source.

## Global flags

| flag | purpose |
|---|---|
| `--json` / `--format json\|plain\|tsv` | output format (data → stdout, everything else → stderr) |
| `--select a,b.c` | project top-level fields (dot-path) |
| `--limit N` | bound list size (default 50) |
| `--detailed` | richer output (e.g. `item get` bullets/specs) |
| `--provider <name>` | choose a backend (or `RIVR_PROVIDER`) |
| `--associate-tag <id>` / `--no-associate-tag` | deep-link affiliate tag |
| `--no-wrap-untrusted` | disable prompt-injection fencing |
| `--no-input` | never prompt; fail with exit 13 instead |

## Reads

```bash
rivr search <query>      # --category --prime --min-rating --min-price --max-price --sort --cursor
rivr item get <ASIN>...  # one or more ASINs; --detailed
rivr item offers <ASIN>  # live offers / buybox / availability
rivr reviews <ASIN>      # --cursor; response carries scope: full | sample
rivr variations <ASIN>   # size/color/style
rivr browse <node-id>    # category (browse-node) tree — creators backend only
rivr provider list       # backends + capability/cost/risk descriptor
```

`--select` projects **top-level** fields, so it works on object results (`item get`) directly;
for `search`, project inside `items` with `jq` (e.g. `… --json | jq '.items[] | {asin,price}'`).

## Auth & ops

```bash
rivr auth login|status|logout|refresh
rivr doctor              # diagnostics
rivr schema              # command tree, flags, exit codes, providers, live safety state
rivr agent               # print the embedded agent guide (SKILL.md)
rivr version
```

## Exit codes

| code | meaning |
|---|---|
| 0 | ok |
| 2 | usage / parse |
| 3 | empty results |
| 4 | auth required |
| 5 | not found |
| 6 | permission / `ASSOCIATE_NOT_ELIGIBLE` |
| 7 | rate limited / blocked (carries `retryAfterSeconds`) |
| 8 | retryable (transient; auto-retried) |
| 9 | upstream / schema drift |
| 10 | config |
| 11 | unsupported by provider |
| 13 | input required (`--no-input` hit a prompt) |
| 130 | cancelled |

Errors are `{error, code, remediation, retryAfterSeconds?}` on stderr.
