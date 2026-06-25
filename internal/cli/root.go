// Package cli wires the kong grammar, the runtime, and the exit-code mapping.
// main() does nothing but os.Exit(cli.Run(...)) so every path is testable in-process.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/fence"
	"github.com/rnwolfe/rivr/internal/output"
	"github.com/rnwolfe/rivr/internal/provider"
	"github.com/rnwolfe/rivr/internal/update"
	"github.com/rnwolfe/rivr/internal/version"
)

// CLI is the kong grammar. Global flags are the universal agent-CLI contract surface plus
// rivr-specific --provider and --wrap-untrusted; subcommands follow noun-verb grammar.
type CLI struct {
	// Output (contract §1, §6)
	Format   string `enum:"json,plain,tsv" default:"plain" help:"Output format: json, plain, or tsv."`
	JSON     bool   `help:"Shorthand for --format=json."`
	NoColor  bool   `help:"Disable colored output."`
	Limit    int    `default:"50" help:"Maximum items to return for list operations."`
	Select   string `help:"Comma-separated dot-path field projection, e.g. asin,title,price."`
	Concise  bool   `help:"Terser output (default)."`
	Detailed bool   `help:"Richer output."`

	// Backend (rivr-specific)
	Provider       string `help:"Backend provider (e.g. serpapi, rainforest, creators). Overrides the configured default." env:"RIVR_PROVIDER"`
	AssociateTag   string `name:"associate-tag" help:"Use your own Amazon Associates tag on product deep links (replaces the built-in default)." env:"RIVR_ASSOCIATE_TAG"`
	NoAssociateTag bool   `name:"no-associate-tag" help:"Disable affiliate attribution on deep links (turns off the built-in tag that funds rivr)." env:"RIVR_NO_ASSOCIATE_TAG"`

	// Prompt-injection hardening (contract §8) — ON by default; --no-wrap-untrusted to disable.
	WrapUntrusted bool `default:"true" negatable:"" help:"Fence attacker-controllable free text (titles, descriptions, reviews) as untrusted."`

	// Safety (contract §2). rivr is read-only: these are present for contract uniformity
	// but inert — no command performs a mutation.
	AllowMutations bool `help:"Permit state-changing operations (no-op: rivr is read-only)."`
	DryRun         bool `help:"Print intended mutations without performing them (no-op: rivr is read-only)."`
	Yes            bool `help:"Assume yes for confirmations (no-op: rivr is read-only)."`
	Force          bool `help:"Bypass safety checks (no-op: rivr is read-only)."`
	NoInput        bool `help:"Never prompt; fail with exit 13 instead."`

	// Commands (all reads)
	Search     SearchCmd     `cmd:"" help:"Search Amazon products by keyword/category."`
	Item       ItemCmd       `cmd:"" help:"Get product detail and offers."`
	Reviews    ReviewsCmd    `cmd:"" help:"Get customer reviews for a product (third-party providers only)."`
	Variations VariationsCmd `cmd:"" help:"List size/color/style variations of a product."`
	Browse     BrowseCmd     `cmd:"" help:"Inspect the browse-node (category) tree."`
	ProviderC  ProviderCmd   `cmd:"" name:"provider" help:"List and inspect configured backends."`
	Auth       AuthCmd       `cmd:"" help:"Manage provider credentials."`
	Doctor     DoctorCmd     `cmd:"" help:"Diagnose setup and report fixes."`
	Schema     SchemaCmd     `cmd:"" help:"Print the machine-readable command schema (JSON)."`
	Agent      AgentCmd      `cmd:"" help:"Print the bundled agent SKILL.md."`
	Version    VersionCmd    `cmd:"" help:"Print the version."`
}

// Runtime is the per-invocation context bound into every command's Run method.
type Runtime struct {
	Cfg          *CLI
	Out          *output.Writer
	Stdin        io.Reader
	Ctx          context.Context
	warnedOptOut bool // ensures the affiliate opt-out notice prints at most once per run
}

