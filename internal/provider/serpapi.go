package provider

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rnwolfe/rivr/internal/auth"
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/httpx"
)

// serpApi is the default backend: SerpApi's Amazon engines. SerpApi exposes only two
// engines — `amazon` (search) and `amazon_product` (detail, with reviews/offers/variants
// as sub-blocks). Browse-node listing is not supported. Reviews are an on-page SAMPLE, not
// the full paginated corpus (declared via scope=sample). Auth: api_key query param.
type serpApi struct {
	base   string // override for tests; defaults to https://serpapi.com
	domain string // amazon_domain
	http   *httpx.Client
}

func newSerpApi() *serpApi {
	return &serpApi{base: "https://serpapi.com", domain: "amazon.com", http: httpx.New()}
}

func (s *serpApi) Name() string { return "serpapi" }

func (s *serpApi) Capabilities() []string {
	// No browse-node engine on SerpApi.
	return []string{CapSearch, CapItem, CapOffers, CapReviews, CapVariations}
}

func (s *serpApi) Configured() bool {
	k, _ := auth.Get(s.Name(), auth.FieldAPIKey)
	return k != ""
}

func (s *serpApi) key() (string, error) {
	k, _ := auth.Get(s.Name(), auth.FieldAPIKey)
	if k == "" {
		return "", errs.AuthRequired(s.Name())
	}
	return k, nil
}

// call hits a SerpApi endpoint and returns the decoded JSON, classifying SerpApi's
// error envelope (which can carry `error` even on HTTP 200) into domain errors.
func (s *serpApi) call(ctx context.Context, path string, q url.Values) (map[string]any, error) {
	if err := preflight(s.Name()); err != nil {
		return nil, err
	}
	key, err := s.key()
	if err != nil {
		return nil, err
	}
	q.Set("api_key", key)
	req, _ := http.NewRequest(http.MethodGet, s.base+path+"?"+q.Encode(), nil)
	resp, err := s.http.Do(ctx, req)
	if err != nil {
		return nil, errs.Retryable(s.Name(), err.Error())
	}
	switch {
	case resp.Status == http.StatusUnauthorized || resp.Status == http.StatusForbidden:
		return nil, errs.AuthRequired(s.Name())
	case resp.Status == http.StatusTooManyRequests:
		ra := resp.RetryAfterSeconds()
		markBlocked(s.Name(), max(ra, 60))
		return nil, errs.RateLimited(s.Name()).WithRetryAfter(ra)
	case resp.Status >= 500:
		return nil, errs.Retryable(s.Name(), "server error "+strconv.Itoa(resp.Status))
	}
	m, derr := httpx.Decode(resp.Body)
	if derr != nil {
		return nil, errs.Upstream(s.Name(), "invalid JSON")
	}
	// SerpApi reports soft failures via a top-level `error` string, even on HTTP 200.
	if msg := httpx.Str(m, "error"); msg != "" {
		switch {
		case strings.Contains(msg, "Invalid API key"):
			return nil, errs.AuthRequired(s.Name())
		case strings.Contains(msg, "run out of searches"), strings.Contains(msg, "hourly limit"):
			markBlocked(s.Name(), 3600)
			return nil, errs.RateLimited(s.Name()).WithRetryAfter(3600)
		case strings.Contains(msg, "any results"):
			return m, nil // empty results — let the command map to EMPTY_RESULTS
		default:
			return nil, errs.Upstream(s.Name(), msg)
		}
	}
	markOK(s.Name())
	return m, nil
}

