package fluent_test

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/hakastein/gocldr/locales/all" // CLDR locale data for the formatting assertions

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the CLDR-backed formatters that NewBundle installs by
// default (see format_cldr.go), through the public Bundle API only. They guard
// the option mapping onto github.com/hakastein/gocldr; the golden strings match
// Node's Intl.* (CLDR 46), as validated in the gocldr module itself.

func TestCLDRNumberFormatting(t *testing.T) {
	// CLDR group/symbol characters emitted by gocldr/number (matching Intl):
	// French uses a narrow no-break space (U+202F), Russian a no-break space
	// (U+00A0); currency symbols that follow the amount are joined with U+00A0.
	const (
		narrowNBSP = " "
		nbsp       = " "
	)
	// style and currency are outside fluent.js's FTL option allowlist, so the
	// percent/currency cases carry them on a Number argument built in code.
	percent := func(n float64) *fluent.Number {
		return fluent.NewNumber(n, fluent.NumberOptions{Style: fluent.StylePercent})
	}
	currency := func(n float64, code string) *fluent.Number {
		return fluent.NewNumber(n, fluent.NumberOptions{Style: fluent.StyleCurrency, Currency: code})
	}
	cases := []struct {
		name   string
		locale string
		opts   string // extra named options after $n
		n      any    // float64 or a *fluent.Number carrying options
		want   string
	}{
		{"en grouping", "en", "", 1234.5, "1,234.5"},
		{"de grouping", "de", "", 1234.5, "1.234,5"},
		{"fr grouping", "fr", "", 1234.5, "1" + narrowNBSP + "234,5"},
		{"ru grouping", "ru", "", 1234.5, "1" + nbsp + "234,5"},
		{"no grouping", "en", ", useGrouping: 0", 1234.5, "1234.5"},
		{"min fraction", "en", ", minimumFractionDigits: 2", 1234, "1,234.00"},
		{"max fraction", "en", ", maximumFractionDigits: 1", 1.239, "1.2"},
		// Intl's default maximumFractionDigits for percent is 0, so 12.5% rounds.
		{"percent", "en", "", percent(0.125), "13%"},
		{"percent maxfrac", "en", ", maximumFractionDigits: 1", percent(0.125), "12.5%"},
		{"currency usd", "en", "", currency(1234, "USD"), "$1,234.00"},
		{"currency eur de", "de", "", currency(1234.5, "EUR"), "1.234,50" + nbsp + "€"},
		{"currency rub ru", "ru", "", currency(1234, "RUB"), "1" + nbsp + "234,00" + nbsp + "₽"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newLocaleBundle(t, c.locale, "v = { NUMBER($n"+c.opts+") }\n")
			got, errs := format(t, b, "v", map[string]any{"n": c.n})
			require.Empty(t, errs)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestCLDRPluralCardinalSelect(t *testing.T) {
	const src = "v = { $n ->\n" +
		"        [zero] zero\n" +
		"        [one] one\n" +
		"        [two] two\n" +
		"        [few] few\n" +
		"        [many] many\n" +
		"       *[other] other\n" +
		"    }\n"
	cases := []struct {
		locale string
		n      float64
		want   string
	}{
		{"en", 1, "one"}, {"en", 2, "other"}, {"en", 0, "other"},
		{"ru", 1, "one"}, {"ru", 2, "few"}, {"ru", 5, "many"}, {"ru", 21, "one"},
		{"pl", 1, "one"}, {"pl", 2, "few"}, {"pl", 5, "many"},
		{"de", 1, "one"}, {"de", 2, "other"},
		{"fr", 0, "one"}, {"fr", 1, "one"}, {"fr", 2, "other"},
		{"ar", 0, "zero"}, {"ar", 1, "one"}, {"ar", 2, "two"},
		{"ar", 3, "few"}, {"ar", 11, "many"}, {"ar", 100, "other"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_%v", c.locale, c.n), func(t *testing.T) {
			b := newLocaleBundle(t, c.locale, src)
			got, errs := format(t, b, "v", map[string]any{"n": c.n})
			require.Empty(t, errs)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestCLDRPluralOrdinalSelect(t *testing.T) {
	const src = "v = { $n ->\n" +
		"        [one] one\n" +
		"        [two] two\n" +
		"        [few] few\n" +
		"       *[other] other\n" +
		"    }\n"
	// The NUMBER() builtin does not parse a "type" option, so ordinal selection
	// is driven by a Number carrying NumberOptions.Type == "ordinal"; the
	// resolver then consults PluralRules.Ordinal.
	cases := []struct {
		n    float64
		want string
	}{
		{1, "one"}, {2, "two"}, {3, "few"}, {4, "other"}, {11, "other"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("en_%v", c.n), func(t *testing.T) {
			b := newLocaleBundle(t, "en", src)
			arg := fluent.NewNumber(c.n, fluent.NumberOptions{Type: fluent.Ordinal})
			got, errs := format(t, b, "v", map[string]any{"n": arg})
			require.Empty(t, errs)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestCLDRDateTimeFormatting(t *testing.T) {
	ts := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	cases := []struct {
		name   string
		locale string
		opts   string
		want   string
	}{
		{"en full date / short time", "en", `dateStyle: "full", timeStyle: "short"`, "Thursday, January 5, 2023 at 2:09 PM"},
		{"en long date / medium time 24h", "en", `dateStyle: "long", timeStyle: "medium", hour12: 0`, "January 5, 2023 at 14:09:07"},
		{"en short date", "en", `dateStyle: "short"`, "1/5/23"},
		{"de long date", "de", `dateStyle: "long"`, "5. Januar 2023"},
		{"fr long date", "fr", `dateStyle: "long"`, "5 janvier 2023"},
		{"de full date", "de", `dateStyle: "full"`, "Donnerstag, 5. Januar 2023"},
		{"ru long date", "ru", `dateStyle: "long"`, "5 января 2023 г."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newLocaleBundle(t, c.locale, "v = { DATETIME($d, "+c.opts+") }\n")
			got, errs := format(t, b, "v", map[string]any{"d": ts})
			require.Empty(t, errs)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestCLDRDateTimeTimeZone verifies the TimeZone option is applied via
// time.LoadLocation: 14:09 UTC becomes 09:09 in America/New_York (EST, UTC-5).
// timeZone is outside fluent.js's FTL option allowlist, so it rides on the
// DateTime argument.
func TestCLDRDateTimeTimeZone(t *testing.T) {
	ts := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	b := newLocaleBundle(t, "en",
		`v = { DATETIME($d, hour: "2-digit", minute: "2-digit", hour12: 0) }`+"\n")
	arg := fluent.NewDateTime(ts, fluent.DateTimeOptions{TimeZone: "America/New_York"})
	got, errs := format(t, b, "v", map[string]any{"d": arg})
	require.Empty(t, errs)
	assert.Equal(t, "09:09", got)
}
