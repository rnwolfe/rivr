package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/rivr/internal/auth"
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/httpx"
)

// creators is the official Amazon Creators API backend (the 2026 successor to PA-API 5.0).
// Auth: OAuth2 client-credentials (client_id + client_secret → cached bearer). partnerTag
// (the Associates tag) is REQUIRED on every request and is supplied by the CLI's resolved
// associate tag (SetPartnerTag). No review text → reviews is Unsupported.
//
// IMPORTANT: the Creators portal is access-gated; the host/path/token-endpoint defaults
// below are community-derived and OVERRIDABLE via env so a user can correct them without a
// rebuild:
//
//	RIVR_CREATORS_TOKEN_URL, RIVR_CREATORS_API_HOST, RIVR_CREATORS_MARKETPLACE
type creators struct {
	http       *httpx.Client
	partnerTag string
}

func newCreators() *creators { return &creators{http: httpx.New()} }

func (c *creators) Name() string { return "creators" }

func (c *creators) Capabilities() []string {
	// Official API returns NO review text → reviews unsupported.
	return []string{CapSearch, CapItem, CapOffers, CapVariations, CapBrowse}
}

func (c *creators) Configured() bool {
	id, _ := auth.Get(c.Name(), auth.FieldClientID)
	sec, _ := auth.Get(c.Name(), auth.FieldClientSecret)
	return id != "" && sec != ""
}

// SetPartnerTag implements TagAware: the CLI injects the resolved Associates tag.
func (c *creators) SetPartnerTag(tag string) { c.partnerTag = tag }

func (c *creators) tokenURL() string {
	if v := os.Getenv("RIVR_CREATORS_TOKEN_URL"); v != "" {
		return v
	}
	return "https://api.amazon.com/auth/o2/token" // v3.1 NA default
}

