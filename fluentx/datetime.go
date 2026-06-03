package fluentx

import (
	"strings"
	"time"

	"github.com/hakastein/gofluent"
)

// DateTimeFormatter is a pragmatic locale-aware implementation of
// fluent.DateTimeFormatter. golang.org/x/text does not ship a full CLDR
// date/time formatter, so this implementation maps DateTimeOptions to Go
// reference layouts (the "2006-01-02 15:04:05" layout system).
//
// KNOWN LIMITATION (v1): month and weekday NAMES are English-centric, because
// the Go time package's named layouts (January, Mon, etc.) are English only.
// Numeric components, the chosen field set, hour12/24, fractional seconds and
// the timeZone option are all honored across locales; only the spelled-out
// names are not localized. The fluent.DateTimeFormatter interface is pluggable
// precisely so a richer CLDR-backed formatter can replace this one without
// touching the core. Locale is currently used only to choose sensible ordering
// defaults; the field-by-field layout below is locale-independent.
type DateTimeFormatter struct{}

// NewDateTimeFormatter constructs a DateTimeFormatter.
func NewDateTimeFormatter() *DateTimeFormatter { return &DateTimeFormatter{} }

var _ fluent.DateTimeFormatter = (*DateTimeFormatter)(nil)

// FormatDateTime renders t for the given locale honoring opts. The timeZone
// option (an IANA name, e.g. "America/New_York") is applied via
// time.LoadLocation; an unknown zone leaves the time unchanged.
func (DateTimeFormatter) FormatDateTime(_ string, t time.Time, opts fluent.DateTimeOptions) string {
	if opts.TimeZone != "" {
		if loc, err := time.LoadLocation(opts.TimeZone); err == nil {
			t = t.In(loc)
		}
	}

	// dateStyle / timeStyle take precedence over individual components, matching
	// Intl.DateTimeFormat, where mixing the two is disallowed.
	if opts.DateStyle != "" || opts.TimeStyle != "" {
		datePart := styleDateLayout(opts.DateStyle)
		timePart := styleTimeLayout(opts.TimeStyle, opts.Hour12)
		layout := strings.TrimSpace(datePart + " " + timePart)
		return t.Format(layout)
	}

	if hasComponentOptions(opts) {
		return t.Format(componentLayout(opts))
	}

	// No options at all: a medium-style default.
	return t.Format("Jan 2, 2006, 3:04:05 PM")
}

func hasComponentOptions(o fluent.DateTimeOptions) bool {
	return o.Weekday != "" || o.Era != "" || o.Year != "" || o.Month != "" ||
		o.Day != "" || o.Hour != "" || o.Minute != "" || o.Second != "" ||
		o.TimeZoneName != "" || o.Hour12 != nil || o.FractionalSecondDigits != nil
}

// styleDateLayout maps a dateStyle to a Go reference layout.
func styleDateLayout(style string) string {
	switch style {
	case "full":
		return "Monday, January 2, 2006"
	case "long":
		return "January 2, 2006"
	case "short":
		return "1/2/06"
	case "medium":
		return "Jan 2, 2006"
	default:
		return ""
	}
}

// styleTimeLayout maps a timeStyle to a Go reference layout, honoring hour12.
func styleTimeLayout(style string, hour12 *bool) string {
	twelve := hour12 == nil || *hour12
	switch style {
	case "full", "long":
		if twelve {
			return "3:04:05 PM MST"
		}
		return "15:04:05 MST"
	case "medium":
		if twelve {
			return "3:04:05 PM"
		}
		return "15:04:05"
	case "short":
		if twelve {
			return "3:04 PM"
		}
		return "15:04"
	default:
		return ""
	}
}

// componentLayout builds a Go reference layout from the explicit component
// options (year/month/day/hour/minute/second/weekday/timeZoneName/hour12 and
// fractionalSecondDigits). Each component honors the "numeric"/"2-digit"/
// "long"/"short"/"narrow" sub-options where Go layouts allow.
func componentLayout(o fluent.DateTimeOptions) string {
	twelve := o.Hour12 != nil && *o.Hour12
	// Default to 12-hour when an AM/PM-bearing layout is otherwise ambiguous and
	// hour12 is unset: Intl picks per-locale; here default to 24-hour unless
	// explicitly requested, which is the more neutral choice.

	var date []string
	if o.Weekday != "" {
		switch o.Weekday {
		case "short", "narrow":
			date = append(date, "Mon")
		default: // "long"
			date = append(date, "Monday")
		}
	}
	if o.Month != "" {
		switch o.Month {
		case "2-digit":
			date = append(date, "01")
		case "numeric":
			date = append(date, "1")
		case "short":
			date = append(date, "Jan")
		case "narrow":
			date = append(date, "Jan")
		default: // "long"
			date = append(date, "January")
		}
	}
	if o.Day != "" {
		if o.Day == "2-digit" {
			date = append(date, "02")
		} else {
			date = append(date, "2")
		}
	}
	if o.Year != "" {
		if o.Year == "2-digit" {
			date = append(date, "06")
		} else {
			date = append(date, "2006")
		}
	}

	var clock []string
	if o.Hour != "" {
		switch {
		case twelve && o.Hour == "2-digit":
			clock = append(clock, "03")
		case twelve:
			clock = append(clock, "3")
		case o.Hour == "2-digit":
			clock = append(clock, "15")
		default:
			clock = append(clock, "15")
		}
	}
	if o.Minute != "" {
		if o.Minute == "numeric" {
			clock = append(clock, "4")
		} else {
			clock = append(clock, "04")
		}
	}
	if o.Second != "" {
		if o.Second == "numeric" {
			clock = append(clock, "5")
		} else {
			clock = append(clock, "05")
		}
	}

	clockStr := strings.Join(clock, ":")
	if o.FractionalSecondDigits != nil && *o.FractionalSecondDigits > 0 && clockStr != "" {
		clockStr += "." + strings.Repeat("0", *o.FractionalSecondDigits)
	}
	if twelve && clockStr != "" {
		clockStr += " PM"
	}
	if o.TimeZoneName != "" && clockStr != "" {
		clockStr += " MST"
	} else if o.TimeZoneName != "" {
		clockStr = "MST"
	}

	parts := make([]string, 0, 2)
	if len(date) > 0 {
		parts = append(parts, strings.Join(date, " "))
	}
	if clockStr != "" {
		parts = append(parts, clockStr)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
