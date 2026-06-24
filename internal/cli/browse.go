package cli

import (
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/provider"
)

// BrowseCmd implements `rivr browse <node-id>` (read). Browse nodes are an official
// Creators-API feature; providers that lack it return UNSUPPORTED_BY_PROVIDER.
type BrowseCmd struct {
	NodeID string `arg:"" name:"node-id" help:"Browse-node id."`
}

func (c *BrowseCmd) Run(rt *Runtime) error {
	p, err := rt.Provider()
	if err != nil {
		return err
	}
	if !provider.Supports(p, provider.CapBrowse) {
		return errs.Unsupported("browse", p.Name())
	}
	res, err := p.GetBrowseNode(rt.Ctx, c.NodeID)
	if err != nil {
		return err
	}
	return rt.Out.Emit(res)
}
