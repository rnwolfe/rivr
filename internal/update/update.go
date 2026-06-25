// Package update does a passive, cached "is there a newer release?" check against GitHub.
// It is deliberately conservative for an agent-first tool: never auto-updates, never blocks a
// command's data path, caches results (24h) in $XDG_STATE_HOME/rivr/, and fails silently.
// The CLI only ever SURFACES an available upgrade (to a human, on stderr) — it never runs one.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const ttl = 24 * time.Hour

// apiURL is the GitHub "latest release" endpoint; overridable via env for tests.
func apiURL() string {
	if v := safeReleaseURL(os.Getenv("RIVR_UPDATE_URL")); v != "" {
		return v
	}
	return "https://api.github.com/repos/rnwolfe/rivr/releases/latest"
}

// safeReleaseURL allows a RIVR_UPDATE_URL override only over https (any host) or http to
// localhost (for tests). A misconfigured/hostile env var (file://, http://169.254.169.254, …)
// is ignored — the version check falls back to the default — so the override can't be used for
// SSRF or local-file reads. Returns "" for empty/disallowed input.
func safeReleaseURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	switch u.Scheme {
	case "https":
		return raw
	case "http":
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return raw
		}
	}
	return ""
}

// UpgradeHint is the command(s) a human runs to upgrade. rivr can't reliably detect the
// install method, so it offers both common ones.
const UpgradeHint = "go install github.com/rnwolfe/rivr/cmd/rivr@latest  (or: brew upgrade rivr)"

type cache struct {
	Latest      string `json:"latest"`
	CheckedUnix int64  `json:"checkedUnix"`
}

func cachePath() string {
	d := os.Getenv("XDG_STATE_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(d, "rivr", "update-check.json")
}

// loadCache returns the cached entry and whether it parsed (freshness is decided by the
// caller against an injected clock, for testability).
func loadCache() (cache, bool) {
	var c cache
	b, err := os.ReadFile(cachePath())
	if err != nil || json.Unmarshal(b, &c) != nil || c.Latest == "" {
		return c, false
	}
	return c, true
}

func saveCache(latest string, now time.Time) {
	_ = os.MkdirAll(filepath.Dir(cachePath()), 0o700)
	b, _ := json.Marshal(cache{Latest: latest, CheckedUnix: now.Unix()})
	_ = os.WriteFile(cachePath(), b, 0o600)
}

func fetchLatest(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL(), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "rivr-version-check") // GitHub's REST API rejects requests with no UA
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release source: %s", resp.Status)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.TagName, nil
}

// Check reports the latest release and whether it's newer than current. It uses the 24h cache
// unless force=true. Network/parse failures return err with updateAvailable=false (callers
// treat any error as "don't nag"). now is injected for testability.
func Check(ctx context.Context, current string, force bool, now time.Time) (latest string, updateAvailable bool, err error) {
	if c, ok := loadCache(); ok && !force && now.Sub(time.Unix(c.CheckedUnix, 0)) < ttl {
		latest = c.Latest
	} else {
		latest, err = fetchLatest(ctx)
		if err != nil {
			return "", false, err
		}
		saveCache(latest, now)
	}
	return latest, newer(current, latest), nil
}

// newer reports whether b is a strictly greater release version than a. Unparseable versions
// (dev builds, pseudo-versions) yield false — we never nag when we can't compare cleanly.
func newer(a, b string) bool {
	pa, oka := parse(a)
	pb, okb := parse(b)
	if !oka || !okb {
		return false
	}
	for i := 0; i < 3; i++ {
		if pb[i] != pa[i] {
			return pb[i] > pa[i]
		}
	}
	return false
}

// parse extracts [major,minor,patch] from a clean "v1.2.3"/"1.2.3". Returns ok=false for
// anything with a pre-release/build suffix (Go pseudo-versions, `-rc1`, `+dirty`) or non-
// numeric parts — so a dev/source build is never nagged.
func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if strings.ContainsAny(v, "-+") {
		return out, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
