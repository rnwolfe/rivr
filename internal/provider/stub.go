package provider

import "context"

// stub is a placeholder backend returning deterministic fixtures. It lets the skeleton
// compile, run, and pass contract tests offline. cli-implement replaces it with real
// backends (SerpApi, Rainforest, official Creators API) that issue HTTP requests.
//
// REPLACE ME: this is the example target, not a real Amazon integration.
type stub struct{}

func (s *stub) Name() string { return "stub" }

func (s *stub) Capabilities() []string {
	return []string{CapSearch, CapItem, CapOffers, CapReviews, CapVariations, CapBrowse}
}

// Configured reports true so the skeleton is runnable with no credentials. Real backends
// return false until `rivr auth login` has stored a key in the keyring.
func (s *stub) Configured() bool { return true }

func (s *stub) Search(_ context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	items := []SearchItem{
		{ASIN: "B0STUB0001", Title: "[stub] " + query + " — Example Product A", Price: 12.99, Currency: "USD", Rating: 4.6, ReviewCount: 21034, Prime: true, URL: "https://www.amazon.com/dp/B0STUB0001", Image: "https://example.invalid/a.jpg"},
		{ASIN: "B0STUB0002", Title: "[stub] " + query + " — Example Product B", Price: 24.50, Currency: "USD", Rating: 4.2, ReviewCount: 813, Prime: false, URL: "https://www.amazon.com/dp/B0STUB0002", Image: "https://example.invalid/b.jpg"},
	}
	return &SearchResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), Query: query,
		Items: items, Count: len(items), Limit: opts.Limit,
	}, nil
}

func (s *stub) GetItem(_ context.Context, asin string, detailed bool) (*Item, error) {
	it := &Item{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin,
		Title: "[stub] Example Product " + asin, Brand: "ExampleBrand",
		Price: 12.99, Currency: "USD",
		Offers:    []Offer{{Price: 12.99, Currency: "USD", Condition: "new", Merchant: "Amazon.com", Prime: true, Availability: "in_stock"}},
		Images:    []string{"https://example.invalid/" + asin + ".jpg"},
		Rating:    4.6, ReviewCount: 21034, SalesRank: 142,
		URL:       "https://www.amazon.com/dp/" + asin,
	}
	if detailed {
		it.Features = []string{"[stub] feature one", "[stub] feature two"}
		it.Description = "[stub] A longer description of the example product."
	}
	return it, nil
}

func (s *stub) GetOffers(_ context.Context, asin string) (*OffersResult, error) {
	return &OffersResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin,
		Offers: []Offer{
			{Price: 12.99, Currency: "USD", Condition: "new", Merchant: "Amazon.com", Prime: true, Availability: "in_stock"},
			{Price: 9.49, Currency: "USD", Condition: "used_good", Merchant: "ThirdPartySeller", Prime: false, Availability: "in_stock"},
		},
		BuyboxPrice: 12.99,
	}, nil
}

func (s *stub) GetReviews(_ context.Context, asin, cursor string) (*ReviewsResult, error) {
	return &ReviewsResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin,
		Reviews: []Review{
			{Rating: 5, Title: "[stub] Great", Body: "[stub] Works as described.", Author: "alice", Date: "2026-05-01", Verified: true},
			{Rating: 2, Title: "[stub] Meh", Body: "[stub] Broke after a week.", Author: "bob", Date: "2026-04-12", Verified: false},
		},
	}, nil
}

func (s *stub) GetVariations(_ context.Context, asin string) (*VariationsResult, error) {
	return &VariationsResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ParentASIN: asin,
		Variations: []Variation{
			{ASIN: "B0STUBVAR1", Attributes: map[string]string{"color": "black", "size": "M"}, Price: 12.99, URL: "https://www.amazon.com/dp/B0STUBVAR1"},
			{ASIN: "B0STUBVAR2", Attributes: map[string]string{"color": "blue", "size": "L"}, Price: 13.49, URL: "https://www.amazon.com/dp/B0STUBVAR2"},
		},
	}, nil
}

func (s *stub) GetBrowseNode(_ context.Context, nodeID string) (*BrowseNode, error) {
	return &BrowseNode{
		SchemaVersion: SchemaVersion, Provider: s.Name(), NodeID: nodeID, Name: "[stub] Electronics",
		Ancestors: []string{"All"},
		Children:  []NodeRef{{NodeID: "172282", Name: "[stub] Accessories"}},
	}, nil
}
