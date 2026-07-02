package fluent

import (
	"time"

	"github.com/hakastein/gocldr/datetime"
	"github.com/hakastein/gocldr/number"
)

// cldrNumberFormatter is the default NumberFormatter, backed by gocldr/number.
// It renders decimals, percents and currency amounts with locale-aware
// grouping, decimal separators and currency symbols, matching Intl.NumberFormat.
type cldrNumberFormatter struct{}

func (cldrNumberFormatter) FormatNumber(locale string, n float64, opts NumberOptions) string {
	return number.Format(locale, n, numberOptions(opts))
}

// numberOptions maps NumberOptions onto gocldr/number.Options.
// Unit/UnitDisplay/Type have no gocldr/number counterpart and are dropped.
func numberOptions(opts NumberOptions) number.Options {
	return number.Options{
		Style:                    string(opts.Style),
		Currency:                 opts.Currency,
		CurrencyDisplay:          string(opts.CurrencyDisplay),
		UseGrouping:              opts.UseGrouping,
		MinimumIntegerDigits:     opts.MinimumIntegerDigits,
		MinimumFractionDigits:    opts.MinimumFractionDigits,
		MaximumFractionDigits:    opts.MaximumFractionDigits,
		MinimumSignificantDigits: opts.MinimumSignificantDigits,
		MaximumSignificantDigits: opts.MaximumSignificantDigits,
	}
}

// cldrDateTimeFormatter is the default DateTimeFormatter, backed by
// gocldr/datetime (standard-library time only). It renders fully localized
// dates and times from the Unicode CLDR data — month and weekday names, era and
// day-period labels, locale field ordering and numbering systems — matching
// Intl.DateTimeFormat for the dateStyle/timeStyle styles and the common
// component options. The timeZone option (an IANA name, e.g. "America/New_York")
// is applied via time.LoadLocation by gocldr/datetime; an unknown zone leaves
// the time unchanged.
type cldrDateTimeFormatter struct{}

func (cldrDateTimeFormatter) FormatDateTime(locale string, t time.Time, opts DateTimeOptions) string {
	return datetime.Format(locale, t, datetimeOptions(opts))
}

// datetimeOptions maps DateTimeOptions onto gocldr/datetime.Options.
func datetimeOptions(opts DateTimeOptions) datetime.Options {
	return datetime.Options{
		Hour12:                 opts.Hour12,
		Weekday:                string(opts.Weekday),
		Era:                    string(opts.Era),
		Year:                   string(opts.Year),
		Month:                  string(opts.Month),
		Day:                    string(opts.Day),
		Hour:                   string(opts.Hour),
		Minute:                 string(opts.Minute),
		Second:                 string(opts.Second),
		TimeZoneName:           string(opts.TimeZoneName),
		DateStyle:              string(opts.DateStyle),
		TimeStyle:              string(opts.TimeStyle),
		DayPeriod:              string(opts.DayPeriod),
		FractionalSecondDigits: opts.FractionalSecondDigits,
		Calendar:               opts.Calendar,
		NumberingSystem:        opts.NumberingSystem,
		TimeZone:               opts.TimeZone,
	}
}

// cldrPluralRules is the default PluralRules, backed by gocldr/number. It returns
// cardinal and ordinal plural categories for a number in a given BCP-47 locale,
// matching Intl.PluralRules.
type cldrPluralRules struct{}

func (cldrPluralRules) Cardinal(locale string, n float64, opts NumberOptions) string {
	return number.CardinalCategory(locale, n, pluralOptions(opts))
}

func (cldrPluralRules) Ordinal(locale string, n float64, opts NumberOptions) string {
	return number.OrdinalCategory(locale, n, pluralOptions(opts))
}

// pluralOptions maps the NumberOptions digit constraints onto the
// gocldr/number.Options understood by plural selection. Selection must see
// exactly the digits the number formatter would display, so the
// integer/fraction/significant-digit options are forwarded verbatim and
// gocldr/number applies the same resolution and rounding. Style/Currency are
// intentionally dropped: Intl.PluralRules has no style (it never scales percents
// or applies currency fraction defaults).
func pluralOptions(opts NumberOptions) number.Options {
	return number.Options{
		MinimumIntegerDigits:     opts.MinimumIntegerDigits,
		MinimumFractionDigits:    opts.MinimumFractionDigits,
		MaximumFractionDigits:    opts.MaximumFractionDigits,
		MinimumSignificantDigits: opts.MinimumSignificantDigits,
		MaximumSignificantDigits: opts.MaximumSignificantDigits,
	}
}
