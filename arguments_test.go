package fluent

import (
	"testing"
	"time"
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
	if got != "Foo 3" || len(errs) != 0 {
		t.Errorf("foo: %q %v", got, errs)
	}
	got, errs = format(t, b, "bar", map[string]any{"num": 3})
	if got != "Foo 3" || len(errs) != 0 {
		t.Errorf("bar: %q %v", got, errs)
	}
	msg, _ := b.GetMessage("baz")
	var aerr []error
	if got := b.FormatPatternAny(msg.Attributes["attr"], map[string]any{"num": 3}, &aerr); got != "Baz Attribute 3" {
		t.Errorf("baz.attr: %q", got)
	}
	got, errs = format(t, b, "qux", map[string]any{"num": 3})
	if got != "Baz Variant A 3" || len(errs) != 0 {
		t.Errorf("qux: %q %v", got, errs)
	}
}

func TestVariableMissing(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	got, errs := format(t, b, "foo", map[string]any{})
	if got != "{$arg}" {
		t.Errorf("got %q", got)
	}
	if len(errs) != 1 || !isReferenceError(errs[0]) {
		t.Errorf("expected reference error, got %v", errs)
	}
}

func TestVariableUnsupportedTypes(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	cases := []any{
		[]int{1, 2, 3},
		map[string]int{"prop": 1},
		true,
		nil,
		func() {},
	}
	for _, c := range cases {
		got, errs := format(t, b, "foo", map[string]any{"arg": c})
		if got != "{$arg}" {
			t.Errorf("value %v: got %q want {$arg}", c, got)
		}
		if len(errs) != 1 || !isTypeError(errs[0]) {
			t.Errorf("value %v: expected type error, got %v", c, errs)
		}
	}
}

func TestVariableStringAndNumber(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")

	got, errs := format(t, b, "foo", map[string]any{"arg": "Argument"})
	if got != "Argument" || len(errs) != 0 {
		t.Errorf("string: %q %v", got, errs)
	}
	got, errs = format(t, b, "foo", map[string]any{"arg": 1})
	if got != "1" || len(errs) != 0 {
		t.Errorf("number: %q %v", got, errs)
	}

	// FluentNumber with minimumFractionDigits=2 renders 1.00.
	arg := NewNumber(1, NumberOptions{MinimumFractionDigits: intPtr(2)})
	msg, _ := b.GetMessage("foo")
	var verr []error
	if got := b.FormatPattern(msg.Value, map[string]Value{"arg": arg}, &verr); got != "1.00" {
		t.Errorf("FluentNumber: got %q want 1.00", got)
	}
}

func TestVariableDate(t *testing.T) {
	b := newTestBundle(t, "foo = { $arg }\n")
	arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
	got, errs := format(t, b, "foo", map[string]any{"arg": arg})
	// Default datetime formatter renders ISO-8601 UTC.
	want := "2016-09-29T00:00:00.000Z"
	if got != want || len(errs) != 0 {
		t.Errorf("date: got %q want %q errs %v", got, want, errs)
	}
}

// customValue is a user-defined Value type (mirrors the CustomType test).
type customValue struct{}

func (customValue) Format(_ *Scope) string { return "CUSTOM" }

func TestCustomArgumentType(t *testing.T) {
	src := "foo = { $arg }\nbar = { foo }\n"
	b := newTestBundle(t, src)
	args := map[string]Value{"arg": customValue{}}

	msg, _ := b.GetMessage("foo")
	var errs []error
	if got := b.FormatPattern(msg.Value, args, &errs); got != "CUSTOM" {
		t.Errorf("interpolation: got %q", got)
	}
	msg, _ = b.GetMessage("bar")
	if got := b.FormatPattern(msg.Value, args, &errs); got != "CUSTOM" {
		t.Errorf("nested: got %q", got)
	}
	if len(errs) != 0 {
		t.Errorf("errs: %v", errs)
	}
}
