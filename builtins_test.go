package fluent_test

import (
	"errors"
	"testing"
	"time"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equalf(t, "{NUMBER($arg)}", got, "%s", id)
			require.Lenf(t, errs, 1, "%s: expected a single reference error", id)
			require.ErrorIsf(t, errs[0], fluent.ErrReference, "%s", id)
		}
	})

	t.Run("number argument", func(t *testing.T) {
		// Default formatter: no grouping.
		got, errs := format(t, b, "num-bare", map[string]any{"arg": 1234})
		assert.Equal(t, "1234", got)
		assert.Empty(t, errs)

		got, errs = format(t, b, "num-fraction-valid", map[string]any{"arg": 1234})
		assert.Equal(t, "1234.0", got)
		assert.Empty(t, errs)

		// Bad option value -> RangeError, falls back to plain value.
		got, errs = format(t, b, "num-fraction-bad", map[string]any{"arg": 1234})
		assert.Equal(t, "1234", got)
		require.Len(t, errs, 1, "expected a single range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
	})

	t.Run("fractional integer-option value is invalid", func(t *testing.T) {
		// A non-integral Number option (e.g. 2.9) is out of spec for an integer
		// option and routes to the same RangeError as the string "2.9", rather
		// than being silently truncated to 2.
		fb := newTestBundle(t, "num-frac-num = { NUMBER($arg, minimumFractionDigits: 2.9) }\n")
		got, errs := format(t, fb, "num-frac-num", map[string]any{"arg": 1234})
		assert.Equal(t, "1234", got)
		require.Len(t, errs, 1, "expected a single range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
	})

	t.Run("string argument is invalid", func(t *testing.T) {
		got, errs := format(t, b, "num-bare", map[string]any{"arg": "Foo"})
		assert.Equal(t, "{NUMBER()}", got)
		require.Len(t, errs, 1, "expected a single type error")
		require.ErrorIs(t, errs[0], fluent.ErrType)
	})

	t.Run("unsupported argument", func(t *testing.T) {
		got, errs := format(t, b, "num-bare", map[string]any{"arg": []int{}})
		assert.Equal(t, "{NUMBER($arg)}", got)
		require.Len(t, errs, 1, "expected a single type error")
		require.ErrorIs(t, errs[0], fluent.ErrType)
	})
}

// recordingNumberFormatter captures the NumberOptions it is handed so a test
// can assert which named options NUMBER() forwarded.
type recordingNumberFormatter struct{ last fluent.NumberOptions }

func (f *recordingNumberFormatter) FormatNumber(_ string, _ float64, opts fluent.NumberOptions) string {
	f.last = opts
	return "fmt"
}

func TestNumberBuiltinForwardsUnit(t *testing.T) {
	rec := &recordingNumberFormatter{}
	b := fluent.NewBundle("en-US",
		fluent.WithUseIsolating(false),
		fluent.WithNumberFormatter(rec),
	)
	b.AddResource(mustParse(t, "n = { NUMBER($arg, style: \"unit\", unit: \"kilometer\") }\n"))

	got, errs := format(t, b, "n", map[string]any{"arg": 5})
	assert.Equal(t, "fmt", got)
	assert.Empty(t, errs)
	assert.Equal(t, "kilometer", rec.last.Unit)
	assert.Equal(t, "unit", rec.last.Style)
}

func TestNumberBuiltinFluentNumberMerge(t *testing.T) {
	src := "num-bare = { NUMBER($arg) }\n" +
		"num-fraction-valid = { NUMBER($arg, minimumFractionDigits: 1) }\n"
	b := newTestBundle(t, src)

	// minimumFractionDigits=3 from the arg is retained unless overridden.
	arg := fluent.NewNumber(1234, fluent.NumberOptions{MinimumFractionDigits: intPtr(3)})
	msg, _ := b.GetMessage("num-bare")
	var errs []error
	assert.Equal(t, "1234.000", b.FormatPattern(msg.Value, map[string]fluent.Value{"arg": arg}, &errs), "bare retains arg fraction digits")

	// The call's minimumFractionDigits:1 overrides the arg's 3.
	msg, _ = b.GetMessage("num-fraction-valid")
	assert.Equal(t, "1234.0", b.FormatPattern(msg.Value, map[string]fluent.Value{"arg": arg}, &errs), "call overrides arg fraction digits")

	assert.Empty(t, errs)
}

func TestNumberBuiltinFromDateTime(t *testing.T) {
	// NUMBER on a FluentDateTime yields its epoch-millis number.
	b := newTestBundle(t, "num-bare = { NUMBER($arg) }\n")
	date := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
	arg := fluent.NewDateTime(date, fluent.DateTimeOptions{Month: "short", Day: "numeric"})
	msg, _ := b.GetMessage("num-bare")
	var errs []error
	assert.Equal(t, "1475107200000", b.FormatPattern(msg.Value, map[string]fluent.Value{"arg": arg}, &errs))
	assert.Empty(t, errs)
}

func TestDateTimeBuiltin(t *testing.T) {
	src := "dt-bare = { DATETIME($arg) }\n" +
		"dt-month-valid = { DATETIME($arg, month: \"long\") }\n"
	b := newTestBundle(t, src)

	t.Run("missing argument", func(t *testing.T) {
		got, errs := format(t, b, "dt-bare", map[string]any{})
		assert.Equal(t, "{DATETIME($arg)}", got)
		require.Len(t, errs, 1, "expected a single reference error")
		require.ErrorIs(t, errs[0], fluent.ErrReference)
	})

	t.Run("date argument default rendering", func(t *testing.T) {
		// ADAPTATION: default formatter renders ISO-8601 UTC.
		arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": arg})
		assert.Equal(t, "2016-09-29T00:00:00.000Z", got)
		assert.Empty(t, errs)
	})

	t.Run("string argument is invalid", func(t *testing.T) {
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": "Foo"})
		assert.Equal(t, "{DATETIME()}", got)
		require.Len(t, errs, 1, "expected a single type error")
		require.ErrorIs(t, errs[0], fluent.ErrType)
	})

	t.Run("number argument becomes datetime", func(t *testing.T) {
		// 0 ms since epoch.
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": 0})
		assert.Equal(t, "1970-01-01T00:00:00.000Z", got)
		assert.Empty(t, errs)
	})
}

// recordingDateTimeFormatter captures the DateTimeOptions it is handed so a
// test can assert that the DATETIME() builtin forwarded a given named option
// into the right field.
type recordingDateTimeFormatter struct{ last fluent.DateTimeOptions }

func (f *recordingDateTimeFormatter) FormatDateTime(_ string, _ time.Time, opts fluent.DateTimeOptions) string {
	f.last = opts
	return "fmt"
}

func TestDateTimeBuiltinForwardsOptions(t *testing.T) {
	rec := &recordingDateTimeFormatter{}
	b := fluent.NewBundle("en-US",
		fluent.WithUseIsolating(false),
		fluent.WithDateTimeFormatter(rec),
	)
	src := "tz = { DATETIME($arg, timeZone: \"America/New_York\") }\n" +
		"cal = { DATETIME($arg, calendar: \"buddhist\") }\n" +
		"ns = { DATETIME($arg, numberingSystem: \"arab\") }\n"
	b.AddResource(mustParse(t, src))

	arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		id   string
		get  func(fluent.DateTimeOptions) string
		want string
	}{
		{"timeZone", "tz", func(o fluent.DateTimeOptions) string { return o.TimeZone }, "America/New_York"},
		{"calendar", "cal", func(o fluent.DateTimeOptions) string { return o.Calendar }, "buddhist"},
		{"numberingSystem", "ns", func(o fluent.DateTimeOptions) string { return o.NumberingSystem }, "arab"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := format(t, b, tc.id, map[string]any{"arg": arg})
			assert.Equal(t, "fmt", got)
			assert.Empty(t, errs)
			assert.Equal(t, tc.want, tc.get(rec.last))
		})
	}
}

