// Package datetime provides CLDR-based date/time formatting for Go, generated
// directly from the Unicode CLDR data (cldr-dates-full). It has ZERO external
// dependencies (standard-library time only) and is designed to match the
// behaviour of JavaScript's Intl.DateTimeFormat (and therefore fluent.js) for
// the dateStyle/timeStyle styles and the common component options.
//
// The symbol/pattern tables in tables_gen.go are produced by the generator in
// internal/gen. To regenerate them, run:
//
//	go generate ./cldr/datetime/...
//
// Usage:
//
//	out := datetime.Format("en", t, datetime.Options{DateStyle: "long", TimeStyle: "short"})
package datetime

import (
	"strings"
	"time"
)

//go:generate go run ./internal/gen/main.go -dates ../../.reference/cldr-data/node_modules/cldr-dates-full/main -numbers ../../.reference/cldr-data/node_modules/cldr-numbers-full/main -core ../../.reference/cldr-data/node_modules/cldr-core/supplemental -out tables_gen.go

// Options mirrors the subset of Intl.DateTimeFormatOptions that fluent.js uses.
// It intentionally mirrors the field names of fluent.DateTimeOptions in the
// core package (without importing it). Empty strings / nil pointers mean
// "unset".
type Options struct {
	Hour12                 *bool
	Weekday                string // "long" | "short" | "narrow"
	Era                    string // "long" | "short" | "narrow"
	Year                   string // "numeric" | "2-digit"
	Month                  string // "numeric" | "2-digit" | "long" | "short" | "narrow"
	Day                    string // "numeric" | "2-digit"
	Hour                   string // "numeric" | "2-digit"
	Minute                 string // "numeric" | "2-digit"
	Second                 string // "numeric" | "2-digit"
	TimeZoneName           string // "long" | "short"
	DateStyle              string // "full" | "long" | "medium" | "short"
	TimeStyle              string // "full" | "long" | "medium" | "short"
	DayPeriod              string // "narrow" | "short" | "long"
	FractionalSecondDigits *int   // 1..3
	Calendar               string // only "gregory"/"gregorian" supported
	NumberingSystem        string // overrides the locale default
	TimeZone               string // IANA name; defaults to local
}

// localeData holds the resolved CLDR symbol tables and patterns for one locale.
// The concrete values are emitted into tables_gen.go by the generator.
type localeData struct {
	MonthsFormat map[string][]string
	MonthsStand  map[string][]string
	DaysFormat   map[string][]string
	DaysStand    map[string][]string
	QuartersFmt  map[string][]string
	QuartersStd  map[string][]string

	DayPeriodsFmt map[string]map[string]string
	DayPeriodsStd map[string]map[string]string
	Eras          map[string][]string

	DateFormats map[string]string
	TimeFormats map[string]string
	DateTime    map[string]string
	AtTime      map[string]string
	Available   map[string]string

	NumberingSystem string
	Zones           map[string]string
}

// resolveLocale walks the CLDR fallback chain to find a locale present in the
// generated table: exact id -> explicit parentLocale -> truncate trailing
// subtag -> "en" as the ultimate root substitute.
func resolveLocale(locale string) (*localeData, string) {
	cur := normalizeLocale(locale)
	seen := map[string]bool{}
	for cur != "" && cur != "und" && !seen[cur] {
		seen[cur] = true
		if idx, ok := localeIndex[cur]; ok {
			return &localeBlobs[idx], cur
		}
		if p, ok := parentLocaleMap[cur]; ok {
			cur = p
			continue
		}
		if i := strings.LastIndexByte(cur, '-'); i >= 0 {
			cur = cur[:i]
			continue
		}
		break
	}
	// ultimate fallback
	if idx, ok := localeIndex["en"]; ok {
		return &localeBlobs[idx], "en"
	}
	return nil, ""
}

// normalizeLocale lowercases language/region tags into CLDR's '-' form while
// keeping script subtags title-cased the way CLDR keys them (e.g. zh-Hant).
func normalizeLocale(locale string) string {
	locale = strings.ReplaceAll(locale, "_", "-")
	parts := strings.Split(locale, "-")
	for i, p := range parts {
		switch {
		case i == 0:
			parts[i] = strings.ToLower(p)
		case len(p) == 4:
			// script: Titlecase
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		case len(p) == 2:
			// region: uppercase
			parts[i] = strings.ToUpper(p)
		default:
			parts[i] = strings.ToLower(p)
		}
	}
	return strings.Join(parts, "-")
}

