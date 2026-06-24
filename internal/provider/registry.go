package provider

import (
	"os"
	"sort"
)

// registry holds the available backends by name. "stub" is the offline/test backend; the
// real backends are the third-party providers (serpapi default, rainforest), the official
// Creators API, and the gated scrape provider.
var registry = map[string]Provider{
	"serpapi":    newSerpApi(),
	"rainforest": newRainforest(),
	"creators":   newCreators(),
	"scrape":     newScrape(),
	"stub":       &stub{},
}

// Names returns the registered provider names, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// DefaultName resolves the default backend: --provider flag handling happens in the cli
// layer; this is the fallback chain when no flag is given (RIVR_PROVIDER env, else the
// SerpApi third-party backend — the one with a renewing free tier and review text).
func DefaultName() string {
	if p := os.Getenv("RIVR_PROVIDER"); p != "" {
		return p
	}
	return "serpapi"
}

// Select returns the named provider, or the default when name is empty.
func Select(name string) (Provider, bool) {
	if name == "" {
		name = DefaultName()
	}
	p, ok := registry[name]
	return p, ok
}

// All returns every registered provider (for `provider list`).
func All() []Provider {
	out := make([]Provider, 0, len(registry))
	for _, n := range Names() {
		out = append(out, registry[n])
	}
	return out
}
