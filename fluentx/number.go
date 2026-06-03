package fluentx

import (
	"github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/cldr/number"
)

// NumberFormatter is a CLDR-backed implementation of fluent.NumberFormatter
// built on the self-contained cldr/number package (no external dependencies).
// It renders decimals, percents and currency amounts with locale-aware
// grouping, decimal separators and currency symbols, matching JavaScript's
// Intl.NumberFormat (and therefore fluent.js).
type NumberFormatter struct{}

// NewNumberFormatter constructs a NumberFormatter.
func NewNumberFormatter() *NumberFormatter { return &NumberFormatter{} }

var _ fluent.NumberFormatter = (*NumberFormatter)(nil)

// FormatNumber renders n for the given locale honoring opts (style, currency,
// grouping, fraction/significant/integer digit constraints).
func (NumberFormatter) FormatNumber(locale string, n float64, opts fluent.NumberOptions) string {
	return number.Format(locale, n, numberOptions(opts))
}

// numberOptions maps the core fluent.NumberOptions onto cldr/number.Options. The
// two structs share field names; this is a straight field-by-field copy of the
// fields cldr/number understands. (fluent's Unit/UnitDisplay/Type fields have no
// cldr/number counterpart and are not number-rendering concerns here.)
func numberOptions(opts fluent.NumberOptions) number.Options {
	return number.Options{
		Style:                    opts.Style,
		Currency:                 opts.Currency,
		CurrencyDisplay:          opts.CurrencyDisplay,
		UseGrouping:              opts.UseGrouping,
		MinimumIntegerDigits:     opts.MinimumIntegerDigits,
		MinimumFractionDigits:    opts.MinimumFractionDigits,
		MaximumFractionDigits:    opts.MaximumFractionDigits,
		MinimumSignificantDigits: opts.MinimumSignificantDigits,
		MaximumSignificantDigits: opts.MaximumSignificantDigits,
	}
}
