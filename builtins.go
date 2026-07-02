package fluent

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// optString unwraps a named option Value to its string form. A String is
// returned verbatim; a Number renders its plain value. Anything else yields
// ("", false). Mirrors opt.valueOf() in fluent.js.
func optString(v Value) (string, bool) {
	switch t := v.(type) {
	case String:
		return string(t), true
	case *Number:
		return strconv.FormatFloat(t.Value, 'f', -1, 64), true
	}
	return "", false
}

// optInt unwraps a named option Value to an int. A Number yields its integer
// value; a numeric String is parsed. Non-numeric strings yield a
// RangeError-equivalent (ok == false) so the builtin can mirror Intl's behavior
// of throwing on an invalid option value.
func optInt(v Value) (int, bool) {
	switch t := v.(type) {
	case *Number:
		// Fractional values must fail like non-numeric strings (Intl), not truncate.
		if t.Value != math.Trunc(t.Value) {
			return 0, false
		}
		return int(t.Value), true
	case String:
		if n, err := strconv.Atoi(strings.TrimSpace(string(t))); err == nil {
			return n, true
		}
	}
	return 0, false
}

// optBool unwraps a named option Value to a bool.
func optBool(v Value) (bool, bool) {
	switch t := v.(type) {
	case String:
		switch string(t) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	case *Number:
		return t.Value != 0, true
	}
	return false, false
}

// sortedKeys returns the option names in a stable order, so that when several
// options are invalid the first reported error does not depend on Go's random
// map iteration order.
func sortedKeys(opts map[string]Value) []string {
	names := make([]string, 0, len(opts))
	for name := range opts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// numberOptionsFrom merges the NUMBER_ALLOWED named options into a
// NumberOptions, starting from a base set of options. Options outside the
// fluent.js allowlist (style, currency, unit, ...) are ignored. An invalid
// integer option value (non-numeric, fractional, or out of its Intl range)
// returns a deferred range error: the NUMBER builtin still succeeds and the
// error surfaces at format time, mirroring Intl.NumberFormat throwing in its
// constructor-deferred way.
func numberOptionsFrom(base NumberOptions, opts map[string]Value) (NumberOptions, error) {
	out := base
	setInt := func(dst **int, name string, v Value, min, max int) error {
		n, ok := optInt(v)
		if !ok || n < min || n > max {
			return newRangeError("Invalid option value for " + name)
		}
		*dst = intPtr(n)
		return nil
	}
	var err error
	for _, name := range sortedKeys(opts) {
		v := opts[name]
		switch name {
		case "currencyDisplay":
			if s, ok := optString(v); ok {
				out.CurrencyDisplay = s
			}
		case "unitDisplay":
			if s, ok := optString(v); ok {
				out.UnitDisplay = s
			}
		case "useGrouping":
			if b, ok := optBool(v); ok {
				out.UseGrouping = boolPtr(b)
			}
		case "minimumIntegerDigits":
			err = setInt(&out.MinimumIntegerDigits, name, v, 1, 21)
		case "minimumFractionDigits":
			err = setInt(&out.MinimumFractionDigits, name, v, 0, 100)
		case "maximumFractionDigits":
			err = setInt(&out.MaximumFractionDigits, name, v, 0, 100)
		case "minimumSignificantDigits":
			err = setInt(&out.MinimumSignificantDigits, name, v, 1, 21)
		case "maximumSignificantDigits":
			err = setInt(&out.MaximumSignificantDigits, name, v, 1, 21)
		}
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// dateTimeOptionsFrom merges the DATETIME_ALLOWED named options into a
// DateTimeOptions. Options outside the fluent.js allowlist (timeZone,
// calendar, numberingSystem) are ignored. An invalid fractionalSecondDigits
// value returns a range error, mirroring Intl.DateTimeFormat.
func dateTimeOptionsFrom(base DateTimeOptions, opts map[string]Value) (DateTimeOptions, error) {
	out := base
	setStr := func(dst *string, v Value) {
		if s, ok := optString(v); ok {
			*dst = s
		}
	}
	for name, v := range opts {
		switch name {
		case "dateStyle":
			setStr(&out.DateStyle, v)
		case "timeStyle":
			setStr(&out.TimeStyle, v)
		case "dayPeriod":
			setStr(&out.DayPeriod, v)
		case "hour12":
			if b, ok := optBool(v); ok {
				out.Hour12 = boolPtr(b)
			}
		case "weekday":
			setStr(&out.Weekday, v)
		case "era":
			setStr(&out.Era, v)
		case "year":
			setStr(&out.Year, v)
		case "month":
			setStr(&out.Month, v)
		case "day":
			setStr(&out.Day, v)
		case "hour":
			setStr(&out.Hour, v)
		case "minute":
			setStr(&out.Minute, v)
		case "second":
			setStr(&out.Second, v)
		case "timeZoneName":
			setStr(&out.TimeZoneName, v)
		case "fractionalSecondDigits":
			n, ok := optInt(v)
			if !ok || n < 1 || n > 3 {
				return out, newRangeError("Invalid option value for fractionalSecondDigits")
			}
			out.FractionalSecondDigits = intPtr(n)
		}
	}
	return out, nil
}

// builtinNUMBER implements the NUMBER() builtin.
func builtinNUMBER(args []Value, opts map[string]Value) (Value, error) {
	var arg Value
	if len(args) > 0 {
		arg = args[0]
	}

	switch a := arg.(type) {
	case *None:
		return NewNone("NUMBER(" + a.fallback + ")"), nil
	case *Number:
		merged, optErr := numberOptionsFrom(a.Options, opts)
		n := NewNumber(a.Value, merged)
		n.optErr = optErr
		return n, nil
	case *DateTime:
		merged, optErr := numberOptionsFrom(NumberOptions{}, opts)
		n := NewNumber(a.toNumber(), merged)
		n.optErr = optErr
		return n, nil
	}

	return nil, newTypeError("Invalid argument to NUMBER")
}

// maxTimeValueMillis is ECMA-262's time value range: ±8.64e15 ms from the
// epoch. Intl.DateTimeFormat throws a RangeError beyond it.
const maxTimeValueMillis = 8.64e15

// builtinDATETIME implements the DATETIME() builtin.
func builtinDATETIME(args []Value, opts map[string]Value) (Value, error) {
	var arg Value
	if len(args) > 0 {
		arg = args[0]
	}

	switch a := arg.(type) {
	case *None:
		return NewNone("DATETIME(" + a.fallback + ")"), nil
	case *DateTime:
		merged, err := dateTimeOptionsFrom(a.Options, opts)
		if err != nil {
			return nil, err
		}
		return NewDateTime(a.Time, merged), nil
	case *Number:
		if math.IsNaN(a.Value) || math.Abs(a.Value) > maxTimeValueMillis {
			return nil, newRangeError("Invalid time value")
		}
		merged, err := dateTimeOptionsFrom(DateTimeOptions{}, opts)
		if err != nil {
			return nil, err
		}
		return NewDateTime(time.UnixMilli(int64(a.Value)).UTC(), merged), nil
	}

	return nil, newTypeError("Invalid argument to DATETIME")
}