func TestRuntimeFunctions(t *testing.T) {
	concat := func(args []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		s := ""
		for _, a := range args {
			s += a.Format(nil)
		}
		return fluent.FluentString(s), nil
	}
	sum := func(args []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		total := 0.0
		for _, a := range args {
			if n, ok := a.(*fluent.Number); ok {
				total += n.Value()
			}
		}
		return fluent.NewNumber(total, fluent.NumberOptions{}), nil
	}
	platform := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		return fluent.FluentString("windows"), nil
	}

	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false), fluent.WithFunctions(map[string]fluent.Function{
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
	assert.Equal(t, "FooBar", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "bar", nil)
	assert.Equal(t, "3", got)
	assert.Empty(t, errs)

	got, errs = format(t, b, "pref", nil)
	assert.Equal(t, "Options", got)
	assert.Empty(t, errs)
}

func TestFunctionErrorIsRecovered(t *testing.T) {
	// A function that returns any error has that error recovered and the call
	// rendered as the {FUNC()} fallback. The error's concrete kind is the
	// function's own choice, so a plain error exercises the public contract.
	boom := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		return nil, errors.New("boom")
	}
	b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
	b.AddFunction("BOOM", boom)
	b.AddResource(mustParse(t, "foo = { BOOM() }\n"))

	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "{BOOM()}", got)
	assert.Len(t, errs, 1, "the function error should be recovered into the sink")
}

