// Package provider defines the normalized Amazon-Shopping data schema and the pluggable
// backend interface. The output types here ARE the append-only output contract
// (spec.md "Output schema") — every backend normalizes into them.
//
// This file ships only the interface + types + a stub backend so the skeleton compiles,
// runs, and is testable offline. cli-implement registers the real backends
// (SerpApi default, Rainforest, official Creators API) and wires keyring auth.
package provider

import "context"

// SchemaVersion is the output-contract version. Bump only for breaking field changes.
const SchemaVersion = "1"

// Capability names the operations a backend can serve. Not all providers serve all
// capabilities (e.g. the official Creators API returns no review text).
const (
	CapSearch     = "search"
	CapItem       = "item"
	CapOffers     = "offers"
	CapReviews    = "reviews"
	CapVariations = "variations"
	CapBrowse     = "browse"
)

// SearchOpts carries the search filters (all optional).
type SearchOpts struct {
	Category string
	MinRating float64
	Prime     bool
	MinPrice  float64
	MaxPrice  float64
	Sort      string
	Limit     int
	Cursor    string
}

// --- normalized output types (the stable JSON schema) -----------------------

type SearchResult struct {
	SchemaVersion string       `json:"schemaVersion"`
	Provider      string       `json:"provider"`
	Query         string       `json:"query"`
	Items         []SearchItem `json:"items"`
	NextCursor    string       `json:"nextCursor,omitempty"`
	Count         int          `json:"count"`
	Limit         int          `json:"limit"`
}

type SearchItem struct {
	ASIN        string  `json:"asin"`
	Title       string  `json:"title"` // free text — fenced in agent mode
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"reviewCount"`
	Prime       bool    `json:"prime"`
	URL         string  `json:"url"`
	Image       string  `json:"image"`
}

type Item struct {
	SchemaVersion string   `json:"schemaVersion"`
	Provider      string   `json:"provider"`
	ASIN          string   `json:"asin"`
	Title         string   `json:"title"`       // free text — fenced
	Brand         string   `json:"brand"`
	Price         float64  `json:"price"`
	Currency      string   `json:"currency"`
	Offers        []Offer  `json:"offers"`
	Features      []string `json:"features"`    // free text — fenced
	Description   string   `json:"description"` // free text — fenced
	Images        []string `json:"images"`
	Rating        float64  `json:"rating"`
	ReviewCount   int      `json:"reviewCount"`
	SalesRank     int      `json:"salesRank"`
	URL           string   `json:"url"`
}

type Offer struct {
	Price        float64 `json:"price"`
	Currency     string  `json:"currency"`
	Condition    string  `json:"condition"`
	Merchant     string  `json:"merchant"`
	Prime        bool    `json:"prime"`
	Availability string  `json:"availability"`
}

type OffersResult struct {
	SchemaVersion string  `json:"schemaVersion"`
	Provider      string  `json:"provider"`
	ASIN          string  `json:"asin"`
	Offers        []Offer `json:"offers"`
	BuyboxPrice   float64 `json:"buyboxPrice"`
}

type ReviewsResult struct {
	SchemaVersion string   `json:"schemaVersion"`
	Provider      string   `json:"provider"`
	ASIN          string   `json:"asin"`
	Reviews       []Review `json:"reviews"`
	NextCursor    string   `json:"nextCursor,omitempty"`
}

type Review struct {
	Rating   int    `json:"rating"`
	Title    string `json:"title"` // free text — fenced
	Body     string `json:"body"`  // free text — fenced
	Author   string `json:"author"`
	Date     string `json:"date"`
	Verified bool   `json:"verified"`
}

type VariationsResult struct {
	SchemaVersion string      `json:"schemaVersion"`
	Provider      string      `json:"provider"`
	ParentASIN    string      `json:"parentAsin"`
	Variations    []Variation `json:"variations"`
}

type Variation struct {
	ASIN       string            `json:"asin"`
	Attributes map[string]string `json:"attributes"`
	Price      float64           `json:"price"`
	URL        string            `json:"url"`
}

type BrowseNode struct {
	SchemaVersion string    `json:"schemaVersion"`
	Provider      string    `json:"provider"`
	NodeID        string    `json:"nodeId"`
	Name          string    `json:"name"`
	Ancestors     []string  `json:"ancestors"`
	Children      []NodeRef `json:"children"`
}

type NodeRef struct {
	NodeID string `json:"nodeId"`
	Name   string `json:"name"`
}

// Provider is the pluggable backend interface. cli-implement adds real implementations.
type Provider interface {
	Name() string
	Capabilities() []string
	Configured() bool
	Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error)
	GetItem(ctx context.Context, asin string, detailed bool) (*Item, error)
	GetOffers(ctx context.Context, asin string) (*OffersResult, error)
	GetReviews(ctx context.Context, asin, cursor string) (*ReviewsResult, error)
	GetVariations(ctx context.Context, asin string) (*VariationsResult, error)
	GetBrowseNode(ctx context.Context, nodeID string) (*BrowseNode, error)
}

// Supports reports whether a provider advertises a capability.
func Supports(p Provider, cap string) bool {
	for _, c := range p.Capabilities() {
		if c == cap {
			return true
		}
	}
	return false
}
