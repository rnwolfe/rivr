// Package fence wraps attacker-controllable free text (product titles, bullet features,
// descriptions, and especially review bodies) so an agent treats it as untrusted data,
// not instructions. ON by default in agent mode (contract §8, spec.md "Prompt-injection
// surface"). Numeric/ID/URL fields are never fenced.
package fence

// Marker tags delimit untrusted spans. Stable strings so an agent can rely on them.
const (
	open  = "‹untrusted›"  // ‹untrusted›
	close = "‹/untrusted›" // ‹/untrusted›
)

// Wrap fences a single free-text value. Empty strings pass through unchanged so absent
// fields stay empty rather than becoming a hollow fence.
func Wrap(s string) string {
	if s == "" {
		return s
	}
	return open + s + close
}

// WrapAll fences each element of a free-text slice in place and returns it.
func WrapAll(ss []string) []string {
	for i := range ss {
		ss[i] = Wrap(ss[i])
	}
	return ss
}
