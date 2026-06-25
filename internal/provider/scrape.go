package provider

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/httpx"
)

// scrape is the DIY amazon.com scraping backend — keyless, ADVANCED, OFF BY DEFAULT.
//
// Intended use: an agent or user on a RESIDENTIAL connection at home, making modest,
// throttled requests (which look like an ordinary shopper). Hosted/cloud/datacenter use
// should NOT use this backend — it will be blocked, and it carries Amazon ToS risk — those
// deployments should use the official Creators backend or an API provider.
//
// Scraping is inherently fragile: Amazon A/B-tests its DOM and runs aggressive bot
// detection. Selectors here are best-effort and WILL break; failures surface as SCHEMA_DRIFT
// or BLOCKED (with a persistent cooldown so a fresh process fails fast instead of hammering).
// Enable with RIVR_SCRAPE_ENABLE=1 (you accept the ToS/breakage risk).
type scrape struct {
	domain string
	http   *httpx.Client
}

const scrapeUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func newScrape() *scrape {
	c := httpx.New()
	c.UserAgent = scrapeUA // present as a real browser, not "rivr"
	return &scrape{domain: "amazon.com", http: c}
}

func (s *scrape) Name() string { return "scrape" }

func (s *scrape) Capabilities() []string {
	// Reviews are NOT supported: Amazon serves /product-reviews/ behind a login/bot wall, so
	// keyless scraping reliably returns nothing — better to declare it unsupported than to
	// return a silent empty list. Variations (twister) and browse-nodes aren't parsed either.
	return []string{CapSearch, CapItem, CapOffers}
}

func (s *scrape) Configured() bool { return scrapeEnabled() }

func scrapeEnabled() bool { return os.Getenv("RIVR_SCRAPE_ENABLE") == "1" }

// UnconfiguredErr explains the opt-in instead of a misleading "pipe an API key" message.
func (s *scrape) UnconfiguredErr() error { return s.notEnabled() }

func (s *scrape) notEnabled() *errs.CLIError {
	return errs.New(errs.ExitConfig, "SCRAPE_DISABLED",
		"the scrape backend is disabled (violates Amazon ToS; aggressive bot detection)",
		"prefer an API backend (--provider serpapi), or for home/residential use set "+
			"RIVR_SCRAPE_ENABLE=1 — you accept the ToS/breakage risk. Do NOT use scrape from cloud/hosted IPs.")
}

func (s *scrape) base() string {
	if v := os.Getenv("RIVR_SCRAPE_BASE"); v != "" { // override for tests/mirrors
		return strings.TrimRight(v, "/")
	}
	return "https://www." + s.domain
}

var captchaRe = regexp.MustCompile(`(?i)(robot check|validateCaptcha|type the characters|enter the characters you see)`)

