package provider

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rnwolfe/rivr/internal/auth"
	"github.com/rnwolfe/rivr/internal/errs"
	"github.com/rnwolfe/rivr/internal/httpx"
	"github.com/rnwolfe/rivr/internal/throttle"
)

// Validator is implemented by backends that can actively test credentials/connectivity
// (used by `auth status` and `doctor`). Refresher re-mints a cached token.
type Validator interface {
	Validate(ctx context.Context) error
}

type Refresher interface {
	Refresh(ctx context.Context) error
}

// ValidatorFor / RefresherFor are capability probes for the cli layer.
func ValidatorFor(p Provider) (Validator, bool) { v, ok := p.(Validator); return v, ok }
func RefresherFor(p Provider) (Refresher, bool) { v, ok := p.(Refresher); return v, ok }

// Cooldown reports the seconds remaining on a persistent block for a provider (0 if none).
func Cooldown(provider string) int { return throttle.Load(provider).RetryAfter(time.Now()) }

// serpApi: validate via the free account.json endpoint (not the search envelope).
func (s *serpApi) Validate(ctx context.Context) error {
	key, _ := auth.Get(s.Name(), auth.FieldAPIKey)
	if key == "" {
		return errs.AuthRequired(s.Name())
	}
	q := url.Values{"api_key": {key}}
	req, _ := http.NewRequest(http.MethodGet, s.base+"/account.json?"+q.Encode(), nil)
	resp, err := s.http.Do(ctx, req)
	if err != nil {
		return errs.Retryable(s.Name(), err.Error())
	}
	if resp.Status == http.StatusUnauthorized || resp.Status == http.StatusForbidden {
		return errs.AuthRequired(s.Name())
	}
	if resp.Status != http.StatusOK {
		return errs.Upstream(s.Name(), "account check status "+strconv.Itoa(resp.Status))
	}
	m, _ := httpx.Decode(resp.Body)
	if httpx.Str(m, "error") != "" {
		return errs.AuthRequired(s.Name())
	}
	return nil
}

// rainforest: validate via the free /account endpoint.
func (r *rainforest) Validate(ctx context.Context) error {
	key, _ := auth.Get(r.Name(), auth.FieldAPIKey)
	if key == "" {
		return errs.AuthRequired(r.Name())
	}
	q := url.Values{"api_key": {key}}
	req, _ := http.NewRequest(http.MethodGet, r.base+"/account?"+q.Encode(), nil)
	resp, err := r.http.Do(ctx, req)
	if err != nil {
		return errs.Retryable(r.Name(), err.Error())
	}
	if resp.Status == http.StatusUnauthorized {
		return errs.AuthRequired(r.Name())
	}
	if resp.Status != http.StatusOK {
		return errs.Upstream(r.Name(), "account check status "+strconv.Itoa(resp.Status))
	}
	m, _ := httpx.Decode(resp.Body)
	if !httpx.Bool(m, "request_info.success") {
		return errs.AuthRequired(r.Name())
	}
	return nil
}

// creators: validate by minting an OAuth token (the eligibility wall surfaces on real calls).
func (c *creators) Validate(ctx context.Context) error {
	_, err := c.token(ctx)
	return err
}

// stub: always valid (offline backend).
func (s *stub) Validate(_ context.Context) error { return nil }

// scrape.Validate lives in scrape.go (it gates on the opt-in flag).
