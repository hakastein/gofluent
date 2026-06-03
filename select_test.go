package fluent

import "testing"

// Ported from select_expressions_test.js and literals_test.js.

func TestSelectMissingSelector(t *testing.T) {
	src := "select = {$none ->\n    [a] A\n   *[b] B\n}\n"
	b := newTestBundle(t, src)
	got, errs := format(t, b, "select", nil)
	if got != "B" {
		t.Errorf("got %q want B", got)
	}
	if len(errs) != 1 || !isReferenceError(errs[0]) {
		t.Errorf("expected 1 reference error, got %v", errs)
	}
}

func TestStringSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [a] A\n   *[b] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": "a"})
	if got != "A" || len(errs) != 0 {
		t.Errorf("matching: got %q errs %v", got, errs)
	}
	got, errs = format(t, b, "select", map[string]any{"selector": "c"})
	if got != "B" || len(errs) != 0 {
		t.Errorf("non-matching: got %q errs %v", got, errs)
	}
}

func TestNumberSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [0] A\n   *[1] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": 0})
	if got != "A" || len(errs) != 0 {
		t.Errorf("matching: got %q errs %v", got, errs)
	}
	got, errs = format(t, b, "select", map[string]any{"selector": 2})
	if got != "B" || len(errs) != 0 {
		t.Errorf("non-matching: got %q errs %v", got, errs)
	}
}

func TestPluralCategoryStringMatch(t *testing.T) {
	// A string selector matching a CLDR-category-named variant works without a
	// real plural ruleset: the keys are compared as strings.
	src := "select = {$selector ->\n    [one] A\n   *[other] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": "one"})
	if got != "A" || len(errs) != 0 {
		t.Errorf("string one: got %q errs %v", got, errs)
	}
	got, errs = format(t, b, "select", map[string]any{"selector": "other"})
	if got != "B" || len(errs) != 0 {
		t.Errorf("string other: got %q errs %v", got, errs)
	}
}

func TestLiteralSelectors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"matching string", "foo = { \"a\" ->\n    [a] A\n   *[b] B\n}\n", "A"},
		{"non-matching string", "foo = { \"c\" ->\n    [a] A\n   *[b] B\n}\n", "B"},
		{"matching number", "foo = { 0 ->\n    [0] A\n   *[1] B\n}\n", "A"},
		{"non-matching number", "foo = { 2 ->\n    [0] A\n   *[1] B\n}\n", "B"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := newTestBundle(t, tc.src)
			got, errs := format(t, b, "foo", nil)
			if got != tc.want || len(errs) != 0 {
				t.Errorf("got %q want %q errs %v", got, tc.want, errs)
			}
		})
	}
}

// TestNumberLiteralPluralCategory documents an adaptation from fluent.js.
// In fluent.js, `{ 1 -> [one] A *[other] B }` selects "A" because Intl plural
// rules categorize 1 as "one" in en-US. The dependency-free default PluralRules
// returns "other" for everything, so the default variant "B" is selected here.
// A real CLDR PluralRules (wired via WithPluralRules) would select "A".
func TestNumberLiteralPluralCategoryDefault(t *testing.T) {
	src := "foo = { 1 ->\n    [one] A\n   *[other] B\n}\n"
	b := newTestBundle(t, src)
	got, errs := format(t, b, "foo", nil)
	if got != "B" || len(errs) != 0 {
		t.Errorf("default plural rules: got %q want B errs %v", got, errs)
	}

	// With a stub CLDR-like ruleset, the "one" category matches.
	b2 := NewBundle("en-US", WithUseIsolating(false), WithPluralRules(enPluralRules{}))
	b2.AddResource(mustParse(t, src))
	got2, _ := format(t, b2, "foo", nil)
	if got2 != "A" {
		t.Errorf("with plural rules: got %q want A", got2)
	}
}

// enPluralRules is a minimal English cardinal ruleset used to exercise the
// pluggable PluralRules hook.
type enPluralRules struct{}

func (enPluralRules) Cardinal(_ string, n float64, _ NumberOptions) string {
	if n == 1 {
		return "one"
	}
	return "other"
}
func (enPluralRules) Ordinal(_ string, _ float64, _ NumberOptions) string { return "other" }
