// Package fluentx provides locale-aware, CLDR-backed implementations of the
// pluggable formatting interfaces defined in the core fluent package
// (fluent.PluralRules, fluent.NumberFormatter, fluent.DateTimeFormatter).
//
// It is backed by the external github.com/hakastein/gocldr module
// (gocldr/plural, gocldr/number, gocldr/datetime), which is generated directly
// from the Unicode CLDR data. Its output is validated against Node's Intl, so
// fluentx matches JavaScript's Intl.* objects (and therefore fluent.js). There
// is no dependency on golang.org/x/text.
//
// Locale data in gocldr is opt-in: a program links only the locales it imports.
// Applications that format numbers or dates through fluentx MUST import the
// locale data they need, for example:
//
//	import _ "github.com/hakastein/gocldr/locales/en"  // a single locale
//	import _ "github.com/hakastein/gocldr/locales/all" // every locale
//
// With no locale data imported, formatting degrades gracefully (dates render as
// RFC3339, numbers as ASCII root).
//
// The core fluent package is intentionally dependency-free and ships no-op
// defaults; importing fluentx is the opt-in that wires real CLDR behavior. Plug
// the formatters into a bundle either with the Options helper or with the
// individual bundle options:
//
//	b := fluent.NewBundle("ru", fluentx.Options()...)
//
// or
//
//	b := fluent.NewBundle("ru",
//		fluent.WithPluralRules(fluentx.NewPluralRules()),
//		fluent.WithNumberFormatter(fluentx.NewNumberFormatter()),
//		fluent.WithDateTimeFormatter(fluentx.NewDateTimeFormatter()),
//	)
package fluentx

import "github.com/hakastein/gofluent"

// Options returns the bundle options that wire all three fluentx formatters
// (plural rules, number formatter, datetime formatter) into a fluent.Bundle.
//
//	b := fluent.NewBundle("de", fluentx.Options()...)
func Options() []fluent.BundleOption {
	return []fluent.BundleOption{
		fluent.WithPluralRules(NewPluralRules()),
		fluent.WithNumberFormatter(NewNumberFormatter()),
		fluent.WithDateTimeFormatter(NewDateTimeFormatter()),
	}
}
