package cli

import (
	"github.com/rnwolfe/amzn/internal/errs"
	"github.com/rnwolfe/amzn/internal/provider"
)

// SearchCmd implements `amzn search <query>` (read).
type SearchCmd struct {
	Query     string  `arg:"" help:"Search keywords."`
	Category  string  `help:"Restrict to a category / browse node."`
	MinRating float64 `name:"min-rating" help:"Minimum average star rating (0-5)."`
	Prime     bool    `help:"Only Prime-eligible offers."`
	MinPrice  float64 `name:"min-price" help:"Minimum price."`
	MaxPrice  float64 `name:"max-price" help:"Maximum price."`
	Sort      string  `help:"Sort order (provider-defined, e.g. relevance, price-asc, rating)."`
	Cursor    string  `help:"Opaque pagination cursor from a previous nextCursor."`
}

func (c *SearchCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapSearch) {
		return errs.Unsupported("search", p.Name())
	}
	res, err := p.Search(rt.Ctx, c.Query, provider.SearchOpts{
		Category: c.Category, MinRating: c.MinRating, Prime: c.Prime,
		MinPrice: c.MinPrice, MaxPrice: c.MaxPrice, Sort: c.Sort,
		Limit: rt.Cfg.Limit, Cursor: c.Cursor,
	})
	if err != nil {
		return err
	}
	if len(res.Items) == 0 {
		return errs.Empty("products")
	}
	// Bound output (contract §6) and fence attacker-controllable titles (contract §8).
	if rt.Cfg.Limit > 0 && len(res.Items) > rt.Cfg.Limit {
		rt.Out.Info("note: %d results truncated to --limit=%d (page with --cursor)", len(res.Items), rt.Cfg.Limit)
		res.Items = res.Items[:rt.Cfg.Limit]
	}
	for i := range res.Items {
		res.Items[i].Title = rt.Fence(res.Items[i].Title)
	}
	res.Count = len(res.Items)
	return rt.Out.Emit(res)
}
