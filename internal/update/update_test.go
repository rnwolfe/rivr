package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.2.0", "v0.3.0", true},
		{"0.2.0", "0.2.1", true},
		{"v1.0.0", "v0.9.9", false},
		{"v0.2.0", "v0.2.0", false},
		{"v0.2.0", "v0.2.0-rc1", false}, // prerelease suffix dropped → equal
		{"v0.1.3-0.2026...+dirty", "v0.2.0", false}, // dev/pseudo current → never nag
	}
	for _, c := range cases {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckUsesEndpointAndCache(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte(`{"tag_name":"v0.9.0"}`))
	}))
	defer srv.Close()
	t.Setenv("RIVR_UPDATE_URL", srv.URL)

	now := time.Unix(1_750_000_000, 0)
	latest, avail, err := Check(context.Background(), "v0.2.0", false, now)
	if err != nil || latest != "v0.9.0" || !avail {
		t.Fatalf("Check = (%q,%v,%v)", latest, avail, err)
	}
	// Second call within TTL must hit the cache, not the network.
	if _, _, err := Check(context.Background(), "v0.2.0", false, now); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 network hit (then cache), got %d", hits)
	}
	// force=true bypasses the cache.
	if _, _, err := Check(context.Background(), "v0.2.0", true, now); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("force should re-fetch; hits=%d", hits)
	}
}
