package cli

import (
	"io"
	"strings"

	"github.com/rnwolfe/rivr/internal/auth"
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/provider"
)

// AuthCmd groups credential management. Secrets are read from STDIN, never argv
// (contract §7), and stored via internal/auth (env → OS keyring → 0600 file fallback).
type AuthCmd struct {
	Status  AuthStatusCmd  `cmd:"" help:"Test credentials and show auth status for the active provider."`
	Login   AuthLoginCmd   `cmd:"" help:"Store a provider credential (read from stdin)."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove LOCAL credentials for a provider."`
	Refresh AuthRefreshCmd `cmd:"" help:"Refresh the official backend's OAuth token."`
}

// selectedProvider resolves the provider a credential command targets.
func (rt *Runtime) selectedProvider() string {
	if rt.Cfg.Provider != "" {
		return rt.Cfg.Provider
	}
	return provider.DefaultName()
}

func stdinLines(r io.Reader) []string {
	b, _ := io.ReadAll(r)
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		if s := strings.TrimSpace(ln); s != "" {
			out = append(out, s)
		}
	}
	return out
}

type AuthLoginCmd struct{}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	name := rt.selectedProvider()
	if _, ok := provider.Select(name); !ok {
		return errs.New(errs.ExitConfig, "PROVIDER_UNKNOWN", "unknown provider "+name,
			"run `rivr provider list` to see backends")
	}
	lines := stdinLines(rt.Stdin)
	if len(lines) == 0 {
		if rt.Cfg.NoInput {
			return errs.InputRequired("credential on stdin")
		}
		return errs.New(errs.ExitUsage, "USAGE", "no credential on stdin",
			"pipe the secret, e.g. `printf %s \"$KEY\" | rivr auth login --provider "+name+"`")
	}

	if name == "creators" {
		// Official backend needs client_id + client_secret (two lines, id first).
		if len(lines) < 2 {
			return errs.New(errs.ExitUsage, "USAGE",
				"creators needs two lines on stdin: client_id then client_secret",
				"printf '%s\\n%s' \"$CLIENT_ID\" \"$CLIENT_SECRET\" | rivr auth login --provider creators")
		}
		if err := auth.Set(name, auth.FieldClientID, lines[0]); err != nil {
			return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check keyring/file perms")
		}
		if err := auth.Set(name, auth.FieldClientSecret, lines[1]); err != nil {
			return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check keyring/file perms")
		}
	} else {
		if err := auth.Set(name, auth.FieldAPIKey, lines[0]); err != nil {
			return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check keyring/file perms")
		}
	}
	rt.Out.Info("stored credential for %q in %s; verify with `rivr auth status --provider %s`", name, auth.Backend(), name)
	return rt.Out.Emit(map[string]any{"provider": name, "stored": true, "backend": auth.Backend()})
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	name := rt.selectedProvider()
	p, ok := provider.Select(name)
	if !ok {
		return errs.New(errs.ExitConfig, "PROVIDER_UNKNOWN", "unknown provider "+name,
			"run `rivr provider list`")
	}

	all := make([]map[string]any, 0)
	for _, q := range provider.All() {
		all = append(all, map[string]any{"provider": q.Name(), "configured": q.Configured()})
	}

	active := map[string]any{"provider": name, "configured": p.Configured(), "backend": auth.Backend()}
	out := map[string]any{"active": active, "providers": all}

	if !p.Configured() {
		_ = rt.Out.Emit(out)
		return errs.AuthRequired(name) // exit non-zero on problems (contract §7)
	}
	// Actively TEST auth (skip during cooldown so we don't deepen a block). Token redacted —
	// we only report validity, never the secret.
	if v, vok := provider.ValidatorFor(p); vok {
		if cd := provider.Cooldown(name); cd > 0 {
			active["valid"] = nil
			active["note"] = "skipped live check: provider in cooldown"
		} else if err := v.Validate(rt.Ctx); err != nil {
			active["valid"] = false
			active["error"] = err.Error()
			_ = rt.Out.Emit(out)
			return err
		} else {
			active["valid"] = true
		}
	}
	return rt.Out.Emit(out)
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	name := rt.selectedProvider()
	if err := auth.Delete(name); err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check keyring/file perms")
	}
	rt.Out.Info("removed LOCAL credentials for %q (no server-side revocation)", name)
	return rt.Out.Emit(map[string]any{"provider": name, "ok": true})
}

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	name := rt.selectedProvider()
	p, ok := provider.Select(name)
	if !ok {
		return errs.New(errs.ExitConfig, "PROVIDER_UNKNOWN", "unknown provider "+name, "run `rivr provider list`")
	}
	r, ok := provider.RefresherFor(p)
	if !ok {
		return rt.Out.Emit(map[string]any{"provider": name, "refreshed": false,
			"note": "nothing to refresh (only the OAuth-based official backend has tokens)"})
	}
	if err := r.Refresh(rt.Ctx); err != nil {
		return err
	}
	return rt.Out.Emit(map[string]any{"provider": name, "refreshed": true})
}
