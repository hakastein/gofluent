package fluent

import (
	"strconv"
	"strings"
)

// This file ports fluent.js/fluent-bundle/src/builtins.ts: the NUMBER and
// DATETIME builtins. Each builtin takes positional and named Value arguments
// and returns a Value carrying merged formatting options.

// optString unwraps a named option Value to its string form. A FluentString is
// returned verbatim; a Number renders its plain value. Anything else yields
// ("", false). Mirrors opt.valueOf() in fluent.js.
func optString(v Value) (string, bool) {
	switch t := v.(type) {
	case FluentString:
		return string(t), true
	case *Number:
		return strconv.FormatFloat(t.value, 'f', -1, 64), true
	}
	return "", false
}

// optInt unwraps a named option Value to an int. A Number yields its integer
// value; a numeric FluentString is parsed. Non-numeric strings yield a
// RangeError-equivalent (ok == false) so the builtin can mirror Intl's behavior
// of throwing on an invalid option value.
func optInt(v Value) (int, bool) {
	switch t := v.(type) {
	case *Number:
		return int(t.value), true
	case FluentString:
		if n, err := strconv.Atoi(strings.TrimSpace(string(t))); err == nil {
			return n, true
		}
	}
	return 0, false
}

// optBool unwraps a named option Value to a bool.
func optBool(v Value) (bool, bool) {
	switch t := v.(type) {
	case FluentString:
		switch string(t) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	case *Number:
		return t.value != 0, true
	}
	return false, false
}

// numberOptionsFrom merges the allowed named options into a NumberOptions,
// starting from a base set of options. It returns a deferred RangeError when an
// integer-valued option carries a non-numeric value (e.g.
// minimumFractionDigits: "oops"). The NUMBER builtin still succeeds (mirroring
// fluent.js, where the bad value only causes Intl.NumberFormat to throw at
// format time); the returned error is stored on the Number and surfaced when it
// is formatted.
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
		// Other options (e.g. currency, unknown) are recognized for currency
		// above; truly unknown options are ignored, mirroring fluent.js.
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
		return NewNone("NUMBER(" + a.value + ")"), nil
	case *Number:
		merged, optErr := numberOptionsFrom(a.opts, opts)
		n := NewNumber(a.value, merged)
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

// builtinDATETIME implements the DATETIME() builtin.
func builtinDATETIME(args []Value, opts map[string]Value) (Value, error) {
	var arg Value
	if len(args) > 0 {
		arg = args[0]
	}

	switch a := arg.(type) {
	case *None:
		return NewNone("DATETIME(" + a.value + ")"), nil
	case *DateTime:
		merged := dateTimeOptionsFrom(a.opts, opts)
		return NewDateTime(a.value, merged), nil
	case *Number:
		merged := dateTimeOptionsFrom(DateTimeOptions{}, opts)
		return NewDateTime(millisToTime(a.value), merged), nil
	}

	return nil, newTypeError("Invalid argument to DATETIME")
}
