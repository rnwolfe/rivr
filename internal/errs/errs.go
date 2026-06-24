// Package errs defines the stable exit-code table and the structured CLI error type.
// Exit codes are a contract: distinct, documented, and append-only. See contract.md §4.
package errs

// Stable exit codes. rivr additions over the base table: ExitUpstream (9),
// ExitUnsupported (11).
const (
	ExitOK              = 0
	ExitGeneric         = 1
	ExitUsage           = 2
	ExitEmpty           = 3 // search/list returned zero matches (success-adjacent signal)
	ExitAuth            = 4
	ExitNotFound        = 5
	ExitPerm            = 6 // permission denied OR ASSOCIATE_NOT_ELIGIBLE (official backend)
	ExitRate            = 7 // provider quota exhausted / official TPS limit / blocked
	ExitRetry           = 8 // transient upstream (5xx/network); safe to retry
	ExitUpstream        = 9 // upstream returned something we couldn't classify/parse (rivr addition)
	ExitConfig          = 10
	ExitUnsupported     = 11 // capability not supported by the active provider (rivr addition)
	ExitMutationBlocked = 12 // present for contract uniformity; rivr is read-only (never returned)
	ExitInputRequired   = 13
	ExitCancelled       = 130
)

// Table returns the exit-code table for the `schema` command.
func Table() map[string]int {
	return map[string]int{
		"ok":                ExitOK,
		"generic_error":     ExitGeneric,
		"usage":             ExitUsage,
		"empty_results":     ExitEmpty,
		"auth_required":     ExitAuth,
		"not_found":         ExitNotFound,
		"permission":        ExitPerm,
		"rate_limited":      ExitRate,
		"retryable":         ExitRetry,
		"upstream_error":    ExitUpstream,
		"config_error":      ExitConfig,
		"unsupported":       ExitUnsupported,
		"mutation_blocked":  ExitMutationBlocked,
		"input_required":    ExitInputRequired,
		"cancelled":         ExitCancelled,
	}
}

// CLIError is a structured error carrying a machine-readable code, a remediation hint,
// and the process exit code to return. RetryAfter (seconds) is surfaced to agents when the
// upstream tells us when to retry (Retry-After header, throttle cooldown).
type CLIError struct {
	Message     string
	Code        string
	Remediation string
	Exit        int
	RetryAfter  int // seconds; 0 = unset
}

func (e *CLIError) Error() string { return e.Message }

// New constructs a CLIError.
func New(exit int, code, msg, remediation string) *CLIError {
	return &CLIError{Message: msg, Code: code, Remediation: remediation, Exit: exit}
}

// WithRetryAfter annotates an error with a retry delay (seconds) for the agent to schedule.
func (e *CLIError) WithRetryAfter(seconds int) *CLIError {
	e.RetryAfter = seconds
	return e
}

// Retryable marks a transient upstream failure (5xx / network) that is safe to retry.
func Retryable(provider, detail string) *CLIError {
	return New(ExitRetry, "RETRYABLE", "transient failure from "+provider+": "+detail,
		"retry the request; if it persists, run `rivr doctor`")
}

// Upstream is returned when the provider responds with something we can't classify.
func Upstream(provider, detail string) *CLIError {
	return New(ExitUpstream, "UPSTREAM_ERROR", "unexpected response from "+provider+": "+detail,
		"retry; if it persists the provider API may have changed — please file an issue")
}

// SchemaDrift is returned when an expected field/container is absent from the response,
// so an upstream rename surfaces as a typed error instead of a silent empty result.
func SchemaDrift(provider, field string) *CLIError {
	return New(ExitUpstream, "SCHEMA_DRIFT",
		provider+" response is missing expected field "+field,
		"the provider's response shape may have changed; update rivr or file an issue")
}

// Blocked is returned when a (scraping) target has blocked us; carries a cooldown.
func Blocked(provider string, retryAfter int) *CLIError {
	return New(ExitRate, "BLOCKED", provider+" blocked the request (bot detection)",
		"wait for the cooldown and retry, or use an API-based provider (--provider serpapi)").
		WithRetryAfter(retryAfter)
}

// MutationBlocked is returned when a mutating op runs without --allow-mutations. rivr is
// read-only, so this is never triggered in practice — kept for contract uniformity.
func MutationBlocked(op string) *CLIError {
	return New(ExitMutationBlocked, "MUTATION_BLOCKED",
		op+" is a mutating operation and is blocked by default",
		"re-run with --allow-mutations (rivr is read-only, so this should not occur)")
}

// NotFound is returned when a product/node id does not exist.
func NotFound(kind, id string) *CLIError {
	return New(ExitNotFound, "NOT_FOUND", kind+" "+id+" not found",
		"verify the ASIN/node id; try `rivr search` to find a valid one")
}

// Empty is returned when a search/list yields zero matches (distinct from an error).
func Empty(what string) *CLIError {
	return New(ExitEmpty, "EMPTY_RESULTS", "no "+what+" matched",
		"broaden the query or relax filters (--min-rating, --min-price/--max-price)")
}

// AuthRequired names the exact login command the agent should run.
func AuthRequired(provider string) *CLIError {
	return New(ExitAuth, "AUTH_REQUIRED",
		"no credentials configured for provider "+provider,
		"run `rivr auth login --provider "+provider+"` and pipe the API key on stdin")
}

// RateLimited maps provider quota / official TPS limits to a retry signal.
func RateLimited(provider string) *CLIError {
	return New(ExitRate, "RATE_LIMITED",
		"provider "+provider+" rate limit or quota exhausted",
		"back off and retry, lower --limit, or check the provider quota")
}

// Unsupported is returned when the active provider cannot serve a capability
// (e.g. `reviews` on the official Creators backend, which returns no review text).
func Unsupported(capability, provider string) *CLIError {
	return New(ExitUnsupported, "UNSUPPORTED_BY_PROVIDER",
		capability+" is not supported by provider "+provider,
		"switch to a provider that supports it (e.g. --provider serpapi for reviews)")
}

// AssociateNotEligible is the official Creators backend eligibility wall (HTTP 403).
// It is permission-shaped (exit 6) but carries its own code + dashboard remediation.
func AssociateNotEligible() *CLIError {
	return New(ExitPerm, "ASSOCIATE_NOT_ELIGIBLE",
		"the Amazon Associates account is not eligible for Creators API access",
		"the official API needs >=10 qualified sales in the trailing 30 days; "+
			"check the Associates dashboard, or use a third-party provider instead")
}

// InputRequired is returned when --no-input is set but input is needed.
func InputRequired(what string) *CLIError {
	return New(ExitInputRequired, "INPUT_REQUIRED", what+" is required",
		"pass it as a flag/argument (running with --no-input, so prompts are disabled)")
}
