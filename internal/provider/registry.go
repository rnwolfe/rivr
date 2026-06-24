package provider

import (
	"os"
	"sort"
)

// registry holds the available backends by name. cli-implement registers the real
// providers (serpapi, rainforest, creators) here; the scaffold ships only "stub".
var registry = map[string]Provider{
	"stub": &stub{},
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
// layer; this is the fallback chain when no flag is given (RIVR_PROVIDER env, else stub).
// cli-implement will point the bare default at "serpapi" once it is registered.
func DefaultName() string {
	if p := os.Getenv("RIVR_PROVIDER"); p != "" {
		return p
	}
	return "stub"
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
