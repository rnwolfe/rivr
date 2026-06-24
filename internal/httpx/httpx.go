// Package httpx is the shared HTTP core for provider backends: a client with bounded
// retries + backoff on transient failures, Retry-After handling, and defensive JSON
// helpers so an upstream field rename surfaces as SCHEMA_DRIFT, not a panic. Providers
// classify status codes into domain errors (internal/errs); this layer is transport-only.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

// Client is a thin retrying HTTP client. Zero value is not usable — call New.
type Client struct {
	HTTP       *http.Client
	UserAgent  string
	MaxRetries int           // retries AFTER the first attempt
	BaseDelay  time.Duration // backoff base
}

// New returns a Client with sane defaults for I/O-bound provider calls.
func New() *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		UserAgent:  "rivr (+https://github.com/rnwolfe/rivr)",
		MaxRetries: 2,
		BaseDelay:  500 * time.Millisecond,
	}
}

// Response is a fully-read HTTP response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// RetryAfterSeconds parses the Retry-After header (delta-seconds form), 0 if absent/invalid.
func (r *Response) RetryAfterSeconds() int {
	if r == nil {
		return 0
	}
	if v := r.Header.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

// Do executes req, retrying on network errors, 429, and 5xx up to MaxRetries with
// exponential backoff (honoring Retry-After when present). It returns the final Response
// regardless of status (callers map status → domain errors). A network/exhausted failure
// returns a non-nil error.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	// Buffer the body so it can be replayed across retries.
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = b
	}

	var last *Response
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			delay := c.backoff(attempt, last)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		req2 := req.Clone(ctx)
		if body != nil {
			req2.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err := c.HTTP.Do(req2)
		if err != nil {
			lastErr = err
			continue // network error → retry
		}
		b, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			lastErr = rerr
			continue
		}
		last = &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
		lastErr = nil
		if last.Status == http.StatusTooManyRequests || last.Status >= 500 {
			continue // transient → retry
		}
		return last, nil
	}
	if last != nil {
		return last, nil // exhausted retries on a transient status; let caller classify
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", c.MaxRetries+1, lastErr)
}

func (c *Client) backoff(attempt int, last *Response) time.Duration {
	if ra := last.RetryAfterSeconds(); ra > 0 {
		return time.Duration(ra) * time.Second
	}
	return time.Duration(math.Pow(2, float64(attempt-1))) * c.BaseDelay
}

// --- defensive JSON access ---------------------------------------------------
// Decode parses a JSON object body. Helpers below never panic on missing/mistyped
// fields — they return zero values, so a normalizer degrades gracefully and the caller
// can decide whether absence is SCHEMA_DRIFT.

func Decode(b []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// dig walks a dot-path through nested objects, returning the leaf value and ok.
func dig(m map[string]any, path string) (any, bool) {
	var cur any = m
	start := 0
	for i := 0; i <= len(path); i++ {
		if i < len(path) && path[i] != '.' {
			continue
		}
		key := path[start:i]
		start = i + 1
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// Str returns the string at path (or "" ).
func Str(m map[string]any, path string) string {
	v, ok := dig(m, path)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Float returns the number at path, coercing numeric strings (e.g. "$12.99" is NOT coerced;
// pass already-numeric fields). Returns 0 if absent.
func Float(m map[string]any, path string) float64 {
	v, ok := dig(m, path)
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	}
	return 0
}

// Int returns the integer at path.
func Int(m map[string]any, path string) int { return int(Float(m, path)) }

// Bool returns the boolean at path.
func Bool(m map[string]any, path string) bool {
	v, ok := dig(m, path)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

// Has reports whether path exists (for SCHEMA_DRIFT detection on required containers).
func Has(m map[string]any, path string) bool {
	_, ok := dig(m, path)
	return ok
}

// Arr returns the array at path as []any (nil if absent or not an array).
func Arr(m map[string]any, path string) []any {
	v, ok := dig(m, path)
	if !ok {
		return nil
	}
	a, _ := v.([]any)
	return a
}

// AsObj coerces an arbitrary element to a JSON object (nil if not).
func AsObj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// StrOf reads a string field from an already-extracted object element.
func StrOf(v any, key string) string { return Str(AsObj(v), key) }