// Format renders t for the given locale according to opts, matching
// Intl.DateTimeFormat semantics for dateStyle/timeStyle and common component
// options. The locale is a BCP-47 string; unknown locales fall back through the
// CLDR parent chain to en.
func Format(locale string, t time.Time, opts Options) string {
	ld, resolved := resolveLocale(locale)
	if ld == nil {
		return t.Format(time.RFC3339)
	}

	// Apply timeZone.
	if opts.TimeZone != "" {
		if loc, err := time.LoadLocation(opts.TimeZone); err == nil {
			t = t.In(loc)
		}
	}

	// Numbering system: option overrides locale default.
	ns := opts.NumberingSystem
	if ns == "" {
		ns = ld.NumberingSystem
	}
	digits := numberingSystemDigits[ns]

	ctx := &formatCtx{ld: ld, locale: resolved, digits: digits, opts: opts}

	pattern := ctx.resolvePattern()
	out := ctx.interpret(pattern, t)
	// node/ICU normalizes the narrow no-break space (U+202F) that recent CLDR
	// data places before day periods back to a regular space; mirror that so we
	// match Intl output. The regular no-break space (U+00A0) is preserved.
	return strings.ReplaceAll(out, " ", " ")
}

// resolvePattern selects the CLDR pattern string to interpret based on opts.
func (c *formatCtx) resolvePattern() string {
	o := c.opts
	hasDate := o.DateStyle != ""
	hasTime := o.TimeStyle != ""

	if hasDate || hasTime {
		var datePat, timePat string
		if hasDate {
			datePat = c.ld.DateFormats[o.DateStyle]
		}
		if hasTime {
			timePat = c.ld.TimeFormats[o.TimeStyle]
			timePat = c.applyHourCycle(timePat)
		}
		switch {
		case hasDate && hasTime:
			// Intl combines via dateTimeFormats-atTime, keyed by dateStyle.
			combiner := c.ld.AtTime[o.DateStyle]
			if combiner == "" {
				combiner = c.ld.DateTime[o.DateStyle]
			}
			return combine(combiner, datePat, timePat)
		case hasDate:
			return datePat
		default:
			return timePat
		}
	}

	// ICU renders a "numeric" hour padded to two digits when the caller forces a
	// 24-hour clock onto a locale that defaults to a 12-hour clock (e.g. en with
	// hour12:false -> "09:07"). The reverse (forcing 12-hour onto a 24-hour
	// locale) stays unpadded. Detect that asymmetric case here.
	if o.Hour == "numeric" && o.Hour12 != nil && !*o.Hour12 && c.localeUses12() {
		c.padNumericHour = true
	}

	// Component options -> build skeleton, best-match against availableFormats.
	skel := c.buildSkeleton()
	if skel == "" {
		// No options at all: Intl defaults to numeric y/M/d.
		skel = "yMd"
	}
	pat := c.bestMatch(skel)
	return c.applyHourCycle(pat)
}

// combine substitutes {1}=date and {0}=time into a CLDR combiner pattern,
// honoring single-quoted literals in the combiner.
func combine(combiner, datePat, timePat string) string {
	if combiner == "" {
		return strings.TrimSpace(datePat + " " + timePat)
	}
	var b strings.Builder
	inQuote := false
	runes := []rune(combiner)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\'' {
			// Keep quote markers verbatim so the interpreter treats the literal
			// segment as a literal too.
			inQuote = !inQuote
			b.WriteRune(ch)
			continue
		}
		if !inQuote && ch == '{' && i+2 < len(runes) && runes[i+2] == '}' {
			switch runes[i+1] {
			case '0':
				b.WriteString(timePat)
				i += 2
				continue
			case '1':
				b.WriteString(datePat)
				i += 2
				continue
			}
		}
		b.WriteRune(ch)
	}
	return b.String()
}
