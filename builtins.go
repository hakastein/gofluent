package fluent

import (
	"math"
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
		// Only an integral value is a valid integer option. A fractional value
		// (e.g. 2.9) is out of spec and must fail the same way the string "2.9"
		// does, rather than being silently truncated.
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

// numberOptionsFrom merges the allowed named options into a NumberOptions,
// starting from a base set of options. Unknown options are ignored, mirroring
// fluent.js. An integer-valued option carrying a non-numeric value (e.g.
// minimumFractionDigits: "oops") returns a deferred range error: the NUMBER
// builtin still succeeds and the error surfaces at format time, mirroring
// Intl.NumberFormat throwing in its constructor-deferred way.
func numberOptionsFrom(base NumberOptions, opts map[string]Value) (NumberOptions, error) {
	out := base
	for name, v := range opts {
		switch name {
		case "style":
			if s, ok := optString(v); ok {
				out.Style = s
			}
		case "currency":
			if s, ok := optString(v); ok {
				out.Currency = s
			}
		case "currencyDisplay":
			if s, ok := optString(v); ok {
				out.CurrencyDisplay = s
			}
		case "unit":
			if s, ok := optString(v); ok {
				out.Unit = s
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
			n, ok := optInt(v)
			if !ok {
				return out, newRangeError("Invalid option value for minimumIntegerDigits")
			}
			out.MinimumIntegerDigits = intPtr(n)
		case "minimumFractionDigits":
			n, ok := optInt(v)
			if !ok {
				return out, newRangeError("Invalid option value for minimumFractionDigits")
			}
			out.MinimumFractionDigits = intPtr(n)
		case "maximumFractionDigits":
			n, ok := optInt(v)
			if !ok {
				return out, newRangeError("Invalid option value for maximumFractionDigits")
			}
			out.MaximumFractionDigits = intPtr(n)
		case "minimumSignificantDigits":
			n, ok := optInt(v)
			if !ok {
				return out, newRangeError("Invalid option value for minimumSignificantDigits")
			}
			out.MinimumSignificantDigits = intPtr(n)
		case "maximumSignificantDigits":
			n, ok := optInt(v)
			if !ok {
				return out, newRangeError("Invalid option value for maximumSignificantDigits")
			}
			out.MaximumSignificantDigits = intPtr(n)
		}
	}
	return out, nil
}

// dateTimeOptionsFrom merges the allowed named options into a DateTimeOptions.
func dateTimeOptionsFrom(base DateTimeOptions, opts map[string]Value) DateTimeOptions {
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
		case "timeZone":
			setStr(&out.TimeZone, v)
		case "calendar":
			setStr(&out.Calendar, v)
		case "numberingSystem":
			setStr(&out.NumberingSystem, v)
		case "fractionalSecondDigits":
			if n, ok := optInt(v); ok {
				out.FractionalSecondDigits = intPtr(n)
			}
		}
	}
	return out
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

// millisToTime converts a millisecond Unix timestamp to a time.Time (UTC).
func millisToTime(ms float64) time.Time {
	return time.UnixMilli(int64(ms)).UTC()
}

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
		merged := dateTimeOptionsFrom(a.Options, opts)
		return NewDateTime(a.Time, merged), nil
	case *Number:
		merged := dateTimeOptionsFrom(DateTimeOptions{}, opts)
		return NewDateTime(millisToTime(a.Value), merged), nil
	}

	return nil, newTypeError("Invalid argument to DATETIME")
}
