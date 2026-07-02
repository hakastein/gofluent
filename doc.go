// Package fluent is a Go implementation of Project Fluent (https://projectfluent.org),
// a localization system for natural-sounding translations.
//
// It is a port of the reference JavaScript implementation (@fluent/syntax and
// @fluent/bundle) with one deliberate change: where fluent.js relies on the
// JavaScript Intl.* objects, this port exposes locale-aware formatting (plural
// rules, numbers, dates) through pluggable interfaces. [NewBundle] installs
// CLDR-backed defaults — matching Intl.* — from the external
// github.com/hakastein/gocldr module; callers can override any of them with
// [WithPluralRules], [WithNumberFormatter] and [WithDateTimeFormatter].
//
// # Layers
//
//   - Package fluent (this package): the runtime — a fast FTL parser
//     ([NewResource]), a fault-tolerant resolver, and [Bundle] (one locale).
//     [NewBundle] wires the CLDR-backed number/date/plural formatters by default.
//   - Package fluent/syntax: the full AST, recursive-descent parser, and
//     serializer used by tooling and conformance.
//   - Package fluent/langneg: language negotiation (a port of @fluent/langneg).
//   - Package fluent/localization: a high-level layer that formats messages
//     across an ordered chain of locale bundles with fallback.
//
// Locale-aware formatting is backed by github.com/hakastein/gocldr, whose output
// matches ECMA-402 Intl.*. Its tables are opt-in: blank-import the locale data
// you format, e.g. import _ "github.com/hakastein/gocldr/locales/en" (or
// .../locales/all). With none imported, formatting degrades to the CLDR root.
//
// # Basic use
//
//	res := fluent.NewResource("hello = Hello, { $name }!")
//	b := fluent.NewBundle("en")
//	b.AddResource(res)
//	msg, _ := b.Message("hello")
//	out, err := b.FormatPattern(msg.Value(), map[string]any{"name": "World"})
//
// The resolver is fault-tolerant: missing references and other problems are
// reported in the returned error (joined with errors.Join) and rendered as
// fluent.js-style placeholders (for example {$name}); a best-effort string is
// always returned, even when the error is non-nil.
//
// By default placeables are wrapped in Unicode bidirectional isolation marks
// (FSI/PDI). Disable this with [WithUseIsolating](false).
//
// A [Bundle] is safe for concurrent use: FormatPattern, Message, AddResource,
// and AddResourceOverriding may run from multiple goroutines at once.
package fluent
