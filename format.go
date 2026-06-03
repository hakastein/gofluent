package fluent

import (
	"strconv"
	"strings"
	"time"
)

// This file defines the pluggable formatting layer. It is the ONE deliberate
// architectural deviation from fluent.js: instead of depending on the JS
// `Intl.*` objects (Intl.NumberFormat, Intl.DateTimeFormat, Intl.PluralRules),
// the Go port exposes interfaces that a caller can wire to a real CLDR
// implementation (e.g. a separate `fluentx` package built on x/text). The core
// stays dependency-free.
//
// The locale is represented as a plain BCP-47 string and passed to every
// formatter; no locale library is imported here.

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

// defaultNumberFormatter is the no-op number formatter used when the caller
// wires nothing. It renders a plain decimal honoring min/max fraction digits
// via strconv, without locale-aware grouping or currency symbols.
type defaultNumberFormatter struct{}

func (defaultNumberFormatter) FormatNumber(_ string, n float64, opts NumberOptions) string {
	// Determine fraction digits. If a minimum is given, pad to at least that
	// many; if a maximum is given, round to at most that many.
	minFrac := 0
	if opts.MinimumFractionDigits != nil {
		minFrac = *opts.MinimumFractionDigits
	}
	maxFrac := -1
	if opts.MaximumFractionDigits != nil {
		maxFrac = *opts.MaximumFractionDigits
	}
	if maxFrac >= 0 && maxFrac < minFrac {
		maxFrac = minFrac
	}

	var s string
	if maxFrac >= 0 {
		// Round to maxFrac digits, then trim trailing zeros down to minFrac.
		s = strconv.FormatFloat(n, 'f', maxFrac, 64)
		s = trimToFractionRange(s, minFrac, maxFrac)
	} else if minFrac > 0 {
		s = strconv.FormatFloat(n, 'f', minFrac, 64)
	} else {
		// No constraints: use the shortest representation.
		s = strconv.FormatFloat(n, 'f', -1, 64)
	}
	return s
}

// trimToFractionDigitsRange ensures the fractional part has between minFrac and
// maxFrac digits: it trims trailing zeros but never below minFrac digits.
func trimToFractionRange(s string, minFrac, maxFrac int) string {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		if minFrac > 0 {
			return s + "." + strings.Repeat("0", minFrac)
		}
		return s
	}
	frac := s[dot+1:]
	// Trim trailing zeros but keep at least minFrac digits.
	for len(frac) > minFrac && frac[len(frac)-1] == '0' {
		frac = frac[:len(frac)-1]
	}
	if len(frac) == 0 {
		return s[:dot]
	}
	return s[:dot+1] + frac
}

// defaultPluralRules is the no-op plural ruleset used when the caller wires
// nothing. It only matches exact numeric values handled by the resolver; for
// CLDR category matching it always returns "other".
type defaultPluralRules struct{}

func (defaultPluralRules) Cardinal(_ string, _ float64, _ NumberOptions) string {
	return "other"
}

func (defaultPluralRules) Ordinal(_ string, _ float64, _ NumberOptions) string {
	return "other"
}

// defaultDateTimeFormatter is the no-op datetime formatter used when the caller
// wires nothing. It renders the time in RFC 3339 / ISO-8601 UTC, mirroring the
// fallback behavior of fluent.js (Date.toISOString) when no Intl object can be
// constructed.
type defaultDateTimeFormatter struct{}

func (defaultDateTimeFormatter) FormatDateTime(_ string, t time.Time, _ DateTimeOptions) string {
	// Match JS Date.toISOString(): milliseconds precision, "Z" suffix.
	ms := t.UTC().Truncate(time.Millisecond)
	return ms.Format("2006-01-02T15:04:05.000Z07:00")
}

// boolPtr / intPtr are small helpers for constructing optional fields.
func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
