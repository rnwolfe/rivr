package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnwolfe/rivr/internal/errs"
)

const amazonSearchHTML = `<html><body>
<div class="s-result-item" data-asin="B01ABCDEF1">
  <h2><a href="/dp/B01ABCDEF1"><span>Anker USB-C Cable</span></a></h2>
  <span class="a-price"><span class="a-offscreen">$12.99</span></span>
  <span class="a-icon-alt">4.6 out of 5 stars</span>
  <span class="a-size-base s-underline-text">21,034</span>
  <i class="a-icon-prime"></i>
  <img class="s-image" src="https://m.media-amazon.com/x.jpg"/>
</div>
<div class="s-result-item" data-asin=""><span>ad slot</span></div>
</body></html>`

func TestScrapeSearchParsing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("RIVR_SCRAPE_ENABLE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(amazonSearchHTML))
	}))
	defer srv.Close()
	t.Setenv("RIVR_SCRAPE_BASE", srv.URL)

	s := newScrape()
	res, err := s.Search(context.Background(), "usb-c", SearchOpts{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 { // the empty-data-asin ad slot must be skipped
		t.Fatalf("want 1 item, got %d: %+v", len(res.Items), res.Items)
	}
	it := res.Items[0]
	if it.ASIN != "B01ABCDEF1" || it.Title != "Anker USB-C Cable" || it.Price != 12.99 {
		t.Fatalf("bad parse: %+v", it)
	}
	if it.Rating != 4.6 || it.ReviewCount != 21034 || !it.Prime {
		t.Fatalf("bad rating/reviews/prime: %+v", it)
	}
	if it.URL != srv.URL+"/dp/B01ABCDEF1" {
		t.Fatalf("bad url: %q", it.URL)
	}
}

func TestScrapeDisabledByDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("RIVR_SCRAPE_ENABLE", "") // default off
	s := newScrape()
	if s.Configured() {
		t.Fatal("scrape should be off by default")
	}
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	var ce *errs.CLIError
	if !errors.As(err, &ce) || ce.Code != "SCRAPE_DISABLED" {
		t.Fatalf("want SCRAPE_DISABLED, got %v", err)
	}
}

func TestScrapeBlockDetection(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("RIVR_SCRAPE_ENABLE", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>Robot Check — enter the characters you see</body></html>`))
	}))
	defer srv.Close()
	t.Setenv("RIVR_SCRAPE_BASE", srv.URL)

	s := newScrape()
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	var ce *errs.CLIError
	if !errors.As(err, &ce) || ce.Code != "BLOCKED" {
		t.Fatalf("want BLOCKED on captcha page, got %v", err)
	}
	if ce.RetryAfter <= 0 {
		t.Fatalf("BLOCKED should carry retryAfter, got %d", ce.RetryAfter)
	}
	// the cooldown must persist so the next process fails fast
	if Cooldown("scrape") <= 0 {
		t.Fatal("expected a persisted cooldown after a block")
	}
}
