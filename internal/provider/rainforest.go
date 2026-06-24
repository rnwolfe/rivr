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

// rainforest is the Rainforest API (Traject Data) backend: a single endpoint switched by
// `type` (search/product/offers/reviews). Richest of the third-party backends — full
// paginated reviews (with author) and real offer/seller data. Browse-node trees are not
// modeled here (use the Creators backend). Auth: api_key query param.
type rainforest struct {
	base   string
	domain string
	http   *httpx.Client
}

func newRainforest() *rainforest {
	return &rainforest{base: "https://api.rainforestapi.com", domain: "amazon.com", http: httpx.New()}
}

func (r *rainforest) Name() string { return "rainforest" }

func (r *rainforest) Capabilities() []string {
	return []string{CapSearch, CapItem, CapOffers, CapReviews, CapVariations}
}

func (r *rainforest) Configured() bool {
	k, _ := auth.Get(r.Name(), auth.FieldAPIKey)
	return k != ""
}

func (r *rainforest) call(ctx context.Context, q url.Values) (map[string]any, error) {
	if err := preflight(r.Name()); err != nil {
		return nil, err
	}
	key, _ := auth.Get(r.Name(), auth.FieldAPIKey)
	if key == "" {
		return nil, errs.AuthRequired(r.Name())
	}
	q.Set("api_key", key)
	req, _ := http.NewRequest(http.MethodGet, r.base+"/request?"+q.Encode(), nil)
	resp, err := r.http.Do(ctx, req)
	if err != nil {
		return nil, errs.Retryable(r.Name(), err.Error())
	}
	switch resp.Status {
	case http.StatusUnauthorized:
		return nil, errs.AuthRequired(r.Name())
	case http.StatusPaymentRequired:
		markBlocked(r.Name(), 3600)
		return nil, errs.RateLimited(r.Name()).WithRetryAfter(3600)
	case http.StatusTooManyRequests:
		ra := resp.RetryAfterSeconds()
		markBlocked(r.Name(), max(ra, 60))
		return nil, errs.RateLimited(r.Name()).WithRetryAfter(ra)
	case http.StatusServiceUnavailable:
		return nil, errs.Retryable(r.Name(), "service unavailable")
	}
	if resp.Status >= 500 {
		return nil, errs.Retryable(r.Name(), "server error "+strconv.Itoa(resp.Status))
	}
	m, derr := httpx.Decode(resp.Body)
	if derr != nil {
		return nil, errs.Upstream(r.Name(), "invalid JSON")
	}
	if !httpx.Bool(m, "request_info.success") {
		msg := httpx.Str(m, "request_info.message")
		if msg == "" {
			return nil, errs.Upstream(r.Name(), "request_info.success=false")
		}
		lower := strings.ToLower(msg)
		switch {
		case strings.Contains(lower, "api key"):
			return nil, errs.AuthRequired(r.Name())
		case strings.Contains(lower, "credit"):
			markBlocked(r.Name(), 3600)
			return nil, errs.RateLimited(r.Name()).WithRetryAfter(3600)
		default:
			return nil, errs.Upstream(r.Name(), msg)
		}
	}
	markOK(r.Name())
	return m, nil
}

// upper normalizes Rainforest's inconsistent currency casing (usd vs USD).
func upper(s string) string { return strings.ToUpper(s) }

