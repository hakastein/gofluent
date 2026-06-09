package fluent

import (
	"time"

	"github.com/hakastein/gocldr/datetime"
	"github.com/hakastein/gocldr/number"
)

// This file wires the default, CLDR-backed implementations of the pluggable
// formatting interfaces (PluralRules, NumberFormatter, DateTimeFormatter) onto
// the external github.com/hakastein/gocldr module, whose output is validated
// against Node's Intl.* (and therefore matches fluent.js). NewBundle installs
// these by default; WithPluralRules/WithNumberFormatter/WithDateTimeFormatter
// override them.
//
// Locale data in gocldr is opt-in: a program links only the locales it
// (blank-)imports. Applications MUST import the locale data they format, e.g.:
//
//	import _ "github.com/hakastein/gocldr/locales/en"  // a single locale
//	import _ "github.com/hakastein/gocldr/locales/all" // every locale
//
// With no locale data imported, formatting degrades gracefully: numbers fall
// back to the CLDR-root pattern (ASCII grouped decimals) and dates to RFC 3339.

// cldrNumberFormatter is the default NumberFormatter, backed by gocldr/number.
// It renders decimals, percents and currency amounts with locale-aware
// grouping, decimal separators and currency symbols, matching Intl.NumberFormat.
type cldrNumberFormatter struct{}

func (cldrNumberFormatter) FormatNumber(locale string, n float64, opts NumberOptions) string {
	return number.Format(locale, n, numberOptions(opts))
}

// numberOptions maps NumberOptions onto gocldr/number.Options. The structs share
// field names; this is a straight field-by-field copy of the fields
// gocldr/number understands. (Unit/UnitDisplay/Type have no gocldr/number
// counterpart and are not number-rendering concerns here.)
func numberOptions(opts NumberOptions) number.Options {
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

// datetimeOptions maps DateTimeOptions onto gocldr/datetime.Options. The two
// structs mirror each other field-for-field.
func datetimeOptions(opts DateTimeOptions) datetime.Options {
	return datetime.Options{
		Hour12:                 opts.Hour12,
		Weekday:                opts.Weekday,
		Era:                    opts.Era,
		Year:                   opts.Year,
		Month:                  opts.Month,
		Day:                    opts.Day,
		Hour:                   opts.Hour,
		Minute:                 opts.Minute,
		Second:                 opts.Second,
		TimeZoneName:           opts.TimeZoneName,
		DateStyle:              opts.DateStyle,
		TimeStyle:              opts.TimeStyle,
		DayPeriod:              opts.DayPeriod,
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

// Compile-time checks that the CLDR defaults satisfy the core interfaces.
var (
	_ NumberFormatter   = cldrNumberFormatter{}
	_ DateTimeFormatter = cldrDateTimeFormatter{}
	_ PluralRules       = cldrPluralRules{}
)
