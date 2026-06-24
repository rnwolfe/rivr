package provider

import (
	"context"
	"os"

	"github.com/rnwolfe/rivr/internal/errs"
)

// scrape is the DIY amazon.com scraping backend — an ADVANCED, OFF-BY-DEFAULT option.
// Scraping Amazon violates its Conditions of Use, faces aggressive 2026 bot detection
// (AWS WAF + TLS fingerprinting; a March-2026 update reportedly targets agentic crawlers),
// and breaks constantly (per-session A/B DOMs). rivr does NOT ship scraping selectors:
// shipping a fragile, ToS-violating scraper as a default would be irresponsible.
//
// The backend is registered (so the persistent throttle/circuit-breaker in common.go is
// wired and ready) but every operation returns a structured, opt-in-gated error unless the
// user explicitly enables it via RIVR_SCRAPE_ENABLE=1. Even when enabled, selectors are a
// no-op placeholder — the path exists for an operator who knowingly accepts the risk and
// supplies their own implementation, protected by the throttle.
type scrape struct{}

func newScrape() *scrape { return &scrape{} }

func (s *scrape) Name() string { return "scrape" }

func (s *scrape) Capabilities() []string {
	return []string{CapSearch, CapItem, CapOffers, CapReviews, CapVariations}
}

// Configured is true only when the user has explicitly opted in.
func (s *scrape) Configured() bool { return scrapeEnabled() }

// UnconfiguredErr explains the opt-in instead of a misleading "pipe an API key" message.
func (s *scrape) UnconfiguredErr() error { return s.disabled() }

func scrapeEnabled() bool {
	return os.Getenv("RIVR_SCRAPE_ENABLE") == "1"
}

func (s *scrape) disabled() *errs.CLIError {
	if !scrapeEnabled() {
		return errs.New(errs.ExitConfig, "SCRAPE_DISABLED",
			"the scrape backend is disabled (violates Amazon ToS; aggressive bot detection)",
			"prefer an API backend (--provider serpapi). To knowingly opt in, set RIVR_SCRAPE_ENABLE=1 — you accept the ToS/breakage risk and must supply selectors.")
	}
	return errs.New(errs.ExitUnsupported, "SCRAPE_NOT_IMPLEMENTED",
		"scrape selectors are intentionally not shipped",
		"implement selectors in internal/provider/scrape.go; the persistent throttle is already wired.")
}

func (s *scrape) Search(_ context.Context, _ string, _ SearchOpts) (*SearchResult, error) {
	return nil, s.disabled()
}
func (s *scrape) GetItem(_ context.Context, _ string, _ bool) (*Item, error) { return nil, s.disabled() }
func (s *scrape) GetOffers(_ context.Context, _ string) (*OffersResult, error) {
	return nil, s.disabled()
}
func (s *scrape) GetReviews(_ context.Context, _, _ string) (*ReviewsResult, error) {
	return nil, s.disabled()
}
func (s *scrape) GetVariations(_ context.Context, _ string) (*VariationsResult, error) {
	return nil, s.disabled()
}
func (s *scrape) GetBrowseNode(_ context.Context, _ string) (*BrowseNode, error) {
	return nil, s.disabled()
}
