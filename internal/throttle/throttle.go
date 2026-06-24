// Package throttle is a persistent, cross-process circuit-breaker. An agent invokes a
// fresh rivr process per call, so an in-memory timer is a no-op — state lives in
// $XDG_STATE_HOME/rivr/. Default behavior is FAIL-FAST (a hung CLI deadlocks an agent
// loop); waiting is opt-in. Used to back off after a provider quota/block response so the
// next process doesn't waste a credit or deepen a block. See contract / cli-implement §1.
package throttle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State is the persisted per-provider breaker state.
type State struct {
	LastRequestUnix  int64 `json:"lastRequestUnix"`
	BlockedUntilUnix int64 `json:"blockedUntilUnix"`
}

func dir() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "rivr")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "rivr")
}

func path(provider string) string {
	return filepath.Join(dir(), "throttle-"+provider+".json")
}

// Load reads the breaker state for a provider (zero value if absent/unreadable).
func Load(provider string) State {
	var s State
	b, err := os.ReadFile(path(provider))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

// Save persists the breaker state (best-effort; failures are non-fatal).
func Save(provider string, s State) {
	_ = os.MkdirAll(dir(), 0o700)
	b, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(path(provider), b, 0o600)
}

// RetryAfter returns seconds remaining on an active block (0 if not blocked).
func (s State) RetryAfter(now time.Time) int {
	if s.BlockedUntilUnix <= 0 {
		return 0
	}
	d := s.BlockedUntilUnix - now.Unix()
	if d <= 0 {
		return 0
	}
	return int(d)
}

// Block records a cooldown of `seconds` for a provider, starting now.
func Block(provider string, seconds int, now time.Time) {
	if seconds <= 0 {
		return
	}
	s := Load(provider)
	s.BlockedUntilUnix = now.Add(time.Duration(seconds) * time.Second).Unix()
	Save(provider, s)
}

// Touch records that a request was just made (for min-interval pacing by callers).
func Touch(provider string, now time.Time) {
	s := Load(provider)
	s.LastRequestUnix = now.Unix()
	Save(provider, s)
}

// Clear removes any active block (e.g. after a successful request).
func Clear(provider string) {
	s := Load(provider)
	if s.BlockedUntilUnix != 0 {
		s.BlockedUntilUnix = 0
		Save(provider, s)
	}
}
