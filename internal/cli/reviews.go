package cli

import (
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/provider"
)

// ReviewsCmd implements `rivr reviews <asin>` (read). Review text is the sharpest
// prompt-injection vector — bodies/titles are always fenced. Not all backends serve it
// (the official Creators API returns no review text → UNSUPPORTED_BY_PROVIDER).
type ReviewsCmd struct {
	ASIN   string `arg:"" help:"Product ASIN."`
	Cursor string `help:"Opaque pagination cursor from a previous nextCursor."`
}

func (c *ReviewsCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapReviews) {
		return errs.Unsupported("reviews", p.Name())
	}
	res, err := p.GetReviews(rt.Ctx, c.ASIN, c.Cursor)
	if err != nil {
		return err
	}
	if len(res.Reviews) == 0 {
		return errs.Empty("reviews")
	}
	if rt.Cfg.Limit > 0 && len(res.Reviews) > rt.Cfg.Limit {
		rt.Out.Info("note: %d reviews truncated to --limit=%d (page with --cursor)", len(res.Reviews), rt.Cfg.Limit)
		res.Reviews = res.Reviews[:rt.Cfg.Limit]
	}
	for i := range res.Reviews {
		res.Reviews[i].Title = rt.Fence(res.Reviews[i].Title)
		res.Reviews[i].Body = rt.Fence(res.Reviews[i].Body)
	}
	return rt.Out.Emit(res)
}
