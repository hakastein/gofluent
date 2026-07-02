package fluent

import (
	"strconv"
	"time"
)

// Value is the base of Fluent's runtime type system. Every expression resolves
// to a Value; Format renders it to its final string. Custom argument types
// implement Value directly (mirroring user subclasses of FluentType in
// fluent.js).
type Value interface {
	// Format renders this value to a string. The scope carries the bundle's
	// locale and formatters; it may be nil when formatting outside a
	// resolution (implementations must tolerate that).
	Format(scope *Scope) string
}

// String is the Value for a plain string.
type String string

// Format returns the string unchanged.
func (s String) Format(_ *Scope) string { return string(s) }

// None is the Value representing a missing or invalid value (FluentNone in
// fluent.js). It renders using the fluent.js fallback convention: a missing
// variable as `{$name}`, a missing message as `{message}`, a missing term as
// `{-term}`, a failed function as `{FUNC()}`. The default fallback is `???`,
// rendering as `{???}`.
type None struct {
	fallback string
}

// NewNone constructs a None with the given fallback. An empty fallback
// defaults to "???".
func NewNone(fallback string) *None {
	if fallback == "" {
		fallback = "???"
	}
	return &None{fallback: fallback}
}

// Format renders the None as `{fallback}`.
func (n *None) Format(_ *Scope) string { return "{" + n.fallback + "}" }

// Number is the Value for a number (FluentNumber in fluent.js): a float64
// plus the option bag passed to the NumberFormatter.
type Number struct {
	Value   float64
	Options NumberOptions

	// optErr is a deferred option-validation error (e.g. a non-numeric
	// minimumFractionDigits). Mirroring Intl.NumberFormat, which throws at
	// format time, it is reported via the scope when the number is formatted
	// and the value falls back to its plain rendering.
	optErr error
}

// NewNumber constructs a Number with the given value and options.
func NewNumber(value float64, opts NumberOptions) *Number {
	return &Number{Value: value, Options: opts}
}

// Format renders the number using the bundle's NumberFormatter. A deferred
// option error is reported via the scope and falls back to the plain
// rendering, as does formatting without a scope.
func (n *Number) Format(scope *Scope) string {
	if scope == nil || n.optErr != nil {
		if scope != nil {
			scope.reportError(n.optErr)
		}
		return strconv.FormatFloat(n.Value, 'f', -1, 64)
	}
	return scope.bundle.numberFormatter.FormatNumber(scope.bundle.locale, n.Value, n.Options)
}

// DateTime is the Value for a date/time (FluentDateTime in fluent.js): a
// time.Time plus the option bag passed to the DateTimeFormatter.
type DateTime struct {
	Time    time.Time
	Options DateTimeOptions
}

// NewDateTime constructs a DateTime with the given time and options.
func NewDateTime(t time.Time, opts DateTimeOptions) *DateTime {
	return &DateTime{Time: t, Options: opts}
}

// toNumber returns the timestamp in milliseconds since the Unix epoch,
// mirroring FluentDateTime.toNumber() in fluent.js.
func (d *DateTime) toNumber() float64 {
	return float64(d.Time.UnixMilli())
}

// Format renders the datetime using the bundle's DateTimeFormatter.
func (d *DateTime) Format(scope *Scope) string {
	if scope != nil {
		return scope.bundle.dateTimeFormatter.FormatDateTime(scope.bundle.locale, d.Time, d.Options)
	}
	return cldrDateTimeFormatter{}.FormatDateTime("", d.Time, d.Options)
}