// Provider resolves the active backend (flag > RIVR_PROVIDER > default), erroring with a
// structured config/auth error when unknown or unconfigured.
func (rt *Runtime) Provider() (provider.Provider, error) {
	p, ok := provider.Select(rt.Cfg.Provider)
	if !ok {
		name := rt.Cfg.Provider
		if name == "" {
			name = provider.DefaultName()
		}
		return nil, errs.New(errs.ExitConfig, "PROVIDER_UNKNOWN",
			"unknown provider "+name, "run `rivr provider list` to see configured backends")
	}
	if !p.Configured() {
		if u, ok := p.(provider.Unconfigured); ok {
			return nil, u.UnconfiguredErr() // backend-specific guidance (e.g. scrape opt-in)
		}
		e := errs.AuthRequired(p.Name())
		// If the keyless scraper is opted-in but a credentialed default was selected, the
		// agent almost certainly forgot `--provider scrape` — point them at it.
		if os.Getenv("RIVR_SCRAPE_ENABLE") == "1" && p.Name() != "scrape" {
			e.Remediation += "; or use the keyless backend you enabled: --provider scrape"
		}
		return nil, e
	}
	// Inject the resolved Associates tag into backends that need it (official Creators API
	// requires partnerTag on every request).
	if ta, ok := p.(provider.TagAware); ok {
		ta.SetPartnerTag(rt.PartnerTag())
	}
	return p, nil
}

// Guard enforces the read-only-by-default mutation gate (contract §2). rivr has no
// mutating commands, so this is never reached at runtime — kept for contract uniformity.
func (rt *Runtime) Guard(op string) error {
	if rt.Cfg.AllowMutations {
		return nil
	}
	return errs.MutationBlocked(op)
}

// Fence wraps a free-text value as untrusted when --wrap-untrusted is on (contract §8).
func (rt *Runtime) Fence(s string) string {
	if rt.Cfg.WrapUntrusted {
		return fence.Wrap(s)
	}
	return s
}

// FenceAll fences a slice of free-text values.
func (rt *Runtime) FenceAll(ss []string) []string {
	if rt.Cfg.WrapUntrusted {
		return fence.WrapAll(ss)
	}
	return ss
}

// Link decorates a product deep link with the active Associates tag. The CLI is read-only;
// the deep link is its terminal hand-off to amazon.com (purchase happens in the user's
// browser session). By default the built-in project tag is used (see affiliate.go); a user
// tag replaces it, and --no-associate-tag disables it (with a one-time stderr notice).
// Non-product URLs (e.g. images) are not passed through here.
func (rt *Runtime) Link(u string) string {
	if u == "" {
		return u
	}
	tag, mode := rt.resolveAssociateTag()
	if mode == affiliateDisabled {
		if !rt.warnedOptOut {
			rt.Out.Info(optOutNotice)
			rt.warnedOptOut = true
		}
		return u
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + "tag=" + url.QueryEscape(tag)
}

// rootHelp leads with runnable examples (contract §5: example-led help), then the synopsis.
const rootHelp = `An agent-friendly CLI for Amazon Shopping search and data retrieval (read-only).

EXAMPLES:
  # one-time auth (key read from stdin, never argv); default backend is serpapi
  printf %s "$SERPAPI_KEY" | rivr auth login --provider serpapi

  rivr search "usb-c cable" --json
  rivr search "coffee maker" --prime --min-rating 4 --json
  rivr item get B0XXXXXXX --detailed --json
  rivr item offers B0XXXXXXX --json
  rivr reviews B0XXXXXXX --provider rainforest --json   # full reviews
  rivr provider list --json
  rivr doctor --json

Read-only: every result is structured data plus a canonical /dp/<ASIN> deep link.
See 'rivr agent' for the full agent contract, 'rivr schema' for the machine-readable spec.`

// Run parses args and dispatches, returning the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// `--version`/`-V` is the most common first probe; honor it as a flag (not just the
	// `version` subcommand) before kong's required-subcommand parsing rejects it.
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "--version" || a == "-V" {
			fmt.Fprintln(stdout, version.String())
			return errs.ExitOK
		}
	}

	var cfg CLI
	helpShown := false
	parser, err := kong.New(&cfg,
		kong.Name("rivr"),
		kong.Description(rootHelp),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) { helpShown = true }), // --help/--version: we control exit
	)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return errs.ExitGeneric
	}

	kctx, perr := parser.Parse(args)
	if helpShown {
		return errs.ExitOK
	}
	if perr != nil {
		return handleParseError(stderr, args, perr)
	}

	if cfg.JSON {
		cfg.Format = "json"
	}
	rt := newRuntime(&cfg, stdin, stdout, stderr)

	if err := kctx.Run(rt); err != nil {
		return emitError(rt, err)
	}
	maybeNotifyUpdate(rt)
	return errs.ExitOK
}

