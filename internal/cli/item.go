package cli

import (
	"github.com/rnwolfe/amzn/internal/errs"
	"github.com/rnwolfe/amzn/internal/provider"
)

// ItemCmd groups product-detail read subcommands.
type ItemCmd struct {
	Get    ItemGetCmd    `cmd:"" help:"Get full product detail for one or more ASINs."`
	Offers ItemOffersCmd `cmd:"" help:"Get live offers / buybox / availability for an ASIN."`
}

// ItemGetCmd implements `amzn item get <asin...>` (read).
type ItemGetCmd struct {
	ASIN []string `arg:"" help:"One or more product ASINs."`
}

func (c *ItemGetCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapItem) {
		return errs.Unsupported("item", p.Name())
	}
	items := make([]*provider.Item, 0, len(c.ASIN))
	for _, asin := range c.ASIN {
		it, err := p.GetItem(rt.Ctx, asin, rt.Cfg.Detailed)
		if err != nil {
			return err
		}
		// Fence attacker-controllable free text (contract §8).
		it.Title = rt.Fence(it.Title)
		it.Description = rt.Fence(it.Description)
		it.Features = rt.FenceAll(it.Features)
		it.URL = rt.Link(it.URL)
		items = append(items, it)
	}
	if len(items) == 1 {
		return rt.Out.Emit(items[0])
	}
	return rt.Out.Emit(items)
}

// ItemOffersCmd implements `amzn item offers <asin>` (read).
type ItemOffersCmd struct {
	ASIN string `arg:"" help:"Product ASIN."`
}

func (c *ItemOffersCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapOffers) {
		return errs.Unsupported("offers", p.Name())
	}
	res, err := p.GetOffers(rt.Ctx, c.ASIN)
	if err != nil {
		return err
	}
	return rt.Out.Emit(res)
}