func TestFunctionPanicHandling(t *testing.T) {
	// A function that panics with a custom error value is recovered like a
	// returned error: the call renders {FUNC()} and the error reaches the sink.
	t.Run("custom error panic is recovered", func(t *testing.T) {
		panicErr := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
			panic(errors.New("custom boom"))
		}
		b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
		b.AddFunction("PANICERR", panicErr)
		b.AddResource(mustParse(t, "foo = { PANICERR() }\n"))

		got, errs := format(t, b, "foo", nil)
		assert.Equal(t, "{PANICERR()}", got)
		assert.Len(t, errs, 1, "the panicked error should be recovered into the sink")
	})

	t.Run("non-error panic is recovered", func(t *testing.T) {
		panicStr := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
			panic("plain string boom")
		}
		b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
		b.AddFunction("PANICSTR", panicStr)
		b.AddResource(mustParse(t, "foo = { PANICSTR() }\n"))

		got, errs := format(t, b, "foo", nil)
		assert.Equal(t, "{PANICSTR()}", got)
		assert.Len(t, errs, 1, "the panicked value should be recovered into the sink")
	})

	t.Run("runtime panic propagates", func(t *testing.T) {
		// A genuine programming bug (nil map write -> runtime.Error) must NOT be
		// swallowed; it surfaces so the developer sees the real fault.
		nilDeref := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
			var m map[string]int
			//lint:ignore SA5000 intentional: asserts a genuine runtime panic is not swallowed
			m["x"] = 1 // assignment to entry in nil map -> runtime panic
			return fluent.FluentString("unreachable"), nil
		}
		b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
		b.AddFunction("NILDEREF", nilDeref)
		b.AddResource(mustParse(t, "foo = { NILDEREF() }\n"))

		msg, ok := b.GetMessage("foo")
		require.True(t, ok)
		assert.Panics(t, func() {
			var errs []error
			b.FormatPattern(msg.Value, nil, &errs)
		}, "a runtime panic inside a function must propagate")
	})
}

func TestUnknownFunction(t *testing.T) {
	b := newTestBundle(t, "foo = { MISSING() }\n")
	got, errs := format(t, b, "foo", nil)
	assert.Equal(t, "{MISSING()}", got)
	require.Len(t, errs, 1, "expected a single reference error")
	require.ErrorIs(t, errs[0], fluent.ErrReference)
}
