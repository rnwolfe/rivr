package cli

import (
	"github.com/alecthomas/kong"

	"github.com/rnwolfe/rivr/internal/auth"
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/provider"
	"github.com/rnwolfe/rivr/internal/skill"
	"github.com/rnwolfe/rivr/internal/version"
)

// auth commands live in auth.go.

// --- doctor -----------------------------------------------------------------

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	def := provider.DefaultName()
	if rt.Cfg.Provider != "" {
		def = rt.Cfg.Provider
	}
	p, known := provider.Select(def)
	configured := known && p.Configured()

	checks := []map[string]any{
		{"name": "default_provider", "ok": known, "detail": "active provider: " + def},
		{"name": "credentials", "ok": configured, "detail": credDetail(def, configured)},
	}

	// Connectivity / credential validity — but never deepen an active block.
	if configured {
		if cd := provider.Cooldown(def); cd > 0 {
			checks = append(checks, map[string]any{"name": "connectivity", "ok": true,
				"detail": "skipped: provider in cooldown (" + itoa(cd) + "s); not probing"})
		} else if v, ok := provider.ValidatorFor(p); ok {
			if err := v.Validate(rt.Ctx); err != nil {
				checks = append(checks, map[string]any{"name": "connectivity", "ok": false, "detail": err.Error()})
			} else {
				checks = append(checks, map[string]any{"name": "connectivity", "ok": true, "detail": "reachable; credentials valid"})
			}
		}
	}

	if insecure, fix := auth.InsecureFilePerms(); insecure {
		checks = append(checks, map[string]any{"name": "secret_perms", "ok": false, "detail": fix})
	}

	tag, mode := rt.resolveAssociateTag()
	affDetail := "tag=" + tag + " (" + string(mode) + ")"
	if mode == affiliateDisabled {
		affDetail = "disabled (no affiliate attribution)"
	}
	checks = append(checks,
		map[string]any{"name": "wrap_untrusted", "ok": rt.Cfg.WrapUntrusted, "detail": "prompt-injection fencing enabled"},
		map[string]any{"name": "affiliate", "ok": true, "detail": affDetail},
	)

	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	out := map[string]any{"ok": allOK, "provider": def, "secret_backend": auth.Backend(), "checks": checks}
	if !allOK {
		// doctor reports findings on stdout AND signals failure via exit code.
		_ = rt.Out.Emit(out)
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed", "see each failing check's detail")
	}
	return rt.Out.Emit(out)
}

func credDetail(provider string, configured bool) string {
	if configured {
		return provider + " credentials present (" + auth.Backend() + ")"
	}
	return "no credentials; run `rivr auth login --provider " + provider + "`"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// --- schema -----------------------------------------------------------------

type SchemaCmd struct{}

func (c *SchemaCmd) Run(rt *Runtime) error {
	k, err := kong.New(&CLI{}, kong.Name("rivr"))
	if err != nil {
		return errs.New(errs.ExitGeneric, "SCHEMA_ERROR", err.Error(), "")
	}
	out := map[string]any{
		"tool":          "rivr",
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
			"associate_tag":   schemaAssociateTag(rt),
		},
	}
	return rt.Out.EmitJSON(out) // schema is always JSON
}

// schemaAssociateTag reports the active affiliate attribution state for `schema --json`.
func schemaAssociateTag(rt *Runtime) map[string]any {
	tag, mode := rt.resolveAssociateTag()
	return map[string]any{"mode": string(mode), "tag": tag}
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
