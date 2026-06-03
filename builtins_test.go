package fluent

import (
	"testing"
	"time"
)

// Ported from functions_builtin_test.js and functions_runtime_test.js.
//
// ADAPTATION: fluent.js asserts CLDR/Intl output such as "1,234" (grouping) and
// "$1,234.00" (currency). The dependency-free default NumberFormatter does not
// do locale grouping or currency symbols, so the numeric assertions below match
// the DEFAULT formatter behavior (plain decimal honoring min/max fraction
// digits). The option-merging, type dispatch, and error semantics are identical
// to fluent.js.

func TestNumberBuiltinDefaults(t *testing.T) {
	src := "num-bare = { NUMBER($arg) }\n" +
		"num-fraction-valid = { NUMBER($arg, minimumFractionDigits: 1) }\n" +
		"num-fraction-bad = { NUMBER($arg, minimumFractionDigits: \"oops\") }\n" +
		"num-style = { NUMBER($arg, style: \"percent\") }\n" +
		"num-unknown = { NUMBER($arg, unknown: \"unknown\") }\n"
	b := newTestBundle(t, src)

	t.Run("missing argument", func(t *testing.T) {
		for _, id := range []string{"num-bare", "num-fraction-valid", "num-fraction-bad", "num-style", "num-unknown"} {
			got, errs := format(t, b, id, map[string]any{})
			if got != "{NUMBER($arg)}" {
				t.Errorf("%s: got %q", id, got)
			}
			if len(errs) != 1 || !isReferenceError(errs[0]) {
				t.Errorf("%s: expected reference error, got %v", id, errs)
			}
		}
	})

	t.Run("number argument", func(t *testing.T) {
		// Default formatter: no grouping.
		got, errs := format(t, b, "num-bare", map[string]any{"arg": 1234})
		if got != "1234" || len(errs) != 0 {
			t.Errorf("num-bare: got %q errs %v", got, errs)
		}
		got, errs = format(t, b, "num-fraction-valid", map[string]any{"arg": 1234})
		if got != "1234.0" || len(errs) != 0 {
			t.Errorf("num-fraction-valid: got %q errs %v", got, errs)
		}
		// Bad option value -> RangeError, falls back to plain value.
		got, errs = format(t, b, "num-fraction-bad", map[string]any{"arg": 1234})
		if got != "1234" {
			t.Errorf("num-fraction-bad: got %q", got)
		}
		if len(errs) != 1 || !isRangeError(errs[0]) {
			t.Errorf("num-fraction-bad: expected range error, got %v", errs)
		}
	})

	t.Run("string argument is invalid", func(t *testing.T) {
		got, errs := format(t, b, "num-bare", map[string]any{"arg": "Foo"})
		if got != "{NUMBER()}" {
			t.Errorf("got %q", got)
		}
		if len(errs) != 1 || !isTypeError(errs[0]) {
			t.Errorf("expected type error, got %v", errs)
		}
	})

	t.Run("unsupported argument", func(t *testing.T) {
		got, errs := format(t, b, "num-bare", map[string]any{"arg": []int{}})
		if got != "{NUMBER($arg)}" {
			t.Errorf("got %q", got)
		}
		if len(errs) != 1 || !isTypeError(errs[0]) {
			t.Errorf("expected type error, got %v", errs)
		}
	})
}

func TestNumberBuiltinFluentNumberMerge(t *testing.T) {
	src := "num-bare = { NUMBER($arg) }\n" +
		"num-fraction-valid = { NUMBER($arg, minimumFractionDigits: 1) }\n"
	b := newTestBundle(t, src)

	// minimumFractionDigits=3 from the arg is retained unless overridden.
	arg := NewNumber(1234, NumberOptions{MinimumFractionDigits: intPtr(3)})
	msg, _ := b.GetMessage("num-bare")
	var errs []error
	if got := b.FormatPattern(msg.Value, map[string]Value{"arg": arg}, &errs); got != "1234.000" {
		t.Errorf("bare: got %q want 1234.000", got)
	}
	// The call's minimumFractionDigits:1 overrides the arg's 3.
	msg, _ = b.GetMessage("num-fraction-valid")
	if got := b.FormatPattern(msg.Value, map[string]Value{"arg": arg}, &errs); got != "1234.0" {
		t.Errorf("override: got %q want 1234.0", got)
	}
	if len(errs) != 0 {
		t.Errorf("errs: %v", errs)
	}
}

