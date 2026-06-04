package fluent_test

import (
	"testing"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ported from select_expressions_test.js and literals_test.js.

func TestSelectMissingSelector(t *testing.T) {
	src := "select = {$none ->\n    [a] A\n   *[b] B\n}\n"
	b := newTestBundle(t, src)
	got, errs := format(t, b, "select", nil)
	assert.Equal(t, "B", got)
	require.Len(t, errs, 1, "a missing selector reports a single reference error")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
}

func TestStringSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [a] A\n   *[b] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": "a"})
	assert.Equal(t, "A", got, "matching")
	assert.Empty(t, errs)

	got, errs = format(t, b, "select", map[string]any{"selector": "c"})
	assert.Equal(t, "B", got, "non-matching falls back to default")
	assert.Empty(t, errs)
}

func TestNumberSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [0] A\n   *[1] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": 0})
	assert.Equal(t, "A", got, "matching")
	assert.Empty(t, errs)

	got, errs = format(t, b, "select", map[string]any{"selector": 2})
	assert.Equal(t, "B", got, "non-matching falls back to default")
	assert.Empty(t, errs)
}

func TestPluralCategoryStringMatch(t *testing.T) {
	// A string selector matching a CLDR-category-named variant works without a
	// real plural ruleset: the keys are compared as strings.
	src := "select = {$selector ->\n    [one] A\n   *[other] B\n}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "select", map[string]any{"selector": "one"})
	assert.Equal(t, "A", got, "string one")
	assert.Empty(t, errs)

	got, errs = format(t, b, "select", map[string]any{"selector": "other"})
	assert.Equal(t, "B", got, "string other")
	assert.Empty(t, errs)
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
			assert.Equal(t, tc.want, got)
			assert.Empty(t, errs)
		})
	}
}

// TestNumberLiteralPluralCategoryDefault documents an adaptation from fluent.js.
// In fluent.js, `{ 1 -> [one] A *[other] B }` selects "A" because Intl plural
// rules categorize 1 as "one" in en-US. The dependency-free default PluralRules
// returns "other" for everything, so the default variant "B" is selected here.
// A real CLDR PluralRules (wired via WithPluralRules) would select "A".
func TestNumberLiteralPluralCategoryDefault(t *testing.T) {
	src := "foo = { 1 ->\n    [one] A\n   *[other] B\n}\n"
	b := newTestBundle(t, src)
	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "B", got, "default plural rules select the other variant")
	assert.Empty(t, errs)

	// With a stub CLDR-like ruleset, the "one" category matches.
	b2 := fluent.NewBundle("en-US", fluent.WithUseIsolating(false), fluent.WithPluralRules(enPluralRules{}))
	b2.AddResource(mustParse(t, src))
	got2, _ := format(t, b2, "foo", nil)
	assert.Equal(t, "A", got2, "with plural rules the one category matches")
}

// enPluralRules is a minimal English cardinal ruleset used to exercise the
// pluggable PluralRules hook.
type enPluralRules struct{}

func (enPluralRules) Cardinal(_ string, n float64, _ fluent.NumberOptions) string {
	if n == 1 {
		return "one"
	}
	return "other"
}
func (enPluralRules) Ordinal(_ string, _ float64, _ fluent.NumberOptions) string { return "other" }