func (r *rainforest) Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	q := url.Values{}
	q.Set("type", "search")
	q.Set("amazon_domain", r.domain)
	q.Set("search_term", query)
	if opts.Category != "" {
		q.Set("category_id", opts.Category)
	}
	if opts.Sort != "" {
		q.Set("sort_by", opts.Sort)
	}
	page := 1
	if opts.Cursor != "" {
		if p, err := strconv.Atoi(opts.Cursor); err == nil && p > 0 {
			page = p
		}
	}
	q.Set("page", strconv.Itoa(page))
	m, err := r.call(ctx, q)
	if err != nil {
		return nil, err
	}
	rows := httpx.Arr(m, "search_results")
	items := make([]SearchItem, 0, len(rows))
	for _, row := range rows {
		o := httpx.AsObj(row)
		items = append(items, SearchItem{
			ASIN:        httpx.Str(o, "asin"),
			Title:       httpx.Str(o, "title"),
			Price:       httpx.Float(o, "price.value"),
			Currency:    upper(httpx.Str(o, "price.currency")),
			Rating:      httpx.Float(o, "rating"),
			ReviewCount: httpx.Int(o, "ratings_total"),
			Prime:       httpx.Bool(o, "is_prime"),
			URL:         httpx.Str(o, "link"),
			Image:       httpx.Str(o, "image"),
		})
	}
	res := &SearchResult{
		SchemaVersion: SchemaVersion, Provider: r.Name(), Query: query,
		Items: items, Count: len(items), Limit: opts.Limit,
	}
	if cur, tot := httpx.Int(m, "pagination.current_page"), httpx.Int(m, "pagination.total_pages"); tot > cur {
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (r *rainforest) GetItem(ctx context.Context, asin string, detailed bool) (*Item, error) {
	q := url.Values{}
	q.Set("type", "product")
	q.Set("amazon_domain", r.domain)
	q.Set("asin", asin)
	m, err := r.call(ctx, q)
	if err != nil {
		return nil, err
	}
	p := httpx.AsObj(mget(m, "product"))
	if p == nil {
		return nil, errs.NotFound("item", asin)
	}
	it := &Item{
		SchemaVersion: SchemaVersion, Provider: r.Name(), ASIN: asin,
		Title:       httpx.Str(p, "title"),
		Brand:       httpx.Str(p, "brand"),
		Price:       httpx.Float(p, "buybox_winner.price.value"),
		Currency:    upper(httpx.Str(p, "buybox_winner.price.currency")),
		Rating:      httpx.Float(p, "rating"),
		ReviewCount: httpx.Int(p, "ratings_total"),
		SalesRank:   firstRank(httpx.Arr(p, "bestsellers_rank")),
		URL:         httpx.Str(p, "link"),
	}
	if it.URL == "" {
		it.URL = "https://www." + r.domain + "/dp/" + asin
	}
	for _, img := range httpx.Arr(p, "images") {
		if u := httpx.Str(httpx.AsObj(img), "link"); u != "" {
			it.Images = append(it.Images, u)
		}
	}
	if detailed {
		it.Description = httpx.Str(p, "description")
		for _, f := range httpx.Arr(p, "feature_bullets") {
			if s, ok := f.(string); ok {
				it.Features = append(it.Features, s)
			}
		}
	}
	return it, nil
}

func (r *rainforest) GetOffers(ctx context.Context, asin string) (*OffersResult, error) {
	q := url.Values{}
	q.Set("type", "offers")
	q.Set("amazon_domain", r.domain)
	q.Set("asin", asin)
	m, err := r.call(ctx, q)
	if err != nil {
		return nil, err
	}
	res := &OffersResult{SchemaVersion: SchemaVersion, Provider: r.Name(), ASIN: asin}
	for _, off := range httpx.Arr(m, "offers") {
		o := httpx.AsObj(off)
		offer := Offer{
			Price:        httpx.Float(o, "price.value"),
			Currency:     upper(httpx.Str(o, "price.currency")),
			Condition:    strings.ToLower(httpx.Str(o, "condition.title")),
			Merchant:     httpx.Str(o, "seller.name"),
			Prime:        httpx.Bool(o, "is_prime"),
			Availability: deliveryAvailability(o),
		}
		if httpx.Bool(o, "buybox_winner") {
			res.BuyboxPrice = offer.Price
		}
		res.Offers = append(res.Offers, offer)
	}
	if len(res.Offers) == 0 {
		return nil, errs.NotFound("offers for item", asin)
	}
	return res, nil
}

func (r *rainforest) GetReviews(ctx context.Context, asin, cursor string) (*ReviewsResult, error) {
	q := url.Values{}
	q.Set("type", "reviews")
	q.Set("amazon_domain", r.domain)
	q.Set("asin", asin)
	page := 1
	if cursor != "" {
		if p, err := strconv.Atoi(cursor); err == nil && p > 0 {
			page = p
		}
	}
	q.Set("page", strconv.Itoa(page))
	m, err := r.call(ctx, q)
	if err != nil {
		return nil, err
	}
	res := &ReviewsResult{SchemaVersion: SchemaVersion, Provider: r.Name(), ASIN: asin, Scope: "full"}
	for _, rv := range httpx.Arr(m, "reviews") {
		o := httpx.AsObj(rv)
		res.Reviews = append(res.Reviews, Review{
			Rating:   httpx.Int(o, "rating"),
			Title:    httpx.Str(o, "title"),
			Body:     httpx.Str(o, "body"),
			Author:   httpx.Str(o, "profile.name"),
			Date:     httpx.Str(o, "date.utc"),
			Verified: httpx.Bool(o, "verified_purchase"),
		})
	}
	if cur, tot := httpx.Int(m, "pagination.current_page"), httpx.Int(m, "pagination.total_pages"); tot > cur {
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (r *rainforest) GetVariations(ctx context.Context, asin string) (*VariationsResult, error) {
	q := url.Values{}
	q.Set("type", "product")
	q.Set("amazon_domain", r.domain)
	q.Set("asin", asin)
	m, err := r.call(ctx, q)
	if err != nil {
		return nil, err
	}
	p := httpx.AsObj(mget(m, "product"))
	res := &VariationsResult{SchemaVersion: SchemaVersion, Provider: r.Name(), ParentASIN: asin}
	for _, v := range httpx.Arr(p, "variants") {
		o := httpx.AsObj(v)
		attrs := map[string]string{}
		for _, d := range httpx.Arr(o, "dimensions") {
			do := httpx.AsObj(d)
			attrs[httpx.Str(do, "name")] = httpx.Str(do, "value")
		}
		res.Variations = append(res.Variations, Variation{
			ASIN:       httpx.Str(o, "asin"),
			Attributes: attrs,
			URL:        "https://www." + r.domain + "/dp/" + httpx.Str(o, "asin"),
		})
	}
	return res, nil
}

func (r *rainforest) GetBrowseNode(_ context.Context, _ string) (*BrowseNode, error) {
	return nil, errs.Unsupported("browse", r.Name())
}

func firstRank(ranks []any) int {
	if len(ranks) == 0 {
		return 0
	}
	return httpx.Int(httpx.AsObj(ranks[0]), "rank")
}

func deliveryAvailability(o map[string]any) string {
	if httpx.Bool(o, "delivery.fulfilled_by_amazon") {
		return "in_stock"
	}
	return ""
}
