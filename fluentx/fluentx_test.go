package fluentx

import (
	"strings"
	"testing"
	"time"

	"github.com/hakastein/gofluent"
)

func intp(i int) *int    { return &i }
func boolp(b bool) *bool { return &b }

func TestPluralCardinal(t *testing.T) {
	pr := NewPluralRules()
	cases := []struct {
		locale string
		n      float64
		want   string
	}{
		// English: only one/other.
		{"en", 1, "one"},
		{"en", 2, "other"},
		{"en", 0, "other"},
		// Russian.
		{"ru", 1, "one"},
		{"ru", 2, "few"},
		{"ru", 5, "many"},
		{"ru", 11, "many"},
		{"ru", 21, "one"},
		// Polish.
		{"pl", 1, "one"},
		{"pl", 2, "few"},
		{"pl", 5, "many"},
		// German: one/other.
		{"de", 1, "one"},
		{"de", 2, "other"},
		// French: 0 and 1 are "one".
		{"fr", 0, "one"},
		{"fr", 1, "one"},
		{"fr", 2, "other"},
		// Arabic.
		{"ar", 0, "zero"},
		{"ar", 1, "one"},
		{"ar", 2, "two"},
		{"ar", 3, "few"},
		{"ar", 11, "many"},
		{"ar", 100, "other"},
	}
	for _, c := range cases {
		got := pr.Cardinal(c.locale, c.n, fluent.NumberOptions{})
		if got != c.want {
			t.Errorf("Cardinal(%q, %v) = %q, want %q", c.locale, c.n, got, c.want)
		}
	}
}

func TestPluralOrdinal(t *testing.T) {
	pr := NewPluralRules()
	cases := []struct {
		n    float64
		want string
	}{
		{1, "one"},
		{2, "two"},
		{3, "few"},
		{4, "other"},
		{11, "other"},
	}
	for _, c := range cases {
		got := pr.Ordinal("en", c.n, fluent.NumberOptions{})
		if got != c.want {
			t.Errorf("Ordinal(en, %v) = %q, want %q", c.n, got, c.want)
		}
	}
}

// TestPluralFractionDigits shows that fraction-digit options influence the
// plural operands: with minimumFractionDigits the number is treated as having
// visible decimals (English "1.0" is category "other", not "one").
func TestPluralFractionDigits(t *testing.T) {
	pr := NewPluralRules()
	if got := pr.Cardinal("en", 1, fluent.NumberOptions{MinimumFractionDigits: intp(1)}); got != "other" {
		t.Errorf("Cardinal(en, 1, minFrac=1) = %q, want other", got)
	}
	if got := pr.Cardinal("en", 1, fluent.NumberOptions{}); got != "one" {
		t.Errorf("Cardinal(en, 1) = %q, want one", got)
	}
	// A value with its own fraction digits is treated as decimal even without
	// options (English "1.5" is "other").
	if got := pr.Cardinal("en", 1.5, fluent.NumberOptions{}); got != "other" {
		t.Errorf("Cardinal(en, 1.5) = %q, want other", got)
	}
	// maximumFractionDigits can round a decimal back to an integer category
	// (English "1.0" rounded to 0 fraction digits selects "one").
	if got := pr.Cardinal("en", 1.0, fluent.NumberOptions{MaximumFractionDigits: intp(0)}); got != "one" {
		t.Errorf("Cardinal(en, 1.0, maxFrac=0) = %q, want one", got)
	}
}

