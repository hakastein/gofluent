package fluent_test

import (
	"testing"
	"time"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ported from arguments_test.js.

func TestVariablesInValues(t *testing.T) {
	src := "foo = Foo { $num }\n" +
		"bar = { foo }\n" +
		"baz =\n" +
		"    .attr = Baz Attribute { $num }\n" +
		"qux = { \"a\" ->\n" +
		"   *[a]     Baz Variant A { $num }\n" +
		"}\n"
	b := newTestBundle(t, src)

	got, errs := format(t, b, "foo", map[string]any{"num": 3})
	assert.Equal(t, "Foo 3", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "bar", map[string]any{"num": 3})
	assert.Equal(t, "Foo 3", got)
	assert.Empty(t, errs)

	msg, _ := b.Message("baz")
	got, errs = b.FormatPattern(msg.Attributes["attr"], map[string]any{"num": 3})
	assert.Equal(t, "Baz Attribute 3", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "qux", map[string]any{"num": 3})
	assert.Equal(t, "Baz Variant A 3", got)
	assert.Empty(t, errs)
}

func TestVariableMissing(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	got, errs := format(t, b, "foo", map[string]any{})
	assert.Equal(t, "{$arg}", got)
	require.Len(t, errs, 1, "expected a single reference error")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
}

func TestVariableUnsupportedTypes(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	cases := []struct {
		name string
		arg  any
	}{
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"prop": 1}},
		{"bool", true},
		{"nil", nil},
		{"func", func() {}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := format(t, b, "foo", map[string]any{"arg": tc.arg})
			assert.Equal(t, "{$arg}", got)
			require.Len(t, errs, 1, "expected a single type error")
			require.ErrorIs(t, errs[0], fluent.ErrType)
		})
	}
}

func TestVariableStringAndNumber(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")

	got, errs := format(t, b, "foo", map[string]any{"arg": "Argument"})
	assert.Equal(t, "Argument", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "foo", map[string]any{"arg": 1})
	assert.Equal(t, "1", got)
	assert.Empty(t, errs)

	// A Number argument with minimumFractionDigits=2 renders 1.00.
	arg := fluent.NewNumber(1, fluent.NumberOptions{MinimumFractionDigits: intPtr(2)})
	got, errs = format(t, b, "foo", map[string]any{"arg": arg})
	assert.Equal(t, "1.00", got)
	assert.Empty(t, errs)
}

func TestVariableDate(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
	got, errs := format(t, b, "foo", map[string]any{"arg": arg})
	// CLDR default: en-US short date.
	assert.Equal(t, "9/29/2016", got)
	assert.Empty(t, errs)
}

// customValue is a user-defined Value type (mirrors the CustomType test).
type customValue struct{}

func (customValue) Format(_ *fluent.Scope) string { return "CUSTOM" }

func TestCustomArgumentType(t *testing.T) {
	src := "foo = { $arg }\nbar = { foo }\n"
	b := newTestBundle(t, src)
	args := map[string]any{"arg": customValue{}}

	got, errs := format(t, b, "foo", args)
	assert.Equal(t, "CUSTOM", got, "interpolation")
	assert.Empty(t, errs)

	got, errs = format(t, b, "bar", args)
	assert.Equal(t, "CUSTOM", got, "nested")
	assert.Empty(t, errs)
}
