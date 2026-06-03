// Package fluentx provides locale-aware, CLDR-backed implementations of the
// pluggable formatting interfaces defined in the core fluent package
// (fluent.PluralRules, fluent.NumberFormatter, fluent.DateTimeFormatter),
// built on golang.org/x/text.
//
// The core fluent package is intentionally dependency-free and ships no-op
// defaults; importing fluentx is the opt-in that pulls in golang.org/x/text and
// wires real CLDR behavior. Plug the formatters into a bundle either with the
// Options helper or with the individual bundle options:
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
