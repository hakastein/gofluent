package fluent_test

import (
	"errors"
	"math"
	"testing"
	"time"

	fluent "github.com/hakastein/gofluent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ported from functions_builtin_test.js and functions_runtime_test.js.
//
// NewBundle installs the CLDR-backed formatters by default; the test binary
// blank-imports gocldr/locales/all (see format_cldr_test.go) to supply the
// locale data for the numeric and date assertions.

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
		// CLDR default: en-US grouping.
		got, errs := format(t, b, "num-bare", map[string]any{"arg": 1234})
		assert.Equal(t, "1,234", got)
		assert.Empty(t, errs)

		got, errs = format(t, b, "num-fraction-valid", map[string]any{"arg": 1234})
		assert.Equal(t, "1,234.0", got)
		assert.Empty(t, errs)

		// Bad option value -> RangeError, falls back to plain value.
		got, errs = format(t, b, "num-fraction-bad", map[string]any{"arg": 1234})
		assert.Equal(t, "1234", got)
		require.Len(t, errs, 1, "expected a single range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
	})

	t.Run("fractional integer-option value is invalid", func(t *testing.T) {
		fb := newTestBundle(t, "num-frac-num = { NUMBER($arg, minimumFractionDigits: 2.9) }\n")
		got, errs := format(t, fb, "num-frac-num", map[string]any{"arg": 1234})
		assert.Equal(t, "1234", got)
		require.Len(t, errs, 1, "expected a single range error")
		require.ErrorIs(t, errs[0], fluent.ErrRange)
	})

	t.Run("out-of-range integer option is invalid", func(t *testing.T) {
		fb := newTestBundle(t, "num-frac-big = { NUMBER($arg, minimumFractionDigits: 999) }\n")
		got, errs := format(t, fb, "num-frac-big", map[string]any{"arg": 1234})
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

// NUMBER() ignores options outside fluent.js's allowlist (style, currency,
// unit, ...): a translation cannot change what kind of quantity a number is.
// Those options reach the formatter only on a Number argument built in code.
func TestNumberBuiltinOptionAllowlist(t *testing.T) {
	rec := &recordingNumberFormatter{}
	b := fluent.NewBundle("en-US",
		fluent.WithUseIsolating(false),
		fluent.WithNumberFormatter(rec),
	)
	b.AddResource(fluent.NewResource("n = { NUMBER($arg, style: \"unit\", unit: \"kilometer\", minimumFractionDigits: 1) }\n"))

	arg := fluent.NewNumber(5, fluent.NumberOptions{Style: "currency", Currency: "USD"})
	got, errs := format(t, b, "n", map[string]any{"arg": arg})
	assert.Equal(t, "fmt", got)
	assert.Empty(t, errs)
	assert.Equal(t, "currency", rec.last.Style, "argument options are forwarded")
	assert.Equal(t, "USD", rec.last.Currency)
	assert.Empty(t, rec.last.Unit, "disallowed FTL options are ignored")
	require.NotNil(t, rec.last.MinimumFractionDigits, "allowed FTL options are merged")
	assert.Equal(t, 1, *rec.last.MinimumFractionDigits)
}

func TestNumberBuiltinFluentNumberMerge(t *testing.T) {
	src := "num-bare = { NUMBER($arg) }\n" +
		"num-fraction-valid = { NUMBER($arg, minimumFractionDigits: 1) }\n"
	b := newTestBundle(t, src)

	// minimumFractionDigits=3 from the arg is retained unless overridden.
	arg := fluent.NewNumber(1234, fluent.NumberOptions{MinimumFractionDigits: intPtr(3)})
	got, errs := format(t, b, "num-bare", map[string]any{"arg": arg})
	assert.Equal(t, "1,234.000", got, "bare retains arg fraction digits")
	assert.Empty(t, errs)

	// The call's minimumFractionDigits:1 overrides the arg's 3.
	got, errs = format(t, b, "num-fraction-valid", map[string]any{"arg": arg})
	assert.Equal(t, "1,234.0", got, "call overrides arg fraction digits")
	assert.Empty(t, errs)
}

func TestNumberBuiltinFromDateTime(t *testing.T) {
	// NUMBER on a DateTime argument yields its epoch-millis number.
	b := newTestBundle(t, "num-bare = { NUMBER($arg) }\n")
	date := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
	arg := fluent.NewDateTime(date, fluent.DateTimeOptions{Month: "short", Day: "numeric"})
	got, errs := format(t, b, "num-bare", map[string]any{"arg": arg})
	assert.Equal(t, "1,475,107,200,000", got)
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
		// CLDR default: en-US short date.
		arg := time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC)
		got, errs := format(t, b, "dt-bare", map[string]any{"arg": arg})
		assert.Equal(t, "9/29/2016", got)
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
		assert.Equal(t, "1/1/1970", got)
		assert.Empty(t, errs)
	})

	t.Run("non-finite number argument is invalid", func(t *testing.T) {
		// Intl.DateTimeFormat throws a RangeError outside ECMA-262's
		// ±8.64e15 ms time value range; NaN and Inf are equally invalid.
		for _, arg := range []float64{math.NaN(), math.Inf(1), 8.64e15 + 1} {
			got, errs := format(t, b, "dt-bare", map[string]any{"arg": arg})
			assert.Equal(t, "{DATETIME()}", got)
			require.Len(t, errs, 1, "expected a single range error")
			require.ErrorIs(t, errs[0], fluent.ErrRange)
		}
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

// DATETIME() ignores options outside fluent.js's allowlist (timeZone,
// calendar, numberingSystem): those reach the formatter only on a DateTime
// argument built in code.
func TestDateTimeBuiltinOptionAllowlist(t *testing.T) {
	rec := &recordingDateTimeFormatter{}
	b := fluent.NewBundle("en-US",
		fluent.WithUseIsolating(false),
		fluent.WithDateTimeFormatter(rec),
	)
	src := "tz = { DATETIME($arg, timeZone: \"America/New_York\") }\n" +
		"cal = { DATETIME($arg, calendar: \"buddhist\") }\n" +
		"ns = { DATETIME($arg, numberingSystem: \"arab\") }\n" +
		"wd = { DATETIME($arg, weekday: \"long\") }\n"
	b.AddResource(fluent.NewResource(src))

	arg := fluent.NewDateTime(time.Date(2016, 9, 29, 0, 0, 0, 0, time.UTC),
		fluent.DateTimeOptions{TimeZone: "Europe/Berlin"})

	t.Run("allowed FTL option is forwarded", func(t *testing.T) {
		got, errs := format(t, b, "wd", map[string]any{"arg": arg})
		assert.Equal(t, "fmt", got)
		assert.Empty(t, errs)
		assert.Equal(t, "long", rec.last.Weekday)
	})

	for _, id := range []string{"tz", "cal", "ns"} {
		t.Run(id+" is ignored", func(t *testing.T) {
			got, errs := format(t, b, id, map[string]any{"arg": arg})
			assert.Equal(t, "fmt", got)
			assert.Empty(t, errs)
			assert.Equal(t, fluent.DateTimeOptions{TimeZone: "Europe/Berlin"}, rec.last,
				"only the argument's own options survive")
		})
	}
}

func TestRuntimeFunctions(t *testing.T) {
	concat := func(args []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		s := ""
		for _, a := range args {
			s += a.Format(nil)
		}
		return fluent.String(s), nil
	}
	sum := func(args []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		total := 0.0
		for _, a := range args {
			if n, ok := a.(*fluent.Number); ok {
				total += n.Value
			}
		}
		return fluent.NewNumber(total, fluent.NumberOptions{}), nil
	}
	platform := func(_ []fluent.Value, _ map[string]fluent.Value) (fluent.Value, error) {
		return fluent.String("windows"), nil
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
	b.AddResource(fluent.NewResource(src))

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
	b.AddResource(fluent.NewResource("foo = { BOOM() }\n"))

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
		b.AddResource(fluent.NewResource("foo = { PANICERR() }\n"))

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
		b.AddResource(fluent.NewResource("foo = { PANICSTR() }\n"))

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
			return fluent.String("unreachable"), nil
		}
		b := fluent.NewBundle("en-US", fluent.WithUseIsolating(false))
		b.AddFunction("NILDEREF", nilDeref)
		b.AddResource(fluent.NewResource("foo = { NILDEREF() }\n"))

		msg, ok := b.Message("foo")
		require.True(t, ok)
		assert.Panics(t, func() {
			b.FormatPattern(msg.Value, nil)
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
