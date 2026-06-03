package number

import (
	"math"
	"testing"
)

func ptrInt(i int) *int    { return &i }
func ptrBool(b bool) *bool { return &b }

// TestSpecialValues covers non-finite inputs, which Intl renders via locale
// symbols.
func TestSpecialValues(t *testing.T) {
	cases := []struct {
		locale string
		value  float64
		want   string
	}{
		{"en", math.NaN(), "NaN"},
		{"en", math.Inf(1), "∞"},
		{"en", math.Inf(-1), "-∞"},
		{"de", math.NaN(), "NaN"},
	}
	for _, c := range cases {
		if got := Format(c.locale, c.value, Options{}); got != c.want {
			t.Errorf("Format(%q, %v) = %q, want %q", c.locale, c.value, got, c.want)
		}
	}
}

// TestLocaleFallback verifies the exact -> truncate -> root resolution chain.
func TestLocaleFallback(t *testing.T) {
	cases := []struct {
		locale string
		want   string
	}{
		{"en-XX", "1,234.5"},      // unknown region falls back to en
		{"zh-Hans-CN", "1,234.5"}, // multi-subtag resolves like zh
		{"EN", "1,234.5"},         // case-insensitive language subtag
		{"de_DE", "1.234,5"},      // underscore separator, region present
	}
	for _, c := range cases {
		if got := Format(c.locale, 1234.5, Options{}); got != c.want {
			t.Errorf("Format(%q, 1234.5) = %q, want %q", c.locale, got, c.want)
		}
	}
}

// TestAccountingUnused confirms the standard (non-accounting) currency pattern
// is used by default and that negative values use the locale minus sign.
func TestCurrencyBasics(t *testing.T) {
	cases := []struct {
		locale string
		value  float64
		opts   Options
		want   string
	}{
		{"en", 1234.5, Options{Style: "currency", Currency: "usd"}, "$1,234.50"}, // lowercase code
		{"en", -5, Options{Style: "currency", Currency: "USD"}, "-$5.00"},
		{"en", 1234.5, Options{Style: "currency", Currency: "JPY"}, "¥1,235"},
		{"en", 1234.5, Options{Style: "currency", Currency: "XYZ"}, "XYZ 1,234.50"}, // unknown currency -> code
	}
	for _, c := range cases {
		if got := Format(c.locale, c.value, c.opts); got != c.want {
			t.Errorf("Format(%q, %v, %+v) = %q, want %q", c.locale, c.value, c.opts, got, c.want)
		}
	}
}

// TestOptionsRoundingModes exercises a few representative option combinations.
func TestOptions(t *testing.T) {
	cases := []struct {
		locale string
		value  float64
		opts   Options
		want   string
	}{
		{"en", 2.5, Options{MaximumFractionDigits: ptrInt(0)}, "3"},   // half away from zero
		{"en", -2.5, Options{MaximumFractionDigits: ptrInt(0)}, "-3"}, // away from zero on negatives too
		{"en", 1234567, Options{UseGrouping: ptrBool(false)}, "1234567"},
		{"en", 1, Options{MinimumIntegerDigits: ptrInt(3)}, "001"},
		{"en", 12345.678, Options{MaximumSignificantDigits: ptrInt(3)}, "12,300"},
	}
	for _, c := range cases {
		if got := Format(c.locale, c.value, c.opts); got != c.want {
			t.Errorf("Format(%q, %v, %+v) = %q, want %q", c.locale, c.value, c.opts, got, c.want)
		}
	}
}
