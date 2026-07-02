package fluent

import "time"

// NumberStyle selects what kind of quantity a number is. It is the type of
// NumberOptions.Style.
type NumberStyle string

const (
	StyleDecimal  NumberStyle = "decimal"
	StyleCurrency NumberStyle = "currency"
	StylePercent  NumberStyle = "percent"
	StyleUnit     NumberStyle = "unit"
)

// CurrencyDisplay selects how a currency is shown. It is the type of
// NumberOptions.CurrencyDisplay.
type CurrencyDisplay string

const (
	CurrencyDisplaySymbol       CurrencyDisplay = "symbol"
	CurrencyDisplayNarrowSymbol CurrencyDisplay = "narrowSymbol"
	CurrencyDisplayCode         CurrencyDisplay = "code"
	CurrencyDisplayName         CurrencyDisplay = "name"
)

// UnitDisplay selects how a unit is shown. It is the type of
// NumberOptions.UnitDisplay.
type UnitDisplay string

const (
	UnitDisplayLong   UnitDisplay = "long"
	UnitDisplayShort  UnitDisplay = "short"
	UnitDisplayNarrow UnitDisplay = "narrow"
)

// PluralType selects the plural ruleset a number is categorized against. It is
// the type of NumberOptions.Type; the zero value ("") behaves as Cardinal.
type PluralType string

const (
	Cardinal PluralType = "cardinal"
	Ordinal  PluralType = "ordinal"
)

// NumberOptions carries the options that the NUMBER() builtin and FluentNumber
// accept. It mirrors the subset of Intl.NumberFormatOptions used by fluent.js.
// Pointer fields distinguish "unset" from a zero value, mirroring how fluent.js
// merges option bags.
type NumberOptions struct {
	Style                    NumberStyle
	Currency                 string // ISO 4217 code, e.g. "USD"
	CurrencyDisplay          CurrencyDisplay
	Unit                     string // sanctioned unit id, e.g. "kilometer"
	UnitDisplay              UnitDisplay
	UseGrouping              *bool
	MinimumIntegerDigits     *int
	MinimumFractionDigits    *int
	MaximumFractionDigits    *int
	MinimumSignificantDigits *int
	MaximumSignificantDigits *int

	Type PluralType
}

// DateTimeStyle is a date or time formatting length. It is the type of
// DateTimeOptions.DateStyle and DateTimeOptions.TimeStyle.
type DateTimeStyle string

const (
	DateTimeFull   DateTimeStyle = "full"
	DateTimeLong   DateTimeStyle = "long"
	DateTimeMedium DateTimeStyle = "medium"
	DateTimeShort  DateTimeStyle = "short"
)

// TextWidth is the rendered width of a named date/time field. It is the type of
// DateTimeOptions.Weekday, Era, and DayPeriod.
type TextWidth string

const (
	WidthLong   TextWidth = "long"
	WidthShort  TextWidth = "short"
	WidthNarrow TextWidth = "narrow"
)

// NumericStyle selects numeric vs. zero-padded rendering of a date/time field.
// It is the type of DateTimeOptions.Year, Day, Hour, Minute, and Second.
type NumericStyle string

const (
	Numeric  NumericStyle = "numeric"
	TwoDigit NumericStyle = "2-digit"
)

// MonthStyle is the rendering of the month field, which accepts both the
// numeric and the named widths. It is the type of DateTimeOptions.Month.
type MonthStyle string

const (
	MonthNumeric  MonthStyle = "numeric"
	MonthTwoDigit MonthStyle = "2-digit"
	MonthLong     MonthStyle = "long"
	MonthShort    MonthStyle = "short"
	MonthNarrow   MonthStyle = "narrow"
)

// TimeZoneNameStyle selects how the time-zone name is shown. It is the type of
// DateTimeOptions.TimeZoneName.
type TimeZoneNameStyle string

const (
	TimeZoneLong         TimeZoneNameStyle = "long"
	TimeZoneShort        TimeZoneNameStyle = "short"
	TimeZoneShortOffset  TimeZoneNameStyle = "shortOffset"
	TimeZoneLongOffset   TimeZoneNameStyle = "longOffset"
	TimeZoneShortGeneric TimeZoneNameStyle = "shortGeneric"
	TimeZoneLongGeneric  TimeZoneNameStyle = "longGeneric"
)

// DateTimeOptions carries the options that the DATETIME() builtin and
// FluentDateTime accept. It mirrors the subset of Intl.DateTimeFormatOptions
// used by fluent.js. Pointer fields distinguish "unset" from a zero value.
type DateTimeOptions struct {
	Hour12                 *bool
	Weekday                TextWidth
	Era                    TextWidth
	Year                   NumericStyle
	Month                  MonthStyle
	Day                    NumericStyle
	Hour                   NumericStyle
	Minute                 NumericStyle
	Second                 NumericStyle
	TimeZoneName           TimeZoneNameStyle
	DateStyle              DateTimeStyle
	TimeStyle              DateTimeStyle
	DayPeriod              TextWidth
	FractionalSecondDigits *int
	Calendar               string // e.g. "gregory"; only Gregorian is rendered
	NumberingSystem        string // e.g. "arab"; overrides the locale default
	TimeZone               string // IANA name, e.g. "America/New_York"
}

// PluralRules returns CLDR plural categories for a number in a given locale.
// Implementations return one of: "zero", "one", "two", "few", "many", "other".
type PluralRules interface {
	// Cardinal returns the cardinal plural category for n.
	Cardinal(locale string, n float64, opts NumberOptions) string
	// Ordinal returns the ordinal plural category for n.
	Ordinal(locale string, n float64, opts NumberOptions) string
}

// NumberFormatter renders a number to a string for a given locale and options.
type NumberFormatter interface {
	FormatNumber(locale string, n float64, opts NumberOptions) string
}

// DateTimeFormatter renders a time to a string for a given locale and options.
type DateTimeFormatter interface {
	FormatDateTime(locale string, t time.Time, opts DateTimeOptions) string
}

// Bool returns a pointer to v, for the pointer-typed option fields that
// distinguish "unset" from a set value (NumberOptions.UseGrouping,
// DateTimeOptions.Hour12). Precedent: aws.Bool.
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v, for the pointer-typed option fields such as
// NumberOptions.MinimumFractionDigits. Precedent: aws.Int.
func Int(v int) *int { return &v }
