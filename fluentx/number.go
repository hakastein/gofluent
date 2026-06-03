package fluentx

import (
	"github.com/hakastein/gofluent"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// NumberFormatter is a CLDR-backed implementation of fluent.NumberFormatter
// built on golang.org/x/text/number, golang.org/x/text/message and
// golang.org/x/text/currency. It renders decimals, percents and currency
// amounts with locale-aware grouping and decimal separators.
//
// Known x/text characteristics (differing from JS Intl.NumberFormat):
//   - Currency output places a space between the symbol and the amount
//     (e.g. "$ 1,234.00", "€ 1.234,50").
//   - The significant-digits options are approximated via number.Precision,
//     which controls total significant digits but cannot express an independent
//     minimum and maximum; the maximum (or minimum) takes precedence.
type NumberFormatter struct{}

// NewNumberFormatter constructs a NumberFormatter.
func NewNumberFormatter() *NumberFormatter { return &NumberFormatter{} }

var _ fluent.NumberFormatter = (*NumberFormatter)(nil)

// FormatNumber renders n for the given locale honoring opts (style, currency,
// grouping, fraction/significant/integer digit constraints).
func (NumberFormatter) FormatNumber(locale string, n float64, opts fluent.NumberOptions) string {
	tag, err := language.Parse(locale)
	if err != nil {
		tag = language.English
	}
	p := message.NewPrinter(tag)

	switch opts.Style {
	case "currency":
		return formatCurrency(p, n, opts)
	case "percent":
		// Like Intl.NumberFormat, the value is treated as a ratio: 0.125 -> 12%.
		return p.Sprint(number.Percent(n, numberOpts(opts)...))
	default:
		return p.Sprint(number.Decimal(n, numberOpts(opts)...))
	}
}

// numberOpts maps NumberOptions to the x/text number.Option list shared by the
// decimal and percent styles.
func numberOpts(opts fluent.NumberOptions) []number.Option {
	var out []number.Option

	if opts.MinimumIntegerDigits != nil && *opts.MinimumIntegerDigits > 0 {
		out = append(out, number.MinIntegerDigits(*opts.MinimumIntegerDigits))
	}
	if opts.MinimumFractionDigits != nil && *opts.MinimumFractionDigits >= 0 {
		out = append(out, number.MinFractionDigits(*opts.MinimumFractionDigits))
	}
	if opts.MaximumFractionDigits != nil && *opts.MaximumFractionDigits >= 0 {
		out = append(out, number.MaxFractionDigits(*opts.MaximumFractionDigits))
	}
	// Significant digits: x/text exposes only a single Precision (total
	// significant digits). Prefer the maximum if set, else the minimum.
	if opts.MaximumSignificantDigits != nil && *opts.MaximumSignificantDigits > 0 {
		out = append(out, number.Precision(*opts.MaximumSignificantDigits))
	} else if opts.MinimumSignificantDigits != nil && *opts.MinimumSignificantDigits > 0 {
		out = append(out, number.Precision(*opts.MinimumSignificantDigits))
	}
	if opts.UseGrouping != nil && !*opts.UseGrouping {
		out = append(out, number.NoSeparator())
	}
	return out
}

// formatCurrency renders n as a currency amount. The currency code comes from
// opts.Currency (ISO 4217); an invalid/empty code falls back to a plain decimal.
func formatCurrency(p *message.Printer, n float64, opts fluent.NumberOptions) string {
	unit, err := currency.ParseISO(opts.Currency)
	if err != nil {
		return p.Sprint(number.Decimal(n, numberOpts(opts)...))
	}

	formatter := currency.Symbol
	switch opts.CurrencyDisplay {
	case "code":
		formatter = currency.ISO
	case "narrowSymbol":
		formatter = currency.NarrowSymbol
	}

	return p.Sprint(formatter(unit.Amount(n)))
}
