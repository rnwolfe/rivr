---
title: Authentication & security
description: How rivr stores credentials, the doctor command, and the secret-handling threat model.
---

Credentials are read from **stdin** (never argv) and stored in the OS keyring, with a `0600`
file fallback. Set `RIVR_KEYRING=file` to force the file backend on headless hosts.

## Logging in

```bash
# third-party providers take a single API key:
printf %s "$KEY" | rivr auth login --provider serpapi
printf %s "$KEY" | rivr auth login --provider rainforest

# the official Creators backend takes client_id then client_secret (two lines):
printf '%s\n%s' "$CLIENT_ID" "$CLIENT_SECRET" | rivr auth login --provider creators
```

Getting a credential:

- **SerpApi** — sign up, copy the key from [serpapi.com/manage-api-key](https://serpapi.com/manage-api-key). Renewing free tier.
- **Rainforest** — [Traject Data](https://trajectdata.com/) dashboard.
- **Creators (official)** — requires an **approved Amazon Associate** with ≥10 qualifying sales in the trailing 30 days, else every call returns `ASSOCIATE_NOT_ELIGIBLE` (exit 6). Generate a Credential ID + Secret in Associates Central → Tools → Creators API.

## status / logout / refresh

```bash
rivr auth status --provider serpapi --json   # actively tests; token redacted
rivr auth logout --provider serpapi          # removes LOCAL creds only
rivr auth refresh --provider creators         # re-mints the OAuth token
```

`auth logout` does **not** revoke server-side — rotate at the provider to invalidate a key.

## doctor

```bash
rivr doctor --json
```

Checks the active provider, live connectivity (skipped during a throttle cooldown so it
doesn't deepen a block), secret-file permissions, fencing, and affiliate state, with a fix
under each failure.

## Secret-handling threat model

- **Storage:** OS keyring (Keychain / Secret Service / Credential Manager); `0600` file
  fallback under `$XDG_DATA_HOME/rivr/`. `doctor` warns on loose perms.
- **Never via argv** — secrets come from stdin/env; argv leaks to `ps`/`/proc`/history/agent logs.
- **Redaction** — `auth status` reports validity only; rivr never logs credentials.
- **Token cache** — the Creators bearer is cached `0600` under `$XDG_STATE_HOME/rivr/`;
  `auth refresh` rotates it.
- **Read-only** — no mutations/purchases, so a compromised invocation's blast radius is data
  disclosure within the provider's scope, not account state change.

See [SECURITY.md](https://github.com/rnwolfe/rivr/blob/main/SECURITY.md) for reporting and the
leaked-token runbook.
