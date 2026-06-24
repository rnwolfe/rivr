package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rnwolfe/rivr/internal/errs"
)

// isolate points credential + throttle state at env/temp so tests need no network/keyring.
func isolate(t *testing.T, provider, keyEnv, keyVal string) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv(keyEnv, keyVal)
}

func wantCode(t *testing.T, err error, code string, exit int) {
	t.Helper()
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		t.Fatalf("want *errs.CLIError, got %v", err)
	}
	if ce.Code != code {
		t.Fatalf("code = %q, want %q", ce.Code, code)
	}
	if ce.Exit != exit {
		t.Fatalf("exit = %d, want %d", ce.Exit, exit)
	}
}

func TestSerpApiSearchNormalizes(t *testing.T) {
	isolate(t, "serpapi", "RIVR_SERPAPI_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("engine") != "amazon" {
			t.Errorf("engine = %q", r.URL.Query().Get("engine"))
		}
		w.Write([]byte(`{
			"organic_results":[
				{"asin":"B01","title":"Cable A","extracted_price":12.99,"rating":4.6,"reviews":21034,"prime":true,"link":"https://amazon.com/dp/B01","thumbnail":"https://img/a.jpg"}
			],
			"serpapi_pagination":{"next":"https://serpapi.com/search?page=2"}
		}`))
	}))
	defer srv.Close()
	s := newSerpApi()
	s.base = srv.URL
	res, err := s.Search(context.Background(), "usb-c", SearchOpts{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ASIN != "B01" || res.Items[0].Price != 12.99 {
		t.Fatalf("bad normalization: %+v", res.Items)
	}
	if res.Items[0].Currency != "USD" {
		t.Fatalf("currency = %q, want USD", res.Items[0].Currency)
	}
	if res.NextCursor != "2" {
		t.Fatalf("nextCursor = %q, want 2", res.NextCursor)
	}
}

func TestSerpApiRateLimitMapsToExit7(t *testing.T) {
	isolate(t, "serpapi", "RIVR_SERPAPI_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":"Your account has run out of searches."}`))
	}))
	defer srv.Close()
	s := newSerpApi()
	s.base = srv.URL
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	wantCode(t, err, "RATE_LIMITED", errs.ExitRate)
}

func TestSerpApiInvalidKeyMapsToAuth(t *testing.T) {
	isolate(t, "serpapi", "RIVR_SERPAPI_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid API key."}`))
	}))
	defer srv.Close()
	s := newSerpApi()
	s.base = srv.URL
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	wantCode(t, err, "AUTH_REQUIRED", errs.ExitAuth)
}

func TestSerpApiAuthRequiredWhenNoKey(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("RIVR_SERPAPI_API_KEY", "")
	s := newSerpApi()
	if s.Configured() {
		t.Fatal("should not be configured without a key")
	}
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	wantCode(t, err, "AUTH_REQUIRED", errs.ExitAuth)
}

func TestRainforestSearchNormalizes(t *testing.T) {
	isolate(t, "rainforest", "RIVR_RAINFOREST_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"request_info":{"success":true},
			"search_results":[
				{"asin":"B02","title":"Cable B","price":{"value":24.5,"currency":"usd"},"rating":4.2,"ratings_total":813,"is_prime":false,"link":"https://amazon.com/dp/B02","image":"https://img/b.jpg"}
			],
			"pagination":{"current_page":1,"total_pages":3}
		}`))
	}))
	defer srv.Close()
	r := newRainforest()
	r.base = srv.URL
	res, err := r.Search(context.Background(), "usb-c", SearchOpts{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ASIN != "B02" || res.Items[0].Price != 24.5 {
		t.Fatalf("bad normalization: %+v", res.Items)
	}
	if res.Items[0].Currency != "USD" { // normalized from lowercase "usd"
		t.Fatalf("currency = %q, want USD", res.Items[0].Currency)
	}
	if res.NextCursor != "2" {
		t.Fatalf("nextCursor = %q, want 2", res.NextCursor)
	}
}

func TestRainforestInvalidKeyMapsToAuth(t *testing.T) {
	isolate(t, "rainforest", "RIVR_RAINFOREST_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"request_info":{"success":false,"message":"Invalid API key."}}`))
	}))
	defer srv.Close()
	r := newRainforest()
	r.base = srv.URL
	_, err := r.GetItem(context.Background(), "B02", false)
	wantCode(t, err, "AUTH_REQUIRED", errs.ExitAuth)
}

func TestRainforestReviewsScopeFull(t *testing.T) {
	isolate(t, "rainforest", "RIVR_RAINFOREST_API_KEY", "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"request_info":{"success":true},"reviews":[{"rating":5,"title":"Great","body":"Works","profile":{"name":"alice"},"date":{"utc":"2026-05-01T00:00:00Z"},"verified_purchase":true}],"pagination":{"current_page":1,"total_pages":1}}`))
	}))
	defer srv.Close()
	r := newRainforest()
	r.base = srv.URL
	res, err := r.GetReviews(context.Background(), "B02", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Scope != "full" {
		t.Fatalf("scope = %q, want full", res.Scope)
	}
	if len(res.Reviews) != 1 || res.Reviews[0].Author != "alice" {
		t.Fatalf("bad review normalization: %+v", res.Reviews)
	}
}

func TestThrottleFailFast(t *testing.T) {
	isolate(t, "serpapi", "RIVR_SERPAPI_API_KEY", "k")
	markBlocked("serpapi", 120)
	s := newSerpApi()
	_, err := s.Search(context.Background(), "x", SearchOpts{})
	var ce *errs.CLIError
	if !errors.As(err, &ce) || ce.Code != "BLOCKED" {
		t.Fatalf("want BLOCKED, got %v", err)
	}
	if ce.RetryAfter <= 0 || ce.RetryAfter > 120 {
		t.Fatalf("retryAfter = %d, want ~120", ce.RetryAfter)
	}
}
