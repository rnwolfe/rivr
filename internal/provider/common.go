package provider

import (
	"strings"
	"time"

	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/throttle"
)

// currencyForDomain maps an amazon_domain to its ISO currency (SerpApi/scrape don't return
// a currency field). Defaults to USD.
func currencyForDomain(domain string) string {
	switch {
	case domain == "" || strings.HasSuffix(domain, "amazon.com"):
		return "USD"
	case strings.HasSuffix(domain, "co.uk"):
		return "GBP"
	case strings.HasSuffix(domain, "co.jp"):
		return "JPY"
	case strings.HasSuffix(domain, ".ca"):
		return "CAD"
	case strings.HasSuffix(domain, ".in"):
		return "INR"
	case strings.HasSuffix(domain, "com.au"):
		return "AUD"
	case strings.HasSuffix(domain, "com.mx"):
		return "MXN"
	case strings.HasSuffix(domain, "com.br"):
		return "BRL"
	case strings.HasSuffix(domain, ".de"), strings.HasSuffix(domain, ".fr"),
		strings.HasSuffix(domain, ".it"), strings.HasSuffix(domain, ".es"),
		strings.HasSuffix(domain, ".nl"):
		return "EUR"
	default:
		return "USD"
	}
}

// preflight fails fast if a persistent cooldown is active for the provider (an agent
// spawns a fresh process per call, so this is the only place a prior block is visible).
func preflight(provider string) error {
	if ra := throttle.Load(provider).RetryAfter(time.Now()); ra > 0 {
		return errs.Blocked(provider, ra)
	}
	return nil
}

// markBlocked records a cooldown so the next process fails fast instead of wasting a call.
func markBlocked(provider string, seconds int) {
	throttle.Block(provider, seconds, time.Now())
}

// markOK clears any stale cooldown after a successful call.
func markOK(provider string) { throttle.Clear(provider) }
