# Security Policy

`rivr` is a credential-handling CLI: it stores provider API keys / OAuth secrets and is
designed to be driven autonomously by LLM agents. Security is a first-class concern.

## Supported versions

| Version | Supported |
|---|---|
| latest `0.x` | ✅ |
| older | ❌ (upgrade to latest) |

Until `1.0`, only the latest release receives fixes.

## Reporting a vulnerability

**Do not open a public issue for security reports.** Use GitHub **Private Vulnerability
Reporting**: the repo's **Security → Report a vulnerability** tab. If that is unavailable,
email the maintainer (see the profile on GitHub) with `rivr security` in the subject.

- **Acknowledgement SLA:** within ~48 hours.
- **Coordinated disclosure:** we'll agree on a fix timeline and a disclosure date; please give
  us a reasonable window before any public detail.
- **Safe harbor:** good-faith research that respects user privacy and avoids data destruction
  will not be pursued.
- Include a minimal reproducible PoC and the affected version (`rivr version`).

## Secret-handling threat model

What rivr stores, where, and how it tries to fail safe:

- **Storage.** Credentials go to the OS keyring (macOS Keychain / Linux Secret Service /
  Windows Credential Manager). The fallback is a `0600` file under `$XDG_DATA_HOME/rivr/`
  (`RIVR_KEYRING=file` forces it on headless hosts). `doctor` warns if the file's perms are
  looser than `0600`.
- **Never via argv.** Secrets are read from **stdin** (`rivr auth login`) or env vars, never
  passed as flags — argv leaks to `ps`, `/proc`, shell history, and an agent's own command log.
- **Redaction.** `auth status` reports validity only; it never prints the stored secret.
  rivr does not log credentials.
- **Cross-process token cache.** The Creators OAuth bearer is cached `0600` under
  `$XDG_STATE_HOME/rivr/`; delete it or run `rivr auth refresh` to rotate.
- **Untrusted upstream content.** Product titles, descriptions, and review text are
  attacker-controllable. rivr fences them as `‹untrusted›…‹/untrusted›` by default
  (`--no-wrap-untrusted` to disable) to reduce prompt-injection risk for the driving agent.
- **Read-only.** rivr performs no mutations and no purchases; the worst-case blast radius of a
  compromised invocation is data disclosure within the configured provider's scope, not state
  change on an Amazon account.

## If a token leaks

1. **Revoke/rotate at the provider** (SerpApi / Rainforest dashboard, or Associates Central for
   Creators) — `rivr auth logout` only removes the **local** copy, it does not revoke server-side.
2. Re-run `rivr auth login` with the new credential.
3. Never paste a real key into an issue; redact it and rotate first.
