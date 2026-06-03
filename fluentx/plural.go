package fluentx

import (
	"strconv"
	"strings"

	"github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/cldr/plural"
)

// PluralRules is a CLDR-backed implementation of fluent.PluralRules built on the
// self-contained cldr/plural package (no external dependencies). It returns
// cardinal and ordinal plural categories for a number in a given BCP-47 locale,
// matching JavaScript's Intl.PluralRules (and therefore fluent.js).
type PluralRules struct{}

// NewPluralRules constructs a PluralRules.
func NewPluralRules() *PluralRules { return &PluralRules{} }

var _ fluent.PluralRules = (*PluralRules)(nil)

// defaultDecimalMaxFrac is Intl.NumberFormat's default maximumFractionDigits for
// the "decimal" style. When no fraction-digit options are given, plural operands
// are derived from a value rounded to at most this many fraction digits.
const defaultDecimalMaxFrac = 3

// Cardinal returns the cardinal plural category ("zero"/"one"/"two"/"few"/
// "many"/"other") for n in the given locale.
func (PluralRules) Cardinal(locale string, n float64, opts fluent.NumberOptions) string {
	minFrac, maxFrac := fractionDigits(n, opts)
	return string(plural.CardinalFor(locale, n, minFrac, maxFrac))
}

// Ordinal returns the ordinal plural category for n in the given locale.
func (PluralRules) Ordinal(locale string, n float64, opts fluent.NumberOptions) string {
	minFrac, maxFrac := fractionDigits(n, opts)
	return string(plural.OrdinalFor(locale, n, minFrac, maxFrac))
}

// fractionDigits resolves the minimum/maximum fraction-digit counts used to
// derive the plural operands for n. This mirrors the resolution the number
// formatter would apply so plural selection sees the same visible decimals as
// the rendered number:
//
//   - If opts specify minimum/maximumFractionDigits, those are honored (with the
//     maximum raised to at least the minimum).
//   - Otherwise the value's own shortest decimal representation is used, i.e. the
//     digits that would be shown by default (min 0, and a max large enough to
//     preserve every fractional digit n actually has).
//
// cldr/plural.NewOperands formats n with at most maxFrac fraction digits then
// trims trailing zeros down to minFrac, so passing the count of n's own
// fractional digits as the max reproduces the default Intl rendering.
func fractionDigits(n float64, opts fluent.NumberOptions) (minFrac, maxFrac int) {
	minFrac = 0
	if opts.MinimumFractionDigits != nil && *opts.MinimumFractionDigits > 0 {
		minFrac = *opts.MinimumFractionDigits
	}

	if opts.MaximumFractionDigits != nil && *opts.MaximumFractionDigits >= 0 {
		maxFrac = *opts.MaximumFractionDigits
	} else {
		// Derive from the value itself: use the number of fraction digits in n's
		// shortest representation, capped at Intl's default decimal
		// maximumFractionDigits of 3 (which rounds rather than pads), but never
		// fewer than minFrac. This makes plural selection see exactly the decimals
		// the number formatter would display.
		maxFrac = naturalFractionDigits(n)
		if maxFrac > defaultDecimalMaxFrac {
			maxFrac = defaultDecimalMaxFrac
		}
		if maxFrac < minFrac {
			maxFrac = minFrac
		}
	}
	if maxFrac < minFrac {
		maxFrac = minFrac
	}
	return minFrac, maxFrac
}

// naturalFractionDigits returns the number of fraction digits in the shortest
// decimal representation of n (e.g. 1 -> 0, 1.5 -> 1, 1.25 -> 2). This is the
// count of decimals Intl shows by default when no fraction-digit options are
// given (its default maximumFractionDigits of 3 only ever rounds, never pads,
// so the shortest form already reflects the visible digits for typical values).
func naturalFractionDigits(n float64) int {
	if n < 0 {
		n = -n
	}
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		return len(s) - dot - 1
	}
	return 0
}
