package cli

import (
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/provider"
)

// ItemCmd groups product-detail read subcommands.
type ItemCmd struct {
	Get     ItemGetCmd     `cmd:"" help:"Get full product detail for one or more ASINs."`
	Offers  ItemOffersCmd  `cmd:"" help:"Get live offers / buybox / availability for an ASIN."`
	Compare ItemCompareCmd `cmd:"" help:"Compare two or more ASINs side-by-side with a best-of summary."`
}

// ItemGetCmd implements `rivr item get <asin...>` (read).
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

// ItemCompareCmd implements `rivr item compare <asin...>` (read): fetch each ASIN and
// return them alongside a "best-of" summary (cheapest / highest-rated / most-reviewed), so
// an agent doesn't have to hand-assemble a comparison from N separate `item get` calls.
type ItemCompareCmd struct {
	ASIN []string `arg:"" help:"Two or more product ASINs to compare."`
}

func (c *ItemCompareCmd) Run(rt *Runtime) error {
	if len(c.ASIN) < 2 {
		return errs.New(errs.ExitUsage, "USAGE", "compare needs at least two ASINs",
			"rivr item compare B0AAA B0BBB --json")
	}
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapItem) {
		return errs.Unsupported("item", p.Name())
	}
	items := make([]*provider.Item, 0, len(c.ASIN))
	var cheapest, highestRated, mostReviewed *provider.Item
	for _, asin := range c.ASIN {
		it, err := p.GetItem(rt.Ctx, asin, rt.Cfg.Detailed)
		if err != nil {
			return err
		}
		// Compute the summary on raw values BEFORE fencing/decorating.
		if it.Price > 0 && (cheapest == nil || it.Price < cheapest.Price) {
			cheapest = it
		}
		if highestRated == nil || it.Rating > highestRated.Rating {
			highestRated = it
		}
		if mostReviewed == nil || it.ReviewCount > mostReviewed.ReviewCount {
			mostReviewed = it
		}
		it.Title = rt.Fence(it.Title)
		it.Description = rt.Fence(it.Description)
		it.Features = rt.FenceAll(it.Features)
		it.URL = rt.Link(it.URL)
		items = append(items, it)
	}
	asinOf := func(it *provider.Item) string {
		if it == nil {
			return ""
		}
		return it.ASIN
	}
	return rt.Out.Emit(map[string]any{
		"schemaVersion": provider.SchemaVersion,
		"provider":      p.Name(),
		"items":         items,
		"summary": map[string]any{
			"cheapest":     asinOf(cheapest),
			"highestRated": asinOf(highestRated),
			"mostReviewed": asinOf(mostReviewed),
		},
	})
}

// ItemOffersCmd implements `rivr item offers <asin>` (read).
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
