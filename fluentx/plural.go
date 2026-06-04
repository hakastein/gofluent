package fluentx

import (
	"github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/cldr/number"
)

// PluralRules is a CLDR-backed implementation of fluent.PluralRules built on the
// self-contained cldr/plural package (no external dependencies). It returns
// cardinal and ordinal plural categories for a number in a given BCP-47 locale,
// matching JavaScript's Intl.PluralRules (and therefore fluent.js).
type PluralRules struct{}

// NewPluralRules constructs a PluralRules.
func NewPluralRules() *PluralRules { return &PluralRules{} }

var _ fluent.PluralRules = (*PluralRules)(nil)

// Cardinal returns the cardinal plural category ("zero"/"one"/"two"/"few"/
// "many"/"other") for n in the given locale.
func (PluralRules) Cardinal(locale string, n float64, opts fluent.NumberOptions) string {
	return number.CardinalCategory(locale, n, pluralOptions(opts))
}

// Ordinal returns the ordinal plural category for n in the given locale.
func (PluralRules) Ordinal(locale string, n float64, opts fluent.NumberOptions) string {
	return number.OrdinalCategory(locale, n, pluralOptions(opts))
}

// pluralOptions maps the fluent.NumberOptions digit constraints onto the
// cldr/number.Options understood by number.PluralOperands. Plural selection must
// see exactly the digits the number formatter would display, so we forward the
// integer/fraction/significant-digit options verbatim and let cldr/number apply
// the same resolution and rounding. Style/Currency are intentionally dropped:
// Intl.PluralRules has no style (it never scales percents or applies currency
// fraction defaults).
func pluralOptions(opts fluent.NumberOptions) number.Options {
	return number.Options{
		MinimumIntegerDigits:     opts.MinimumIntegerDigits,
		MinimumFractionDigits:    opts.MinimumFractionDigits,
		MaximumFractionDigits:    opts.MaximumFractionDigits,
		MinimumSignificantDigits: opts.MinimumSignificantDigits,
		MaximumSignificantDigits: opts.MaximumSignificantDigits,
	}
}