// fetch GETs a page as a browser and returns a parsed document, detecting block pages.
func (s *scrape) fetch(ctx context.Context, rawurl string) (*goquery.Document, error) {
	if !scrapeEnabled() {
		return nil, s.notEnabled()
	}
	if err := preflight(s.Name()); err != nil { // honor an active cooldown
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodGet, rawurl, nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := s.http.Do(ctx, req)
	if err != nil {
		return nil, errs.Retryable(s.Name(), err.Error())
	}
	if resp.Status == http.StatusTooManyRequests || resp.Status == http.StatusServiceUnavailable || captchaRe.Match(resp.Body) {
		markBlocked(s.Name(), 900) // 15m cooldown; the next process fails fast
		return nil, errs.Blocked(s.Name(), 900)
	}
	if resp.Status != http.StatusOK {
		return nil, errs.Upstream(s.Name(), "status "+strconv.Itoa(resp.Status))
	}
	doc, derr := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
	if derr != nil {
		return nil, errs.Upstream(s.Name(), "could not parse HTML")
	}
	markOK(s.Name())
	return doc, nil
}

var priceRe = regexp.MustCompile(`[\d.]+`)

// parsePrice extracts a float from an Amazon price string like "$1,234.56".
func parsePrice(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	m := priceRe.FindString(s)
	f, _ := strconv.ParseFloat(m, 64)
	return f
}

// parseLeadingFloat pulls the leading number from e.g. "4.6 out of 5 stars".
func parseLeadingFloat(s string) float64 {
	m := regexp.MustCompile(`[\d.]+`).FindString(strings.TrimSpace(s))
	f, _ := strconv.ParseFloat(m, 64)
	return f
}

// parseCount pulls an int from e.g. "1,234 ratings".
func parseCount(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	m := regexp.MustCompile(`\d+`).FindString(s)
	n, _ := strconv.Atoi(m)
	return n
}

func (s *scrape) absURL(href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return s.base() + href
}

func (s *scrape) Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	page := 1
	if opts.Cursor != "" {
		if p, e := strconv.Atoi(opts.Cursor); e == nil && p > 0 {
			page = p
		}
	}
	q := url.Values{"k": {query}, "page": {strconv.Itoa(page)}}
	if opts.Category != "" {
		q.Set("i", opts.Category)
	}
	doc, err := s.fetch(ctx, s.base()+"/s?"+q.Encode())
	if err != nil {
		return nil, err
	}
	res := &SearchResult{SchemaVersion: SchemaVersion, Provider: s.Name(), Query: query, Limit: opts.Limit}
	doc.Find("div.s-result-item[data-asin]").Each(func(_ int, sel *goquery.Selection) {
		asin, _ := sel.Attr("data-asin")
		if asin == "" {
			return
		}
		title := firstNonEmpty(sel.Find("h2 a span").First().Text(), sel.Find("h2 span").First().Text(), sel.Find("h2").First().Text())
		if title == "" {
			return // not a real product row
		}
		href, _ := sel.Find("h2 a").Attr("href")
		img, _ := sel.Find("img.s-image").Attr("src")
		// Amazon's DOM drifts; the URL must never be empty, so fall back to the canonical
		// /dp/<ASIN> form (the deep link is rivr's whole point).
		u := s.absURL(href)
		if u == "" {
			u = s.base() + "/dp/" + asin
		}
		item := SearchItem{
			ASIN:        asin,
			Title:       strings.TrimSpace(title),
			Price:       parsePrice(sel.Find("span.a-price span.a-offscreen").First().Text()),
			Currency:    currencyForDomain(s.domain),
			Rating:      parseLeadingFloat(sel.Find("span.a-icon-alt").First().Text()),
			ReviewCount: scrapeReviewCount(sel),
			Prime:       sel.Find("i.a-icon-prime").Length() > 0,
			URL:         u,
			Image:       img,
		}
		res.Items = append(res.Items, item)
	})
	res.Count = len(res.Items)
	if len(res.Items) > 0 {
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (s *scrape) GetItem(ctx context.Context, asin string, detailed bool) (*Item, error) {
	doc, err := s.fetch(ctx, s.base()+"/dp/"+asin)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(doc.Find("#productTitle").First().Text())
	if title == "" {
		return nil, errs.SchemaDrift(s.Name(), "#productTitle")
	}
	it := &Item{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin,
		Title:       title,
		Brand:       strings.TrimSpace(doc.Find("#bylineInfo").First().Text()),
		Price:       parsePrice(doc.Find("#corePrice_feature_div span.a-offscreen, span.a-price span.a-offscreen").First().Text()),
		Currency:    currencyForDomain(s.domain),
		Rating:      parseLeadingFloat(doc.Find("#acrPopover").AttrOr("title", "")),
		ReviewCount: parseCount(doc.Find("#acrCustomerReviewText").First().Text()),
		URL:         s.base() + "/dp/" + asin,
	}
	if src, ok := doc.Find("#landingImage").Attr("src"); ok {
		it.Images = []string{src}
	}
	if detailed {
		it.Description = strings.TrimSpace(doc.Find("#productDescription").First().Text())
		doc.Find("#feature-bullets li span.a-list-item").Each(func(_ int, sel *goquery.Selection) {
			if t := strings.TrimSpace(sel.Text()); t != "" {
				it.Features = append(it.Features, t)
			}
		})
	}
	return it, nil
}

func (s *scrape) GetOffers(ctx context.Context, asin string) (*OffersResult, error) {
	doc, err := s.fetch(ctx, s.base()+"/dp/"+asin)
	if err != nil {
		return nil, err
	}
	price := parsePrice(doc.Find("#corePrice_feature_div span.a-offscreen, span.a-price span.a-offscreen").First().Text())
	if price == 0 {
		return nil, errs.NotFound("offers for item", asin)
	}
	merchant := strings.TrimSpace(doc.Find("#sellerProfileTriggerId, #merchant-info").First().Text())
	res := &OffersResult{
		SchemaVersion: SchemaVersion, Provider: s.Name(), ASIN: asin, BuyboxPrice: price,
		Offers: []Offer{{
			Price: price, Currency: currencyForDomain(s.domain), Condition: "new",
			Merchant: merchant, Availability: strings.TrimSpace(doc.Find("#availability span").First().Text()),
		}},
	}
	return res, nil
}

// GetReviews is unsupported on scrape: Amazon's /product-reviews/ page is login/bot-walled,
// so keyless scraping returns nothing. Return a clear, structured error instead of an empty
// list that an agent would mistake for "this product has no reviews".
func (s *scrape) GetReviews(_ context.Context, _, _ string) (*ReviewsResult, error) {
	if !scrapeEnabled() {
		return nil, s.notEnabled()
	}
	return nil, errs.New(errs.ExitUnsupported, "UNSUPPORTED_BY_PROVIDER",
		"review text is not available via the keyless scrape backend (Amazon walls /product-reviews/)",
		"use --provider rainforest for full reviews, or --provider serpapi for a sample")
}

func (s *scrape) GetVariations(_ context.Context, _ string) (*VariationsResult, error) {
	return nil, errs.Unsupported("variations", s.Name())
}

func (s *scrape) GetBrowseNode(_ context.Context, _ string) (*BrowseNode, error) {
	return nil, errs.Unsupported("browse", s.Name())
}

// Validate (for doctor/auth status): scrape is "valid" only when explicitly enabled.
func (s *scrape) Validate(_ context.Context) error {
	if !scrapeEnabled() {
		return s.notEnabled()
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// scrapeReviewCount extracts a search row's review count, which Amazon renders inconsistently
// across DOM variants. Try the known shapes in order; the rating-row aria-label is the most
// stable (e.g. aria-label="2,578"). Returns 0 if none match (degrade, don't break).
func scrapeReviewCount(sel *goquery.Selection) int {
	// 1) the count link next to the stars, e.g. <a aria-label="2,578 ratings">
	if v, ok := sel.Find("a[aria-label$='ratings'], a[aria-label$='reviews']").First().Attr("aria-label"); ok {
		if n := parseCount(v); n > 0 {
			return n
		}
	}
	// 2) the second aria-label in the rating row (first is the stars, second is the count)
	if v, ok := sel.Find("span[aria-label]").Eq(1).Attr("aria-label"); ok {
		if n := parseCount(v); n > 0 {
			return n
		}
	}
	// 3) legacy underline-text span
	if n := parseCount(sel.Find("span.a-size-base.s-underline-text").First().Text()); n > 0 {
		return n
	}
	return 0
}
