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

// TestNumberLiteralPluralCategory mirrors fluent.js: `{ 1 -> [one] A *[other] B }`
// selects "A" because CLDR plural rules categorize 1 as "one" in en-US. The
// CLDR-backed default does this out of the box — the plural rule tables are
// always linked, independent of which locale data is imported.
func TestNumberLiteralPluralCategory(t *testing.T) {
	src := "foo = { 1 ->\n    [one] A\n   *[other] B\n}\n"

	b := newTestBundle(t, src)
	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "A", got, "CLDR default selects the one category")
	assert.Empty(t, errs)

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
