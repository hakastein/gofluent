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
	got, err := format(t, b, "select", nil)
	assert.Equal(t, "B", got)
	require.Len(t, errList(err), 1, "a missing selector reports a single reference error")
	require.ErrorIs(t, err, fluent.ErrReference)
}

func TestStringSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [a] A\n   *[b] B\n}\n"
	b := newTestBundle(t, src)

	got, err := format(t, b, "select", map[string]any{"selector": "a"})
	assert.Equal(t, "A", got, "matching")
	assert.NoError(t, err)

	got, err = format(t, b, "select", map[string]any{"selector": "c"})
	assert.Equal(t, "B", got, "non-matching falls back to default")
	assert.NoError(t, err)
}

func TestNumberSelectors(t *testing.T) {
	src := "select = {$selector ->\n    [0] A\n   *[1] B\n}\n"
	b := newTestBundle(t, src)

	got, err := format(t, b, "select", map[string]any{"selector": 0})
	assert.Equal(t, "A", got, "matching")
	assert.NoError(t, err)

	got, err = format(t, b, "select", map[string]any{"selector": 2})
	assert.Equal(t, "B", got, "non-matching falls back to default")
	assert.NoError(t, err)
}

func TestPluralCategoryStringMatch(t *testing.T) {
	// A string selector matching a CLDR-category-named variant works without a
	// real plural ruleset: the keys are compared as strings.
	src := "select = {$selector ->\n    [one] A\n   *[other] B\n}\n"
	b := newTestBundle(t, src)

	got, err := format(t, b, "select", map[string]any{"selector": "one"})
	assert.Equal(t, "A", got, "string one")
	assert.NoError(t, err)

	got, err = format(t, b, "select", map[string]any{"selector": "other"})
	assert.Equal(t, "B", got, "string other")
	assert.NoError(t, err)
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
			got, err := format(t, b, "foo", nil)
			assert.Equal(t, tc.want, got)
			assert.NoError(t, err)
		})
	}
}

// TestNumberLiteralPluralCategory mirrors fluent.js: `{ 1 -> [one] A *[other] B }`
// selects "A" because CLDR plural rules categorize 1 as "one" in en-US. The
// CLDR-backed default does this out of the box — the plural rule tables are
// always linked, independent of which locale data is imported.
func TestNumberLiteralPluralCategory(t *testing.T) {
	src := "foo = { 1 ->\n    [one] A\n   *[other] B\n}\n"

	b := newTestBundle(t, src)
	got, err := format(t, b, "foo", nil)
	assert.Equal(t, "A", got, "CLDR default selects the one category")
	assert.NoError(t, err)

	// WithPluralRules overrides the default: this stub reports "other" for every
	// number, so the *[other] variant "B" is selected instead.
	b2 := fluent.NewBundle("en-US", fluent.WithUseIsolating(false), fluent.WithPluralRules(otherOnlyPluralRules{}))
	b2.AddResource(fluent.NewResource(src))
	got2, _ := format(t, b2, "foo", nil)
	assert.Equal(t, "B", got2, "override forces the other category")
}

// otherOnlyPluralRules is a stub reporting the "other" category for every number,
// used to exercise the WithPluralRules override hook.
type otherOnlyPluralRules struct{}

func (otherOnlyPluralRules) Cardinal(_ string, _ float64, _ fluent.NumberOptions) string {
	return "other"
}
func (otherOnlyPluralRules) Ordinal(_ string, _ float64, _ fluent.NumberOptions) string {
	return "other"
}

// splitPluralRules reports different constant categories for the cardinal and
// ordinal rulesets, so a test can verify which one the resolver consulted.
type splitPluralRules struct{}

func (splitPluralRules) Cardinal(_ string, _ float64, _ fluent.NumberOptions) string {
	return "other"
}
func (splitPluralRules) Ordinal(_ string, _ float64, _ fluent.NumberOptions) string {
	return "one"
}

// TestWithPluralRulesOrdinalDispatch verifies that a Number argument carrying
// NumberOptions.Type == "ordinal" is categorized via PluralRules.Ordinal while
// plain numbers go through Cardinal — using an override whose two rulesets
// disagree on every input.
func TestWithPluralRulesOrdinalDispatch(t *testing.T) {
	src := "foo = { $n ->\n    [one] A\n   *[other] B\n}\n"
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false), fluent.WithPluralRules(splitPluralRules{}))
	b.AddResource(fluent.NewResource(src))

	ordinal := fluent.NewNumber(1, fluent.NumberOptions{Type: fluent.Ordinal})
	got, err := format(t, b, "foo", map[string]any{"n": ordinal})
	assert.Equal(t, "A", got, "ordinal numbers consult the override's Ordinal ruleset")
	assert.NoError(t, err)

	// Plain (cardinal) 1 hits the stub's Cardinal ("other"), even though real
	// CLDR rules would say "one" — proving the override is in effect.
	got, err = format(t, b, "foo", map[string]any{"n": 1})
	assert.Equal(t, "B", got, "cardinal numbers consult the override's Cardinal ruleset")
	assert.NoError(t, err)
}

// A panic inside an injected plural ruleset must be recovered like a function
// panic: selection falls through to the default variant and the error is
// collected instead of escaping FormatPattern.
func TestPanickingPluralRulesRecovered(t *testing.T) {
	src := "sel = { $n ->\n    [one] A\n   *[other] B\n}\n"
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false), fluent.WithPluralRules(panicPluralRules{}))
	b.AddResource(fluent.NewResource(src))

	var got string
	var err error
	require.NotPanics(t, func() {
		got, err = format(t, b, "sel", map[string]any{"n": 1})
	}, "a panicking plural ruleset must not escape FormatPattern")
	assert.Equal(t, "B", got, "selection falls through to the default variant")
	require.Error(t, err, "the panic is collected as an error")
}

// A NUMBER() selector carrying a deferred option error must surface that error
// even though a selector is never formatted: matching still proceeds on the raw
// value, but the RangeError is reported exactly once.
func TestSelectorOptionErrorReported(t *testing.T) {
	src := "sel = { NUMBER($n, minimumFractionDigits: 999) ->\n    [1] ONE\n   *[other] OTHER\n}\n"
	b := newTestBundle(t, src)

	got, err := format(t, b, "sel", map[string]any{"n": 1})
	assert.Equal(t, "ONE", got, "selection still matches on the raw value")
	require.Len(t, errList(err), 1, "the deferred option error surfaces in selector position")
	require.ErrorIs(t, err, fluent.ErrRange)
}