func TestNumberBuiltinFromDateTime(t *testing.T) {
	// NUMBER on a FluentDateTime yields its epoch-millis number.
	b := newTestBundle(t, "num-bare = { NUMBER($arg) }\n")
	date := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
	arg := NewDateTime(date, DateTimeOptions{Month: "short", Day: "numeric"})
	msg, _ := b.GetMessage("num-bare")
	var errs []error
	got := b.FormatPattern(msg.Value, map[string]Value{"arg": arg}, &errs)
	if got != "1475107200000" || len(errs) != 0 {
		t.Errorf("got %q errs %v", got, errs)
	}
}

func TestDateTimeBuiltin(t *testing.T) {
	src := "dt-bare = { DATETIME($arg) }\n" +
		"dt-month-valid = { DATETIME($arg, month: \"long\") }\n"
	b := newTestBundle(t, src)

	t.Run("missing argument", func(t *testing.T) {
		got, errs := format(t, b, "dt-bare", map[string]any{})
		if got != "{DATETIME($arg)}" {
			t.Errorf("got %q", got)
		}
		if len(errs) != 1 || !isReferenceError(errs[0]) {
			t.Errorf("expected reference error, got %v", errs)
		}
	})

	t.Run("date argument default rendering", func(t *testing.T) {
		// ADAPTATION: default formatter renders ISO-8601 UTC.
		arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": arg})
		if got != "2016-09-29T00:00:00.000Z" || len(errs) != 0 {
			t.Errorf("got %q errs %v", got, errs)
		}
	})

	t.Run("string argument is invalid", func(t *testing.T) {
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": "Foo"})
		if got != "{DATETIME()}" {
			t.Errorf("got %q", got)
		}
		if len(errs) != 1 || !isTypeError(errs[0]) {
			t.Errorf("expected type error, got %v", errs)
		}
	})

	t.Run("number argument becomes datetime", func(t *testing.T) {
		// 0 ms since epoch.
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": 0})
		if got != "1970-01-01T00:00:00.000Z" || len(errs) != 0 {
			t.Errorf("got %q errs %v", got, errs)
		}
	})
}

func TestRuntimeFunctions(t *testing.T) {
	concat := func(args []Value, _ map[string]Value) (Value, error) {
		s := ""
		for _, a := range args {
			s += a.Format(nil)
		}
		return FluentString(s), nil
	}
	sum := func(args []Value, _ map[string]Value) (Value, error) {
		total := 0.0
		for _, a := range args {
			if n, ok := a.(*Number); ok {
				total += n.Value()
			}
		}
		return NewNumber(total, NumberOptions{}), nil
	}
	platform := func(_ []Value, _ map[string]Value) (Value, error) {
		return FluentString("windows"), nil
	}

	b := NewBundle("en-US", WithUseIsolating(false), WithFunctions(map[string]Function{
		"CONCAT":   concat,
		"SUM":      sum,
		"PLATFORM": platform,
	}))
	src := "foo = { CONCAT(\"Foo\", \"Bar\") }\n" +
		"bar = { SUM(1, 2) }\n" +
		"pref =\n" +
		"  { PLATFORM() ->\n" +
		"      [windows] Options\n" +
		"     *[other] Preferences\n" +
		"  }\n"
	b.AddResource(mustParse(t, src))

	got, errs := format(t, b, "foo", nil)
	if got != "FooBar" || len(errs) != 0 {
		t.Errorf("foo: %q %v", got, errs)
	}
	got, errs = format(t, b, "bar", nil)
	if got != "3" || len(errs) != 0 {
		t.Errorf("bar: %q %v", got, errs)
	}
	got, errs = format(t, b, "pref", nil)
	if got != "Options" || len(errs) != 0 {
		t.Errorf("pref: %q %v", got, errs)
	}
}

func TestFunctionErrorIsRecovered(t *testing.T) {
	boom := func(_ []Value, _ map[string]Value) (Value, error) {
		return nil, newTypeError("boom")
	}
	b := NewBundle("en-US", WithUseIsolating(false))
	b.AddFunction("BOOM", boom)
	b.AddResource(mustParse(t, "foo = { BOOM() }\n"))

	got, errs := format(t, b, "foo", nil)
	if got != "{BOOM()}" {
		t.Errorf("got %q", got)
	}
	if len(errs) != 1 || !isTypeError(errs[0]) {
		t.Errorf("expected type error, got %v", errs)
	}
}

func TestUnknownFunction(t *testing.T) {
	b := newTestBundle(t, "foo = { MISSING() }\n")
	got, errs := format(t, b, "foo", nil)
	if got != "{MISSING()}" {
		t.Errorf("got %q", got)
	}
	if len(errs) != 1 || !isReferenceError(errs[0]) {
		t.Errorf("expected reference error, got %v", errs)
	}
}
