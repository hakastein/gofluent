package fluent

import "time"

// NumberOptions carries the options that the NUMBER() builtin and FluentNumber
// accept. It mirrors the subset of Intl.NumberFormatOptions used by fluent.js.
// Pointer fields distinguish "unset" from a zero value, mirroring how fluent.js
// merges option bags.
type NumberOptions struct {
	Style                    string // "decimal" | "currency" | "percent" | "unit"
	Currency                 string
	CurrencyDisplay          string
	Unit                     string
	UnitDisplay              string
	UseGrouping              *bool
	MinimumIntegerDigits     *int
	MinimumFractionDigits    *int
	MaximumFractionDigits    *int
	MinimumSignificantDigits *int
	MaximumSignificantDigits *int

	// Type selects the plural ruleset: "cardinal" (default) or "ordinal".
	Type string
}

// DateTimeOptions carries the options that the DATETIME() builtin and
// FluentDateTime accept. It mirrors the subset of Intl.DateTimeFormatOptions
// used by fluent.js. Pointer fields distinguish "unset" from a zero value.
type DateTimeOptions struct {
	Hour12                 *bool
	Weekday                string
	Era                    string
	Year                   string
	Month                  string
	Day                    string
	Hour                   string
	Minute                 string
	Second                 string
	TimeZoneName           string
	DateStyle              string
	TimeStyle              string
	DayPeriod              string
	FractionalSecondDigits *int
	Calendar               string
	NumberingSystem        string
	TimeZone               string
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

// boolPtr / intPtr are small helpers for constructing optional fields.
func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
