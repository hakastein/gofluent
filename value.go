package fluent

import (
	"strconv"
	"time"
)

// Value is the base of Fluent's runtime type system. Every expression resolves
// to a Value. Callers convert a Value to its native string with Format.
type Value interface {
	// Format renders this value to a string, optionally using the scope's
	// pluggable formatters. Mirrors FluentType.toString(scope) in fluent.js.
	Format(scope *Scope) string
}

// numberValue is implemented by Number; used by select-expression matching to
// compare numeric selectors without locale formatting.
type numberValue interface {
	numberValue() (float64, NumberOptions)
}

// FluentString is the FluentValue for a plain string (the JS string primitive).
type FluentString string

// Format returns the string unchanged.
func (s FluentString) Format(_ *Scope) string { return string(s) }

// None is a FluentType representing no correct value (FluentNone in fluent.js).
// It renders missing references using the fluent.js fallback convention:
// a missing variable as `{$name}`, a missing message as `{message}`, a missing
// term as `{-term}`, a failed function as `{FUNC()}`. The default fallback is
// `???` which renders as `{???}`.
type None struct {
	value string
}

// NewNone constructs a None with the given fallback inner value.
func NewNone(value string) *None {
	if value == "" {
		value = "???"
	}
	return &None{value: value}
}

// Value returns the raw fallback string (without braces).
func (n *None) Value() string { return n.value }

// Format renders the None as `{value}`.
func (n *None) Format(_ *Scope) string { return "{" + n.value + "}" }

// Number is a FluentType representing a number (FluentNumber in fluent.js).
// It stores the numeric value plus an option bag passed to the NumberFormatter.
type Number struct {
	value float64
	opts  NumberOptions
	// optErr is a deferred option-validation error (e.g. a non-numeric
	// minimumFractionDigits). In fluent.js such errors surface at format time
	// from Intl.NumberFormat; here we mirror that: the error is reported via the
	// scope at Format time and the value falls back to its plain rendering.
	optErr error
}

// NewNumber constructs a Number with the given value and options.
func NewNumber(value float64, opts NumberOptions) *Number {
	return &Number{value: value, opts: opts}
}

// Value returns the wrapped numeric value.
func (n *Number) Value() float64 { return n.value }

// Opts returns the number's formatting options.
func (n *Number) Opts() NumberOptions { return n.opts }

func (n *Number) numberValue() (float64, NumberOptions) { return n.value, n.opts }

// Format renders the number using the bundle's NumberFormatter. A deferred
// option error is reported via the scope and the value falls back to its plain
// decimal rendering, mirroring Intl.NumberFormat throwing in fluent.js.
func (n *Number) Format(scope *Scope) string {
	if n.optErr != nil {
		if scope != nil {
			scope.reportError(n.optErr)
		}
		return strconv.FormatFloat(n.value, 'f', -1, 64)
	}
	if scope != nil {
		return scope.bundle.numberFormatter.FormatNumber(scope.bundle.locale, n.value, n.opts)
	}
	return strconv.FormatFloat(n.value, 'f', -1, 64)
}

// DateTime is a FluentType representing a date/time (FluentDateTime in
// fluent.js). It stores a time.Time plus an option bag passed to the
// DateTimeFormatter.
type DateTime struct {
	value time.Time
	opts  DateTimeOptions
}

// NewDateTime constructs a DateTime with the given time and options.
func NewDateTime(value time.Time, opts DateTimeOptions) *DateTime {
	return &DateTime{value: value, opts: opts}
}

// Value returns the wrapped time.Time.
func (d *DateTime) Value() time.Time { return d.value }

// Opts returns the datetime's formatting options.
func (d *DateTime) Opts() DateTimeOptions { return d.opts }

// toNumber returns the timestamp in milliseconds since the Unix epoch,
// mirroring FluentDateTime.toNumber() in fluent.js.
func (d *DateTime) toNumber() float64 {
	return float64(d.value.UnixMilli())
}

// Format renders the datetime using the bundle's DateTimeFormatter.
func (d *DateTime) Format(scope *Scope) string {
	if scope != nil {
		return scope.bundle.dateTimeFormatter.FormatDateTime(scope.bundle.locale, d.value, d.opts)
	}
	return cldrDateTimeFormatter{}.FormatDateTime("", d.value, d.opts)
}