func (s *serpApi) Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	q := url.Values{}
	q.Set("engine", "amazon")
	q.Set("amazon_domain", s.domain)
	if opts.Category != "" {
		q.Set("node", opts.Category)
	} else {
		q.Set("k", query)
	}
	page := 1
	if opts.Cursor != "" {
		if p, err := strconv.Atoi(opts.Cursor); err == nil && p > 0 {
			page = p
		}
	}
	q.Set("page", strconv.Itoa(page))
	if opts.Sort != "" {
		q.Set("s", opts.Sort)
	}
	m, err := s.call(ctx, "/search", q)
	if err != nil {
		return nil, err
	}
	cur := currencyForDomain(s.domain)
	rows := httpx.Arr(m, "organic_results")
	items := make([]SearchItem, 0, len(rows))
	for _, r := range rows {
		o := httpx.AsObj(r)
		items = append(items, SearchItem{
			ASIN:        httpx.Str(o, "asin"),
			Title:       httpx.Str(o, "title"),
			Price:       httpx.Float(o, "extracted_price"),
			Currency:    cur,
			Rating:      httpx.Float(o, "rating"),
			ReviewCount: httpx.Int(o, "reviews"),
			Prime:       httpx.Bool(o, "prime"),
			URL:         httpx.Str(o, "link"),
			Image:       httpx.Str(o, "thumbnail"),
		})
	}
	res := &SearchResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), Query: query,
		Items: items, Count: len(items), Limit: opts.Limit,
	}
	// SerpApi advertises a next page via serpapi_pagination.next.
	if httpx.Has(m, "serpapi_pagination.next") && len(items) > 0 {
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (s *serpApi) product(ctx context.Context, asin string) (map[string]any, error) {
	q := url.Values{}
	q.Set("engine", "amazon_product")
	q.Set("amazon_domain", s.domain)
	q.Set("asin", asin)
	q.Set("other_sellers", "true")
	return s.call(ctx, "/search", q)
}

func (s *serpApi) GetItem(ctx context.Context, asin string, detailed bool) (*Item, error) {
	m, err := s.product(ctx, asin)
	if err != nil {
		return nil, err
	}
	pr := httpx.AsObj(mget(m, "product_results"))
	if pr == nil {
		return nil, errs.NotFound("item", asin)
	}
	cur := currencyForDomain(s.domain)
	it := &Item{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin,
		Title:       httpx.Str(pr, "title"),
		Brand:       httpx.Str(pr, "brand"),
		Price:       httpx.Float(pr, "extracted_price"),
		Currency:    cur,
		Rating:      httpx.Float(pr, "rating"),
		ReviewCount: httpx.Int(pr, "reviews"),
		URL:         "https://www." + s.domain + "/dp/" + asin,
	}
	for _, img := range httpx.Arr(pr, "thumbnails") {
		if u, ok := img.(string); ok {
			it.Images = append(it.Images, u)
		}
	}
	if len(it.Images) == 0 {
		if t := httpx.Str(pr, "thumbnail"); t != "" {
			it.Images = []string{t}
		}
	}
	if detailed {
		it.Description = httpx.Str(pr, "description")
		// Feature-bullets key varies; try the documented candidates in order.
		for _, k := range []string{"about_item", "product_features", "feature_bullets"} {
			for _, f := range httpx.Arr(pr, k) {
				if s, ok := f.(string); ok {
					it.Features = append(it.Features, s)
				}
			}
			if len(it.Features) > 0 {
				break
			}
		}
	}
	return it, nil
}

func (s *serpApi) GetOffers(ctx context.Context, asin string) (*OffersResult, error) {
	m, err := s.product(ctx, asin)
	if err != nil {
		return nil, err
	}
	pr := httpx.AsObj(mget(m, "product_results"))
	cur := currencyForDomain(s.domain)
	res := &OffersResult{SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin}
	// Buybox / featured offer.
	if buy := httpx.Float(pr, "purchase_options.buy_new.extracted_price"); buy > 0 {
		res.BuyboxPrice = buy
		res.Offers = append(res.Offers, Offer{Price: buy, Currency: cur, Condition: "new", Merchant: "Amazon.com", Availability: "in_stock"})
	} else if buy := httpx.Float(pr, "extracted_price"); buy > 0 {
		res.BuyboxPrice = buy
		res.Offers = append(res.Offers, Offer{Price: buy, Currency: cur, Condition: "new", Availability: "in_stock"})
	}
	// Other sellers (limited fields; seller/prime not reliably present).
	for _, p := range httpx.Arr(m, "prices") {
		o := httpx.AsObj(p)
		res.Offers = append(res.Offers, Offer{
			Price:     httpx.Float(o, "extracted_price"),
			Currency:  cur,
			Condition: strings.ToLower(httpx.Str(o, "condition")),
		})
	}
	if len(res.Offers) == 0 {
		return nil, errs.NotFound("offers for item", asin)
	}
	return res, nil
}

func (s *serpApi) GetReviews(ctx context.Context, asin, cursor string) (*ReviewsResult, error) {
	m, err := s.product(ctx, asin)
	if err != nil {
		return nil, err
	}
	pr := httpx.AsObj(mget(m, "product_results"))
	_ = pr
	res := &ReviewsResult{SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin, Scope: "sample"}
	for _, r := range httpx.Arr(m, "reviews_information.authors_reviews") {
		o := httpx.AsObj(r)
		res.Reviews = append(res.Reviews, Review{
			Rating:   httpx.Int(o, "rating"),
			Title:    httpx.Str(o, "title"),
			Body:     httpx.Str(o, "text"),
			Author:   httpx.Str(o, "author"),
			Date:     httpx.Str(o, "date"),
			Verified: httpx.Bool(o, "verified_purchase"),
		})
	}
	return res, nil
}

func (s *serpApi) GetVariations(ctx context.Context, asin string) (*VariationsResult, error) {
	m, err := s.product(ctx, asin)
	if err != nil {
		return nil, err
	}
	pr := httpx.AsObj(mget(m, "product_results"))
	res := &VariationsResult{SchemaVersion: SchemaVersion, Provider: s.Name(), ParentASIN: asin}
	for _, grp := range httpx.Arr(pr, "variants") {
		g := httpx.AsObj(grp)
		dim := httpx.Str(g, "title")
		for _, it := range httpx.Arr(g, "items") {
			io := httpx.AsObj(it)
			va := Variation{
				ASIN:       httpx.Str(io, "asin"),
				Attributes: map[string]string{dim: httpx.Str(io, "name")},
				URL:        "https://www." + s.domain + "/dp/" + httpx.Str(io, "asin"),
			}
			res.Variations = append(res.Variations, va)
		}
	}
	return res, nil
}

func (s *serpApi) GetBrowseNode(_ context.Context, _ string) (*BrowseNode, error) {
	return nil, errs.Unsupported("browse", s.Name())
}

// mget is a tiny helper to read a nested object value as any.
func mget(m map[string]any, key string) any { return m[key] }
