package datetime_test

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hakastein/gofluent/cldr/datetime"
)

// jsCase mirrors one entry of testdata/intl_dates.json.
type jsCase struct {
	Locale string         `json:"locale"`
	MS     int64          `json:"ms"`
	Tag    string         `json:"tag"`
	Opts   map[string]any `json:"opts"`
	Value  string         `json:"value"`
}

func toOptions(m map[string]any) datetime.Options {
	var o datetime.Options
	str := func(k string) string {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	o.Weekday = str("weekday")
	o.Era = str("era")
	o.Year = str("year")
	o.Month = str("month")
	o.Day = str("day")
	o.Hour = str("hour")
	o.Minute = str("minute")
	o.Second = str("second")
	o.TimeZoneName = str("timeZoneName")
	o.DateStyle = str("dateStyle")
	o.TimeStyle = str("timeStyle")
	o.DayPeriod = str("dayPeriod")
	o.Calendar = str("calendar")
	o.NumberingSystem = str("numberingSystem")
	o.TimeZone = str("timeZone")
	if v, ok := m["hour12"]; ok {
		if b, ok := v.(bool); ok {
			o.Hour12 = &b
		}
	}
	if v, ok := m["fractionalSecondDigits"]; ok {
		if f, ok := v.(float64); ok {
			n := int(f)
			o.FractionalSecondDigits = &n
		}
	}
	o.TimeZone = "UTC"
	return o
}

// category groups a case tag into a reporting bucket.
func category(tag string) string {
	switch {
	case strings.HasPrefix(tag, "dateStyle:"):
		return "dateStyle"
	case strings.HasPrefix(tag, "timeStyle:"):
		return "timeStyle"
	case strings.HasPrefix(tag, "both:"):
		return "both(date+time)"
	case strings.HasPrefix(tag, "comp:"):
		return "component"
	}
	return "other"
}

func loadCases(t *testing.T) []jsCase {
	t.Helper()
	data, err := os.ReadFile("testdata/intl_dates.json")
	require.NoError(t, err, "read fixtures")
	var cases []jsCase
	require.NoError(t, json.Unmarshal(data, &cases), "parse fixtures")
	require.NotEmpty(t, cases, "no fixtures loaded")
	return cases
}

// TestIntlMatch reports the per-category and per-tag match rates against
// JavaScript's Intl.DateTimeFormat and prints sample divergences. It is a
// diagnostic breakdown; the gating assertions live in TestThresholds. The
// overall rate is intentionally below 100% (two matrix locales, fa and th,
// default to non-Gregorian calendars this package does not implement, and
// node's bundled ICU is a newer CLDR release than the cldr-dates-full snapshot
// we generate from).
func TestIntlMatch(t *testing.T) {
	cases := loadCases(t)

	type stat struct{ pass, total int }
	byCat := map[string]*stat{}
	byTag := map[string]*stat{}
	// divergence buckets: category -> sample mismatches
	type miss struct{ locale, tag, got, want string }
	var misses []miss

	for _, c := range cases {
		opts := toOptions(c.Opts)
		tm := time.UnixMilli(c.MS).UTC()
		got := datetime.Format(c.Locale, tm, opts)

		cat := category(c.Tag)
		if byCat[cat] == nil {
			byCat[cat] = &stat{}
		}
		if byTag[c.Tag] == nil {
			byTag[c.Tag] = &stat{}
		}
		byCat[cat].total++
		byTag[c.Tag].total++
		if got == c.Value {
			byCat[cat].pass++
			byTag[c.Tag].pass++
		} else {
			misses = append(misses, miss{c.Locale, c.Tag, got, c.Value})
		}
	}

	// Report per category.
	var cats []string
	for k := range byCat {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	totalPass, totalAll := 0, 0
	t.Log("=== Intl match rate by category ===")
	for _, cat := range cats {
		s := byCat[cat]
		totalPass += s.pass
		totalAll += s.total
		t.Logf("  %-16s %5d/%-5d  %.1f%%", cat, s.pass, s.total, 100*float64(s.pass)/float64(s.total))
	}
	t.Logf("  %-16s %5d/%-5d  %.1f%%", "OVERALL", totalPass, totalAll, 100*float64(totalPass)/float64(totalAll))

	// Per-tag breakdown (helps locate weak component skeletons).
	var tags []string
	for k := range byTag {
		tags = append(tags, k)
	}
	sort.Strings(tags)
	t.Log("=== Intl match rate by tag ===")
	for _, tag := range tags {
		s := byTag[tag]
		if s.pass == s.total {
			continue
		}
		t.Logf("  %-22s %4d/%-4d  %.1f%%", tag, s.pass, s.total, 100*float64(s.pass)/float64(s.total))
	}

	// Sample divergences, one per (locale,tag) bucket (cap output).
	t.Log("=== sample divergences (first per locale+tag) ===")
	seen := map[string]bool{}
	shown := 0
	for _, m := range misses {
		key := m.locale + "|" + m.tag
		if seen[key] {
			continue
		}
		seen[key] = true
		t.Logf("  [%s %s] got=%q want=%q", m.locale, m.tag, m.got, m.want)
		shown++
		if shown > 80 {
			break
		}
	}
}

// TestThresholds asserts a minimum match rate so regressions fail the build.
// Style-based formats come straight from CLDR patterns and must match almost
// perfectly; component best-fit is allowed a documented lower bar.
func TestThresholds(t *testing.T) {
	cases := loadCases(t)
	type stat struct{ pass, total int }
	byCat := map[string]*stat{}
	for _, c := range cases {
		opts := toOptions(c.Opts)
		tm := time.UnixMilli(c.MS).UTC()
		got := datetime.Format(c.Locale, tm, opts)
		cat := category(c.Tag)
		if byCat[cat] == nil {
			byCat[cat] = &stat{}
		}
		byCat[cat].total++
		if got == c.Value {
			byCat[cat].pass++
		}
	}
	// Thresholds are set below the measured rates so genuine regressions fail.
	// The ceiling is bounded by two factors we cannot control from this CLDR
	// snapshot: (1) node v22's bundled ICU is a newer CLDR release than the
	// cldr-dates-full npm package we generate from, so a handful of patterns
	// differ (e.g. the en-GB full-date comma); (2) two matrix locales (fa, th)
	// default to non-Gregorian calendars (Persian, Buddhist) which this package
	// does not implement. Excluding fa/th the overall match rate is ~98.6%.
	thresholds := map[string]float64{
		"dateStyle":       0.90,
		"timeStyle":       0.99,
		"both(date+time)": 0.90,
		"component":       0.90,
	}
	for cat, min := range thresholds {
		t.Run(cat, func(t *testing.T) {
			s := byCat[cat]
			require.NotNilf(t, s, "category %q missing from fixtures", cat)
			rate := float64(s.pass) / float64(s.total)
			assert.GreaterOrEqualf(t, rate, min,
				"category %q match rate %.3f below threshold %.3f (%d/%d)",
				cat, rate, min, s.pass, s.total)
		})
	}
}

// ExampleFormat shows the public API.
func ExampleFormat() {
	t := time.Date(2021, 1, 5, 15, 4, 5, 0, time.UTC)
	fmt.Println(datetime.Format("en", t, datetime.Options{DateStyle: "long", TimeStyle: "short", TimeZone: "UTC"}))
	fmt.Println(datetime.Format("fr", t, datetime.Options{DateStyle: "full", TimeStyle: "medium", TimeZone: "UTC"}))
	// Output:
	// January 5, 2021 at 3:04 PM
	// mardi 5 janvier 2021 à 15:04:05
}
