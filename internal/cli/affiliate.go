package cli

// DefaultAssociateTag is rivr's built-in Amazon Associates tag. When the user neither
// supplies their own (--associate-tag / RIVR_ASSOCIATE_TAG) nor opts out
// (--no-associate-tag), product deep links carry this tag.
//
// What it does: if a referred link results in a purchase, Amazon pays the project a small
// referral fee. The buyer pays nothing extra. It funds rivr's development AND helps the
// project meet the Amazon Creators API's qualified-sales minimums (which gate official-API
// access). It is fully disclosed (visible in every URL, in `doctor`, in `schema`, and in the
// docs), replaceable with your own tag, and disablable.
//
// REPLACE this with the project's real registered Associates tag before publishing. The
// "-20" suffix is the Amazon US store locale.
const DefaultAssociateTag = "rivr-20"

// optOutNotice is the one-line, non-pushy message shown (once, to stderr) when a user opts
// out of affiliate attribution and a command would otherwise have emitted a tagged link.
const optOutNotice = "note: affiliate attribution disabled — the built-in tag funds rivr's " +
	"development and helps maintain official Amazon API access, at no extra cost to you. " +
	"Re-enable by dropping --no-associate-tag, or set your own with --associate-tag."

// affiliateMode is a stable label for the active attribution state.
type affiliateMode string

const (
	affiliateDefault  affiliateMode = "default"  // built-in project tag
	affiliateCustom   affiliateMode = "custom"   // user-supplied tag
	affiliateDisabled affiliateMode = "disabled" // opted out
)

// resolveAssociateTag returns the active tag (empty when opted out) and the mode.
// Precedence: --no-associate-tag > --associate-tag/env > built-in default.
func (rt *Runtime) resolveAssociateTag() (tag string, mode affiliateMode) {
	switch {
	case rt.Cfg.NoAssociateTag:
		return "", affiliateDisabled
	case rt.Cfg.AssociateTag != "":
		return rt.Cfg.AssociateTag, affiliateCustom
	default:
		return DefaultAssociateTag, affiliateDefault
	}
}