func TestFormatNumber(t *testing.T) {
	nf := NewNumberFormatter()
	// The CLDR group separators emitted by cldr/number (matching Intl): French
	// uses a narrow no-break space (U+202F); Russian uses a no-break space
	// (U+00A0). Currency amounts that place the symbol after the digits insert a
	// no-break space (U+00A0) between them, exactly as Intl does.
	const (
		narrowNBSP = " "
		nbsp       = " "
	)
	cases := []struct {
		name   string
		locale string
		n      float64
		opts   fluent.NumberOptions
		want   string
	}{
		{"en grouping", "en", 1234.5, fluent.NumberOptions{}, "1,234.5"},
		{"de grouping", "de", 1234.5, fluent.NumberOptions{}, "1.234,5"},
		{"fr grouping", "fr", 1234.5, fluent.NumberOptions{}, "1" + narrowNBSP + "234,5"},
		{"ru grouping", "ru", 1234.5, fluent.NumberOptions{}, "1" + nbsp + "234,5"},
		{"no grouping", "en", 1234.5, fluent.NumberOptions{UseGrouping: boolp(false)}, "1234.5"},
		{"min fraction", "en", 1234, fluent.NumberOptions{MinimumFractionDigits: intp(2)}, "1,234.00"},
		{"max fraction", "en", 1.239, fluent.NumberOptions{MaximumFractionDigits: intp(1)}, "1.2"},
		// Intl's default maximumFractionDigits for percent is 0, so 0.125 -> 12.5%
		// rounds to 13%.
		{"percent", "en", 0.125, fluent.NumberOptions{Style: "percent"}, "13%"},
		{"percent maxfrac", "en", 0.125, fluent.NumberOptions{Style: "percent", MaximumFractionDigits: intp(1)}, "12.5%"},
		// Currency: symbol hugs the amount with no space (Intl), unlike the old
		// x/text "$ 1,234.00".
		{"currency usd", "en", 1234, fluent.NumberOptions{Style: "currency", Currency: "USD"}, "$1,234.00"},
		// German EUR: symbol follows the amount, separated by a no-break space.
		{"currency eur de", "de", 1234.5, fluent.NumberOptions{Style: "currency", Currency: "EUR"}, "1.234,50" + nbsp + "€"},
		// Russian RUB: ruble sign follows the amount after a no-break space.
		{"currency rub ru", "ru", 1234, fluent.NumberOptions{Style: "currency", Currency: "RUB"}, "1" + nbsp + "234,00" + nbsp + "₽"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := nf.FormatNumber(c.locale, c.n, c.opts)
			if got != c.want {
				t.Errorf("FormatNumber(%q, %v, %+v) = %q, want %q", c.locale, c.n, c.opts, got, c.want)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	dtf := NewDateTimeFormatter()
	ts := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)

	cases := []struct {
		name   string
		locale string
		opts   fluent.DateTimeOptions
		want   string
	}{
		// Now backed by real CLDR data: month/weekday names are localized and the
		// dateStyle+timeStyle combiner uses the locale's "at" connector (Intl).
		{"en full date / short time", "en", fluent.DateTimeOptions{DateStyle: "full", TimeStyle: "short"}, "Thursday, January 5, 2023 at 2:09 PM"},
		{"en long date / medium time 24h", "en", fluent.DateTimeOptions{DateStyle: "long", TimeStyle: "medium", Hour12: boolp(false)}, "January 5, 2023 at 14:09:07"},
		{"en short date only", "en", fluent.DateTimeOptions{DateStyle: "short"}, "1/5/23"},
		{"en components y/m/d", "en", fluent.DateTimeOptions{Year: "numeric", Month: "long", Day: "2-digit"}, "January 05, 2023"},
		{"en components h:m 24h", "en", fluent.DateTimeOptions{Hour: "2-digit", Minute: "2-digit", Hour12: boolp(false)}, "14:09"},
		{"en components weekday short", "en", fluent.DateTimeOptions{Weekday: "short"}, "Thu"},
		// Localized long dates now work across locales (Intl).
		{"de long date", "de", fluent.DateTimeOptions{DateStyle: "long"}, "5. Januar 2023"},
		{"fr long date", "fr", fluent.DateTimeOptions{DateStyle: "long"}, "5 janvier 2023"},
		{"de full date", "de", fluent.DateTimeOptions{DateStyle: "full"}, "Donnerstag, 5. Januar 2023"},
		{"ru long date", "ru", fluent.DateTimeOptions{DateStyle: "long"}, "5 января 2023 г."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dtf.FormatDateTime(c.locale, ts, c.opts)
			if got != c.want {
				t.Errorf("FormatDateTime(%q, %+v) = %q, want %q", c.locale, c.opts, got, c.want)
			}
		})
	}
}

// TestDateTimeTimeZone verifies the timeZone option is applied via
// time.LoadLocation: 14:09 UTC becomes 09:09 in America/New_York (EST, UTC-5).
func TestDateTimeTimeZone(t *testing.T) {
	dtf := NewDateTimeFormatter()
	ts := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	got := dtf.FormatDateTime("en", ts, fluent.DateTimeOptions{
		Hour: "2-digit", Minute: "2-digit", Hour12: boolp(false), TimeZone: "America/New_York",
	})
	if got != "09:09" {
		t.Errorf("timeZone conversion = %q, want 09:09", got)
	}
}

// TestIntegration wires the fluentx formatters into a real fluent.Bundle and
// asserts locale-correct output for both a plural select expression and a
// NUMBER() placeable.
func TestIntegration(t *testing.T) {
	const ftl = `items = { $n ->
    [one] { $n } item
    [few] { $n } items (few)
    [many] { $n } items (many)
   *[other] { $n } items
}
total = Total: { NUMBER($n, useGrouping: 1) }
`

	t.Run("ru plural few for 2", func(t *testing.T) {
		b := fluent.NewBundle("ru", Options()...)
		res, errs := fluent.NewResource(ftl)
		if len(errs) > 0 {
			t.Fatalf("parse errors: %v", errs)
		}
		if errs := b.AddResource(res); len(errs) > 0 {
			t.Fatalf("addResource errors: %v", errs)
		}
		msg, ok := b.GetMessage("items")
		if !ok {
			t.Fatal("message items missing")
		}
		out := b.FormatPatternAny(msg.Value, map[string]any{"n": 2}, nil)
		if !strings.Contains(out, "(few)") {
			t.Errorf("ru n=2 => %q, want the [few] variant", out)
		}
	})

	t.Run("en plural one for 1", func(t *testing.T) {
		b := fluent.NewBundle("en", fluent.WithUseIsolating(false))
		for _, opt := range Options() {
			opt(b)
		}
		res, _ := fluent.NewResource(ftl)
		b.AddResource(res)
		msg, _ := b.GetMessage("items")
		out := b.FormatPatternAny(msg.Value, map[string]any{"n": 1}, nil)
		if out != "1 item" {
			t.Errorf("en n=1 => %q, want \"1 item\" (the [one] variant)", out)
		}
	})

	t.Run("en number grouping in NUMBER placeable", func(t *testing.T) {
		b := fluent.NewBundle("en", fluent.WithUseIsolating(false),
			fluent.WithNumberFormatter(NewNumberFormatter()),
			fluent.WithPluralRules(NewPluralRules()),
		)
		res, _ := fluent.NewResource(ftl)
		b.AddResource(res)
		msg, _ := b.GetMessage("total")
		out := b.FormatPatternAny(msg.Value, map[string]any{"n": 1234567}, nil)
		if out != "Total: 1,234,567" {
			t.Errorf("en NUMBER grouping => %q, want \"Total: 1,234,567\"", out)
		}
	})
}
