package number

import "github.com/hakastein/gofluent/cldr/plural"

// defaultCurrencyDigits is the CLDR DEFAULT fraction-digit count for currencies
// not listed in the currencyData fractions table.
const defaultCurrencyDigits = 2

// pluralCategoryForDigits returns the CLDR cardinal plural category for a number
// whose displayed integer and fraction digit strings are given. Intl derives
// plural operands from the digits actually shown (so "1.00" has v=2 and yields
// the "other" category in many locales, not "one"). Used for
// currencyDisplay:"name" pattern selection. It reuses the sibling,
// dependency-free plural package.
func pluralCategoryForDigits(locale, intDigits, fracDigits string) string {
	s := intDigits
	if fracDigits != "" {
		s += "." + fracDigits
	}
	ops, err := plural.OperandsFromString(s)
	if err != nil {
		return "other"
	}
	return string(plural.Cardinal(locale, ops))
}
