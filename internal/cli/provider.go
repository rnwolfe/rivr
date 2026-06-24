package cli

import "github.com/rnwolfe/rivr/internal/provider"

// ProviderCmd groups backend-management read subcommands.
type ProviderCmd struct {
	List ProviderListCmd `cmd:"" help:"List configured backends and their capabilities."`
}

// ProviderListCmd implements `rivr provider list` (read).
type ProviderListCmd struct{}

func (c *ProviderListCmd) Run(rt *Runtime) error {
	def := provider.DefaultName()
	if rt.Cfg.Provider != "" {
		def = rt.Cfg.Provider
	}
	rows := make([]map[string]any, 0)
	for _, p := range provider.All() {
		rows = append(rows, map[string]any{
			"name":         p.Name(),
			"configured":   p.Configured(),
			"default":      p.Name() == def,
			"capabilities": p.Capabilities(),
		})
	}
	return rt.Out.Emit(rows)
}
