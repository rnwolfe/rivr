package provider

// Descriptor is the published, machine-readable profile of a backend: what it can do, what
// it costs, and what it risks. It is the single source of truth behind `rivr provider list`
// AND the docs matrix, so capabilities/cost/risk can never drift between code and docs.
type Descriptor struct {
	Name         string   `json:"name"`
	Summary      string   `json:"summary"`
	Keyless      bool     `json:"keyless"`               // works with no credential?
	Auth         string   `json:"auth"`                  // how you authenticate
	HostedSafe   bool     `json:"hostedSafe"`            // safe from cloud/datacenter IPs?
	Cost         string   `json:"cost"`                  // rough cost (verify current pricing)
	Risk         string   `json:"risk"`                  // headline risk
	ReviewsScope string   `json:"reviewsScope"`          // full | sample | none
	Capabilities []string `json:"capabilities"`          // search/item/offers/reviews/variations/browse
	Official     bool     `json:"official"`              // first-party Amazon API?
}

// Describable backends expose a Descriptor.
type Describable interface {
	Describe() Descriptor
}

// Describe returns a backend's profile, falling back to a minimal one if it doesn't
// implement Describable.
func Describe(p Provider) Descriptor {
	if d, ok := p.(Describable); ok {
		return d.Describe()
	}
	return Descriptor{Name: p.Name(), Capabilities: p.Capabilities()}
}

func (s *serpApi) Describe() Descriptor {
	return Descriptor{
		Name: "serpapi", Summary: "SerpApi Amazon engines (default).",
		Keyless: false, Auth: "API key (stdin → keyring)", HostedSafe: true,
		Cost:         "Free tier ~250 searches/mo (renewing); paid metered plans beyond it",
		Risk:         "third-party dependency; paid beyond free tier; reviews are a page sample, no browse",
		ReviewsScope: "sample", Capabilities: s.Capabilities(), Official: false,
	}
}

func (r *rainforest) Describe() Descriptor {
	return Descriptor{
		Name: "rainforest", Summary: "Rainforest API (Traject Data) — richest third-party data.",
		Keyless: false, Auth: "API key (stdin → keyring)", HostedSafe: true,
		Cost:         "Paid, credit-based (no free tier; small trial); plans from ~$25/mo",
		Risk:         "third-party dependency; paid per request",
		ReviewsScope: "full", Capabilities: r.Capabilities(), Official: false,
	}
}

func (c *creators) Describe() Descriptor {
	return Descriptor{
		Name: "creators", Summary: "Official Amazon Creators API (successor to PA-API 5.0).",
		Keyless: false, Auth: "OAuth2 client-credentials (client_id + secret)", HostedSafe: true,
		Cost:         "Free API, but requires an approved Amazon Associate w/ ≥10 qualifying sales / 30 days",
		Risk:         "eligibility wall (ASSOCIATE_NOT_ELIGIBLE); NO review text; portal access-gated",
		ReviewsScope: "none", Capabilities: c.Capabilities(), Official: true,
	}
}

func (s *scrape) Describe() Descriptor {
	return Descriptor{
		Name: "scrape", Summary: "Keyless amazon.com scraping — opt-in, residential use only.",
		Keyless: true, Auth: "none (opt-in: RIVR_SCRAPE_ENABLE=1)", HostedSafe: false,
		Cost:         "Free (no API key); cost is your bandwidth/IP + selector upkeep",
		Risk:         "Amazon ToS; bot-detection/blocking; fragile DOM; no review text (walled); do NOT use from cloud/hosted IPs",
		ReviewsScope: "none", Capabilities: s.Capabilities(), Official: false,
	}
}

func (s *stub) Describe() Descriptor {
	return Descriptor{
		Name: "stub", Summary: "Deterministic fixtures for offline/testing.",
		Keyless: true, Auth: "none", HostedSafe: true,
		Cost:         "Free",
		Risk:         "NOT real Amazon data — testing/offline only",
		ReviewsScope: "sample", Capabilities: s.Capabilities(), Official: false,
	}
}
