package fluentx

import (
	"strconv"
	"strings"

	"github.com/hakastein/gofluent"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

// PluralRules is a CLDR-backed implementation of fluent.PluralRules built on
// golang.org/x/text/feature/plural. It returns cardinal and ordinal plural
// categories for a number in a given BCP-47 locale.
type PluralRules struct{}

// NewPluralRules constructs a PluralRules.
func NewPluralRules() *PluralRules { return &PluralRules{} }

var _ fluent.PluralRules = (*PluralRules)(nil)

// Cardinal returns the cardinal plural category ("zero"/"one"/"two"/"few"/
// "many"/"other") for n in the given locale.
func (PluralRules) Cardinal(locale string, n float64, opts fluent.NumberOptions) string {
	return match(plural.Cardinal, locale, n, opts)
}

// Ordinal returns the ordinal plural category for n in the given locale.
func (PluralRules) Ordinal(locale string, n float64, opts fluent.NumberOptions) string {
	return match(plural.Ordinal, locale, n, opts)
}

func match(rules *plural.Rules, locale string, n float64, opts fluent.NumberOptions) string {
	tag, err := language.Parse(locale)
	if err != nil {
		tag = language.English
	}
	i, v, w, f, t := operands(n, opts)
	form := rules.MatchPlural(tag, i, v, w, f, t)
	return formString(form)
}

// operands derives the CLDR plural operands from a float64 and the relevant
// NumberOptions fraction-digit settings, mirroring how a formatted number would
// be rendered:
//
//	i  integer digits of n
//	v  number of visible fraction digits, with trailing zeros
//	w  number of visible fraction digits, without trailing zeros
//	f  visible fractional digits, with trailing zeros, as an integer
//	t  visible fractional digits, without trailing zeros, as an integer
//
// The fraction-digit count honors MinimumFractionDigits / MaximumFractionDigits
// so that, e.g., NUMBER($n, minimumFractionDigits: 1) selects the plural form
// for "1.0" rather than "1". Values too large for an int are reduced modulo
// 10,000,000 as permitted by MatchPlural.
func operands(n float64, opts fluent.NumberOptions) (i, v, w, f, t int) {
	if n < 0 {
		n = -n
	}

	minFrac := 0
	if opts.MinimumFractionDigits != nil && *opts.MinimumFractionDigits > 0 {
		minFrac = *opts.MinimumFractionDigits
	}
	maxFrac := -1
	if opts.MaximumFractionDigits != nil && *opts.MaximumFractionDigits >= 0 {
		maxFrac = *opts.MaximumFractionDigits
	}
	if maxFrac >= 0 && maxFrac < minFrac {
		maxFrac = minFrac
	}

	// Render n the way the number formatter would: round to maxFrac digits when
	// a maximum is set, otherwise use the shortest representation, then pad to
	// minFrac trailing zeros.
	var s string
	switch {
	case maxFrac >= 0:
		s = strconv.FormatFloat(n, 'f', maxFrac, 64)
		s = trimTrailingZeros(s, minFrac)
	case minFrac > 0:
		s = strconv.FormatFloat(n, 'f', minFrac, 64)
	default:
		s = strconv.FormatFloat(n, 'f', -1, 64)
	}

	intPart, fracPart := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, fracPart = s[:dot], s[dot+1:]
	}

	const mod = 10_000_000
	i = atoiMod(intPart, mod)

	withZeros := fracPart
	withoutZeros := strings.TrimRight(fracPart, "0")
	v = len(withZeros)
	w = len(withoutZeros)
	f = atoiMod(withZeros, mod)
	t = atoiMod(withoutZeros, mod)
	return i, v, w, f, t
}

// trimTrailingZeros trims trailing fractional zeros from a decimal string but
// keeps at least minFrac fraction digits.
func trimTrailingZeros(s string, minFrac int) string {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return s
	}
	frac := s[dot+1:]
	for len(frac) > minFrac && frac[len(frac)-1] == '0' {
		frac = frac[:len(frac)-1]
	}
	if len(frac) == 0 {
		return s[:dot]
	}
	return s[:dot+1] + frac
}

// atoiMod parses a (possibly empty) digit string into an int, reducing modulo m
// to stay within int range as MatchPlural permits.
func atoiMod(digits string, m int) int {
	if digits == "" {
		return 0
	}
	acc := 0
	for j := 0; j < len(digits); j++ {
		c := digits[j]
		if c < '0' || c > '9' {
			continue
		}
		acc = (acc*10 + int(c-'0')) % m
	}
	return acc
}

func formString(f plural.Form) string {
	switch f {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	default:
		return "other"
	}
}
