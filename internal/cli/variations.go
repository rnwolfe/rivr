package cli

import (
	"github.com/rnwolfe/amzn/internal/errs"
	"github.com/rnwolfe/amzn/internal/provider"
)

// VariationsCmd implements `amzn variations <asin>` (read).
type VariationsCmd struct {
	ASIN string `arg:"" help:"Parent product ASIN."`
}

func (c *VariationsCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapVariations) {
		return errs.Unsupported("variations", p.Name())
	}
	res, err := p.GetVariations(rt.Ctx, c.ASIN)
	if err != nil {
		return err
	}
	if len(res.Variations) == 0 {
		return errs.Empty("variations")
	}
	for i := range res.Variations {
		res.Variations[i].URL = rt.Link(res.Variations[i].URL)
	}
	return rt.Out.Emit(res)
}