func (c *creators) apiHost() string {
	if v := os.Getenv("RIVR_CREATORS_API_HOST"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://creatorsapi.amazon"
}

func (c *creators) marketplace() string {
	if v := os.Getenv("RIVR_CREATORS_MARKETPLACE"); v != "" {
		return v
	}
	return "www.amazon.com"
}

// --- OAuth client-credentials with a cross-process token cache ---------------

type cachedToken struct {
	AccessToken string `json:"access_token"`
	ExpiryUnix  int64  `json:"expiry_unix"`
}

func tokenCachePath() string {
	d := os.Getenv("XDG_STATE_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(d, "rivr", "creators-token.json")
}

func (c *creators) token(ctx context.Context) (string, error) {
	// Reuse a cached token until ~60s before expiry (TTL ~1h; minting per agent call is wasteful).
	if b, err := os.ReadFile(tokenCachePath()); err == nil {
		var t cachedToken
		if json.Unmarshal(b, &t) == nil && t.AccessToken != "" && time.Now().Unix() < t.ExpiryUnix-60 {
			return t.AccessToken, nil
		}
	}
	id, _ := auth.Get(c.Name(), auth.FieldClientID)
	sec, _ := auth.Get(c.Name(), auth.FieldClientSecret)
	if id == "" || sec == "" {
		return "", errs.AuthRequired(c.Name())
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "creatorsapi::default")
	req, _ := http.NewRequest(http.MethodPost, c.tokenURL(), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(id, sec)
	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return "", errs.Retryable(c.Name(), err.Error())
	}
	if resp.Status == http.StatusUnauthorized || resp.Status == http.StatusForbidden {
		return "", errs.AuthRequired(c.Name())
	}
	if resp.Status != http.StatusOK {
		return "", errs.Upstream(c.Name(), "token endpoint status "+resp.Header.Get("Status"))
	}
	m, derr := httpx.Decode(resp.Body)
	if derr != nil {
		return "", errs.Upstream(c.Name(), "invalid token JSON")
	}
	tok := httpx.Str(m, "access_token")
	if tok == "" {
		return "", errs.Upstream(c.Name(), "no access_token in response")
	}
	ttl := httpx.Int(m, "expires_in")
	if ttl <= 0 {
		ttl = 3600
	}
	_ = saveToken(cachedToken{AccessToken: tok, ExpiryUnix: time.Now().Add(time.Duration(ttl) * time.Second).Unix()})
	return tok, nil
}

func saveToken(t cachedToken) error {
	_ = os.MkdirAll(filepath.Dir(tokenCachePath()), 0o700)
	b, _ := json.Marshal(t)
	return os.WriteFile(tokenCachePath(), b, 0o600)
}

// Refresh forces a new token (used by `auth refresh`).
func (c *creators) Refresh(ctx context.Context) error {
	_ = os.Remove(tokenCachePath())
	_, err := c.token(ctx)
	return err
}

// post calls a Creators operation, classifying eligibility/throttle errors.
func (c *creators) post(ctx context.Context, op string, body map[string]any) (map[string]any, error) {
	if err := preflight(c.Name()); err != nil {
		return nil, err
	}
	tok, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	body["partnerTag"] = c.partnerTag
	body["partnerType"] = "Associates"
	body["marketplace"] = c.marketplace()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, c.apiHost()+"/catalog/v1/"+op, strings.NewReader(string(buf)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("x-marketplace", c.marketplace())
	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return nil, errs.Retryable(c.Name(), err.Error())
	}
	if resp.Status == http.StatusForbidden && strings.Contains(string(resp.Body), "AssociateNotEligible") {
		return nil, errs.AssociateNotEligible()
	}
	switch {
	case resp.Status == http.StatusUnauthorized:
		_ = os.Remove(tokenCachePath()) // token may be stale
		return nil, errs.AuthRequired(c.Name())
	case resp.Status == http.StatusForbidden:
		return nil, errs.New(errs.ExitPerm, "PERMISSION_DENIED", "Creators API denied the request", "check credentials and partner tag")
	case resp.Status == http.StatusTooManyRequests:
		ra := resp.RetryAfterSeconds()
		markBlocked(c.Name(), max(ra, 1))
		return nil, errs.RateLimited(c.Name()).WithRetryAfter(ra)
	case resp.Status >= 500:
		return nil, errs.Retryable(c.Name(), "server error")
	}
	m, derr := httpx.Decode(resp.Body)
	if derr != nil {
		return nil, errs.Upstream(c.Name(), "invalid JSON")
	}
	if errsArr := httpx.Arr(m, "errors"); len(errsArr) > 0 {
		code := httpx.Str(httpx.AsObj(errsArr[0]), "code")
		if code == "AssociateNotEligible" {
			return nil, errs.AssociateNotEligible()
		}
		return nil, errs.Upstream(c.Name(), httpx.Str(httpx.AsObj(errsArr[0]), "message"))
	}
	markOK(c.Name())
	return m, nil
}

var itemResources = []any{
	"itemInfo.title", "itemInfo.features", "images.primary.large",
	"offersV2.listings.price", "offersV2.listings.availability",
	"offersV2.listings.condition", "offersV2.listings.merchantInfo",
	"browseNodeInfo.websiteSalesRank",
}

func (c *creators) Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	page := 1
	if opts.Cursor != "" {
		if p, e := strconv.Atoi(opts.Cursor); e == nil && p > 0 {
			page = p
		}
	}
	body := map[string]any{
		"keywords": query, "searchIndex": "All",
		"itemCount": 10, "itemPage": page, "resources": itemResources,
	}
	if opts.Category != "" {
		body["browseNodeId"] = opts.Category
	}
	m, err := c.post(ctx, "searchItems", body)
	if err != nil {
		return nil, err
	}
	res := &SearchResult{SchemaVersion: SchemaVersion, Provider: c.Name(), Query: query, Limit: opts.Limit}
	for _, it := range httpx.Arr(m, "searchResult.items") {
		res.Items = append(res.Items, c.toSearchItem(httpx.AsObj(it)))
	}
	res.Count = len(res.Items)
	if page < 10 && len(res.Items) > 0 { // 100-result ceiling (10 pages x 10)
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (c *creators) toSearchItem(o map[string]any) SearchItem {
	asin := httpx.Str(o, "asin")
	return SearchItem{
		ASIN:     asin,
		Title:    httpx.Str(o, "itemInfo.title.displayValue"),
		Price:    httpx.Float(o, "offersV2.listings.0.price.money.amount"),
		Currency: httpx.Str(o, "offersV2.listings.0.price.money.currency"),
		Prime:    httpx.Bool(o, "offersV2.listings.0.deliveryInfo.isPrimeEligible"),
		URL:      "https://" + strings.TrimPrefix(c.marketplace(), "www.") + "/dp/" + asin,
		Image:    httpx.Str(o, "images.primary.large.url"),
	}
}

func (c *creators) GetItem(ctx context.Context, asin string, detailed bool) (*Item, error) {
	m, err := c.post(ctx, "getItems", map[string]any{
		"itemIds": []any{asin}, "itemIdType": "ASIN", "resources": itemResources,
	})
	if err != nil {
		return nil, err
	}
	items := httpx.Arr(m, "itemsResult.items")
	if len(items) == 0 {
		return nil, errs.NotFound("item", asin)
	}
	o := httpx.AsObj(items[0])
	it := &Item{
		SchemaVersion: SchemaVersion, Provider: c.Name(), ASIN: asin,
		Title:     httpx.Str(o, "itemInfo.title.displayValue"),
		Price:     httpx.Float(o, "offersV2.listings.0.price.money.amount"),
		Currency:  httpx.Str(o, "offersV2.listings.0.price.money.currency"),
		SalesRank: httpx.Int(o, "browseNodeInfo.websiteSalesRank.salesRank"),
		URL:       "https://" + strings.TrimPrefix(c.marketplace(), "www.") + "/dp/" + asin,
	}
	if u := httpx.Str(o, "images.primary.large.url"); u != "" {
		it.Images = []string{u}
	}
	if detailed {
		for _, f := range httpx.Arr(o, "itemInfo.features.displayValues") {
			if s, ok := f.(string); ok {
				it.Features = append(it.Features, s)
			}
		}
	}
	return it, nil
}

func (c *creators) GetOffers(ctx context.Context, asin string) (*OffersResult, error) {
	m, err := c.post(ctx, "getItems", map[string]any{
		"itemIds": []any{asin}, "itemIdType": "ASIN", "resources": itemResources,
	})
	if err != nil {
		return nil, err
	}
	items := httpx.Arr(m, "itemsResult.items")
	if len(items) == 0 {
		return nil, errs.NotFound("item", asin)
	}
	o := httpx.AsObj(items[0])
	res := &OffersResult{SchemaVersion: SchemaVersion, Provider: c.Name(), ASIN: asin}
	for _, l := range httpx.Arr(o, "offersV2.listings") {
		lo := httpx.AsObj(l)
		offer := Offer{
			Price:        httpx.Float(lo, "price.money.amount"),
			Currency:     httpx.Str(lo, "price.money.currency"),
			Condition:    strings.ToLower(httpx.Str(lo, "condition.value")),
			Merchant:     httpx.Str(lo, "merchantInfo.name"),
			Prime:        httpx.Bool(lo, "deliveryInfo.isPrimeEligible"),
			Availability: httpx.Str(lo, "availability.type"),
		}
		if httpx.Bool(lo, "isBuyBoxWinner") {
			res.BuyboxPrice = offer.Price
		}
		res.Offers = append(res.Offers, offer)
	}
	if len(res.Offers) == 0 {
		return nil, errs.NotFound("offers for item", asin)
	}
	return res, nil
}

func (c *creators) GetReviews(_ context.Context, _, _ string) (*ReviewsResult, error) {
	return nil, errs.Unsupported("reviews", c.Name())
}

func (c *creators) GetVariations(ctx context.Context, asin string) (*VariationsResult, error) {
	m, err := c.post(ctx, "getVariations", map[string]any{"asin": asin, "resources": itemResources})
	if err != nil {
		return nil, err
	}
	res := &VariationsResult{SchemaVersion: SchemaVersion, Provider: c.Name(), ParentASIN: asin}
	dims := map[int]string{}
	for i, d := range httpx.Arr(m, "variationsResult.variationSummary.variationDimensions") {
		dims[i] = httpx.Str(httpx.AsObj(d), "name")
	}
	for _, v := range httpx.Arr(m, "variationsResult.items") {
		o := httpx.AsObj(v)
		va := httpx.Str(o, "asin")
		res.Variations = append(res.Variations, Variation{
			ASIN:       va,
			Attributes: map[string]string{},
			URL:        "https://" + strings.TrimPrefix(c.marketplace(), "www.") + "/dp/" + va,
		})
	}
	return res, nil
}

func (c *creators) GetBrowseNode(ctx context.Context, nodeID string) (*BrowseNode, error) {
	m, err := c.post(ctx, "getBrowseNodes", map[string]any{
		"browseNodeIds": []any{nodeID}, "resources": []any{"browseNodeInfo.browseNodes"},
	})
	if err != nil {
		return nil, err
	}
	nodes := httpx.Arr(m, "browseNodesResult.browseNodes")
	if len(nodes) == 0 {
		return nil, errs.NotFound("browse node", nodeID)
	}
	o := httpx.AsObj(nodes[0])
	bn := &BrowseNode{
		SchemaVersion: SchemaVersion, Provider: c.Name(),
		NodeID: httpx.Str(o, "id"), Name: httpx.Str(o, "displayName"),
	}
	if anc := httpx.Str(o, "ancestor.displayName"); anc != "" {
		bn.Ancestors = []string{anc}
	}
	for _, ch := range httpx.Arr(o, "children") {
		co := httpx.AsObj(ch)
		bn.Children = append(bn.Children, NodeRef{NodeID: httpx.Str(co, "id"), Name: httpx.Str(co, "displayName")})
	}
	return bn, nil
}
