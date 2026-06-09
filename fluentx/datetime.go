package fluentx

import (
	"time"

	"github.com/hakastein/gocldr/datetime"
	"github.com/hakastein/gofluent"
)

// DateTimeFormatter is a CLDR-backed implementation of fluent.DateTimeFormatter
// built on the gocldr/datetime package (standard-library time only). It renders
// fully localized dates and times
// from the Unicode CLDR data — month and weekday names, era and day-period
// labels, locale field ordering and numbering systems — matching JavaScript's
// Intl.DateTimeFormat (and therefore fluent.js) for the dateStyle/timeStyle
// styles and the common component options.
type DateTimeFormatter struct{}

// NewDateTimeFormatter constructs a DateTimeFormatter.
func NewDateTimeFormatter() *DateTimeFormatter { return &DateTimeFormatter{} }

var _ fluent.DateTimeFormatter = (*DateTimeFormatter)(nil)

// FormatDateTime renders t for the given locale honoring opts. The timeZone
// option (an IANA name, e.g. "America/New_York") is applied via
// time.LoadLocation by the underlying gocldr/datetime package; an unknown zone
// leaves the time unchanged.
func (DateTimeFormatter) FormatDateTime(locale string, t time.Time, opts fluent.DateTimeOptions) string {
	return datetime.Format(locale, t, datetimeOptions(opts))
}

// datetimeOptions maps the core fluent.DateTimeOptions onto
// gocldr/datetime.Options. The two structs mirror each other field-for-field, so
// this is a straight field-by-field copy.
func datetimeOptions(opts fluent.DateTimeOptions) datetime.Options {
	return datetime.Options{
		Hour12:                 opts.Hour12,
		Weekday:                opts.Weekday,
		Era:                    opts.Era,
		Year:                   opts.Year,
		Month:                  opts.Month,
		Day:                    opts.Day,
		Hour:                   opts.Hour,
		Minute:                 opts.Minute,
		Second:                 opts.Second,
		TimeZoneName:           opts.TimeZoneName,
		DateStyle:              opts.DateStyle,
		TimeStyle:              opts.TimeStyle,
		DayPeriod:              opts.DayPeriod,
		FractionalSecondDigits: opts.FractionalSecondDigits,
		Calendar:               opts.Calendar,
		NumberingSystem:        opts.NumberingSystem,
		TimeZone:               opts.TimeZone,
	}
}
