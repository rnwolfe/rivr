package cli

import (
	"io"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/amzn/internal/errs"
	"github.com/rnwolfe/amzn/internal/provider"
	"github.com/rnwolfe/amzn/internal/skill"
	"github.com/rnwolfe/amzn/internal/version"
)

// --- auth -------------------------------------------------------------------
// Credentials are per-provider API keys (third-party) or client-id+secret (official
// Creators). Keys are read from STDIN, never argv (contract §7). cli-implement persists
// them to the OS keyring (OS-native backend only, env -> keyring -> 0600 file fallback);
// the scaffold leaves storage as a documented placeholder.

type AuthCmd struct {
	Status AuthStatusCmd `cmd:"" help:"Show authentication status for configured providers."`
	Login  AuthLoginCmd  `cmd:"" help:"Store a provider credential (read from stdin)."`
	Logout AuthLogoutCmd `cmd:"" help:"Remove a provider credential."`
	Refresh AuthRefreshCmd `cmd:"" help:"Refresh a provider token (official OAuth backend)."`
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	rows := make([]map[string]any, 0)
	for _, p := range provider.All() {
		rows = append(rows, map[string]any{
			"provider":      p.Name(),
			"authenticated": p.Configured(),
		})
	}
	return rt.Out.Emit(map[string]any{"providers": rows})
}

type AuthLoginCmd struct{}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	name := rt.Cfg.Provider
	if name == "" {
		name = provider.DefaultName()
	}
	// Read the secret from stdin (never argv). --no-input with no piped data hard-fails.
	b, _ := io.ReadAll(rt.Stdin)
	secret := strings.TrimSpace(string(b))
	if secret == "" {
		if rt.Cfg.NoInput {
			return errs.InputRequired("credential on stdin")
		}
		return errs.New(errs.ExitUsage, "USAGE", "no credential on stdin",
			"pipe the key, e.g. `printf %s \"$KEY\" | amzn auth login --provider "+name+"`")
	}
	// PLACEHOLDER: cli-implement persists `secret` to the OS keyring for `name`.
	rt.Out.Info("received credential for provider %q (%d bytes); keyring storage is wired by cli-implement", name, len(secret))
	return rt.Out.Emit(map[string]any{"provider": name, "stored": false, "note": "keyring wiring pending (cli-implement)"})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	name := rt.Cfg.Provider
	if name == "" {
		name = provider.DefaultName()
	}
	return rt.Out.Emit(map[string]any{"provider": name, "ok": true})
}

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	name := rt.Cfg.Provider
	if name == "" {
		name = provider.DefaultName()
	}
	// PLACEHOLDER: only meaningful for the official Creators OAuth client-credentials backend.
	return rt.Out.Emit(map[string]any{"provider": name, "refreshed": false, "note": "token refresh wired by cli-implement (official backend only)"})
}

// --- doctor -----------------------------------------------------------------

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	def := provider.DefaultName()
	if rt.Cfg.Provider != "" {
		def = rt.Cfg.Provider
	}
	_, known := provider.Select(def)
	var defConfigured bool
	if p, ok := provider.Select(def); ok {
		defConfigured = p.Configured()
	}
	checks := []map[string]any{
		{"name": "default_provider", "ok": known, "detail": "resolved default: " + def},
		{"name": "credentials", "ok": defConfigured, "detail": "default provider configured"},
		{"name": "wrap_untrusted", "ok": rt.Cfg.WrapUntrusted, "detail": "prompt-injection fencing enabled"},
	}
	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	if !allOK {
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed", "see the failing check's detail")
	}
	return rt.Out.Emit(map[string]any{"ok": true, "checks": checks})
}

// --- schema -----------------------------------------------------------------

type SchemaCmd struct{}

func (c *SchemaCmd) Run(rt *Runtime) error {
	k, err := kong.New(&CLI{}, kong.Name("amzn"))
	if err != nil {
		return errs.New(errs.ExitGeneric, "SCHEMA_ERROR", err.Error(), "")
	}
	out := map[string]any{
		"tool":          "amzn",
		"version":       version.String(),
		"schemaVersion": provider.SchemaVersion,
		"commands":      nodeToMap(k.Model.Node),
		"exit_codes":    errs.Table(),
		"providers":     provider.Names(),
		"safety": map[string]any{
			"read_only":       true,
			"allow_mutations": rt.Cfg.AllowMutations,
			"dry_run":         rt.Cfg.DryRun,
			"no_input":        rt.Cfg.NoInput,
			"wrap_untrusted":  rt.Cfg.WrapUntrusted,
		},
	}
	return rt.Out.EmitJSON(out) // schema is always JSON
}

func nodeToMap(n *kong.Node) map[string]any {
	m := map[string]any{"name": n.Name}
	if n.Help != "" {
		m["help"] = n.Help
	}
	var flags []map[string]any
	for _, f := range n.Flags {
		if f.Name == "help" {
			continue
		}
		fm := map[string]any{"name": f.Name}
		if f.Help != "" {
			fm["help"] = f.Help
		}
		if f.Default != "" {
			fm["default"] = f.Default
		}
		flags = append(flags, fm)
	}
	if len(flags) > 0 {
		m["flags"] = flags
	}
	var args []map[string]any
	for _, p := range n.Positional {
		args = append(args, map[string]any{"name": p.Name, "help": p.Help})
	}
	if len(args) > 0 {
		m["args"] = args
	}
	var subs []any
	for _, ch := range n.Children {
		subs = append(subs, nodeToMap(ch))
	}
	if len(subs) > 0 {
		m["subcommands"] = subs
	}
	return m
}

// --- agent ------------------------------------------------------------------

type AgentCmd struct{}

func (c *AgentCmd) Run(rt *Runtime) error {
	_, err := rt.Out.Stdout.Write([]byte(skill.Content))
	return err
}

// --- version ----------------------------------------------------------------

type VersionCmd struct{}

func (c *VersionCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{"version": version.String()})
}
