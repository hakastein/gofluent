// Package fluent is a Go implementation of Project Fluent (https://projectfluent.org),
// a localization system for natural-sounding translations.
//
// It is a port of the reference JavaScript implementation (@fluent/syntax and
// @fluent/bundle) with one deliberate change: locale-aware formatting (plural
// rules, numbers, dates) is exposed through pluggable interfaces instead of a
// hard dependency on a CLDR library. This keeps the core dependency-free.
//
// # Layers
//
//   - Package fluent (this package): the runtime — a fast FTL parser
//     ([NewResource]), a fault-tolerant resolver, and [Bundle] (one locale).
//   - Package fluent/syntax: the full AST, recursive-descent parser, and
//     serializer used by tooling and conformance.
//   - Package fluent/fluentx: CLDR-backed [PluralRules], [NumberFormatter], and
//     [DateTimeFormatter] backed by the module's self-contained cldr packages
//     (no external dependencies). Import it to enable real locale formatting.
//   - Package fluent/langneg: language negotiation (a port of @fluent/langneg).
//   - Package fluent/localization: a high-level layer that formats messages
//     across an ordered chain of locale bundles with fallback.
//
// # Basic use
//
//	res, _ := fluent.NewResource("hello = Hello, { $name }!")
//	b := fluent.NewBundle("en")
//	b.AddResource(res)
//	msg, _ := b.GetMessage("hello")
//	var errs []error
//	out := b.FormatPatternAny(msg.Value, map[string]any{"name": "World"}, &errs)
//
// The resolver is fault-tolerant: it never panics. Missing references and other
// problems are appended to the errors slice and rendered as fluent.js-style
// placeholders (for example {$name}); a best-effort string is always returned.
//
// By default placeables are wrapped in Unicode bidirectional isolation marks
// (FSI/PDI). Disable this with [WithUseIsolating](false).
//
// A [Bundle] is safe for concurrent use: FormatPattern, HasMessage, GetMessage,
// and the Add* methods (AddFunction, AddResource, AddResourceOverriding) may run
// from multiple goroutines at once.
package fluent