// maybeNotifyUpdate prints a one-line upgrade notice to STDERR — but only on the human path
// (interactive TTY, plain format), never for agents (JSON/non-TTY/--no-input) and never if
// RIVR_NO_UPDATE_CHECK=1. It uses the 24h cache, so it costs nothing on most invocations and
// at most one ~2s GitHub call per day, AFTER the command's output is already written.
func maybeNotifyUpdate(rt *Runtime) {
	if rt.Out.Format != output.FormatPlain || rt.Cfg.NoInput ||
		os.Getenv("RIVR_NO_UPDATE_CHECK") == "1" || !isTTY(rt.Out.Stdout) {
		return
	}
	latest, avail, err := update.Check(rt.Ctx, version.String(), false, time.Now())
	if err != nil || !avail {
		return
	}
	rt.Out.Info("note: a new rivr release is available (%s → %s). Upgrade: %s",
		version.String(), latest, update.UpgradeHint)
}

func newRuntime(cfg *CLI, stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	format := output.Format(cfg.Format)
	color := !cfg.NoColor && os.Getenv("NO_COLOR") == "" && isTTY(stdout) && format == output.FormatPlain
	var sel []string
	if cfg.Select != "" {
		sel = strings.Split(cfg.Select, ",")
	}
	w := &output.Writer{
		Stdout: stdout, Stderr: stderr,
		Format: format, Color: color, Limit: cfg.Limit, Select: sel,
	}
	// Conflicting affiliate flags: --no-associate-tag wins; say so rather than resolve silently.
	if cfg.AssociateTag != "" && cfg.NoAssociateTag {
		w.Info("note: both --associate-tag and --no-associate-tag set; --no-associate-tag wins (attribution disabled)")
	}
	return &Runtime{Cfg: cfg, Out: w, Stdin: stdin, Ctx: context.Background()}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// emitError prints a structured error to stderr and returns its exit code (contract §3).
func emitError(rt *Runtime, err error) int {
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		ce = errs.New(errs.ExitGeneric, "INTERNAL", err.Error(), "")
	}
	if rt.Out.Format == output.FormatJSON {
		body := map[string]any{
			"error":       ce.Message,
			"code":        ce.Code,
			"remediation": ce.Remediation,
		}
		if ce.RetryAfter > 0 {
			body["retryAfterSeconds"] = ce.RetryAfter
		}
		enc := json.NewEncoder(rt.Out.Stderr)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(body)
	} else {
		fmt.Fprintf(rt.Out.Stderr, "error: %s\n", ce.Message)
		if ce.Code != "" {
			fmt.Fprintf(rt.Out.Stderr, "  code: %s\n", ce.Code)
		}
		if ce.RetryAfter > 0 {
			fmt.Fprintf(rt.Out.Stderr, "  retry after: %ds\n", ce.RetryAfter)
		}
		if ce.Remediation != "" {
			fmt.Fprintf(rt.Out.Stderr, "  fix:  %s\n", ce.Remediation)
		}
	}
	return ce.Exit
}

// handleParseError reports usage errors and offers a "did you mean" suggestion.
// kong already suggests for some cases; only add ours when it didn't, to avoid a dupe line.
func handleParseError(stderr io.Writer, args []string, err error) int {
	fmt.Fprintf(stderr, "error: %s\n", err)
	if strings.Contains(err.Error(), "did you mean") {
		return errs.ExitUsage
	}
	commands := []string{"search", "item", "reviews", "variations", "browse", "provider", "auth", "doctor", "schema", "agent", "version"}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if s, ok := closest(a, commands); ok {
			fmt.Fprintf(stderr, "  did you mean %q?\n", s)
		}
		break
	}
	return errs.ExitUsage
}
