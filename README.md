# gofluent

A Go implementation of [Project Fluent](https://projectfluent.org) — a localization
system for natural-sounding translations.

gofluent is a port of the reference JavaScript implementation
([`@fluent/syntax`](https://github.com/projectfluent/fluent.js) and `@fluent/bundle`)
with one deliberate change: locale-aware formatting (plural rules, numbers, dates) is
exposed through **pluggable interfaces** instead of a hard CLDR dependency, so the core
package has **zero external dependencies**. CLDR-backed formatters live in a separate
`fluentx` package built on `golang.org/x/text`.

The syntax parser and serializer are verified against the upstream Project Fluent
conformance fixtures (62/62 structure, 35/36 reference — the one skip matches fluent.js).

## Install

```sh
go get github.com/hakastein/gofluent
```

Requires Go 1.26+.

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/hakastein/gofluent` | Runtime: fast FTL parser, fault-tolerant resolver, `Bundle` (one locale). No external deps. |
| `.../syntax` (+ `.../syntax/ast`) | Full AST, recursive-descent parser, serializer, visitor — for tooling. |
| `.../fluentx` | CLDR `PluralRules` / `NumberFormatter` / `DateTimeFormatter` on `golang.org/x/text`. |
| `.../langneg` | Language negotiation (port of `@fluent/langneg`). |
| `.../localization` | High-level fallback layer over an ordered chain of locale bundles. |

## Quick start

```go
res, _ := fluent.NewResource("hello = Hello, { $name }!")

b := fluent.NewBundle("en")
b.AddResource(res)

msg, _ := b.GetMessage("hello")
var errs []error
out := b.FormatPatternAny(msg.Value, map[string]any{"name": "World"}, &errs)
// out == "Hello, ⁨World⁩!"  (placeable wrapped in bidi isolation marks)
```

The resolver is **fault-tolerant**: it never panics. Missing references and other
problems are appended to `errs` and rendered as fluent.js-style placeholders (e.g.
`{$name}`); a best-effort string is always returned.

By default placeables are wrapped in Unicode bidirectional isolation marks (FSI/PDI).
Disable with `fluent.NewBundle("en", fluent.WithUseIsolating(false))`.

## Locale-aware formatting

Wire CLDR-backed formatters from `fluentx` to get correct plurals, number grouping,
currency, and dates:

```go
import (
    fluent "github.com/hakastein/gofluent"
    "github.com/hakastein/gofluent/fluentx"
)

b := fluent.NewBundle("ru", fluentx.Options()...)
b.AddResource(res) // { $n -> [one] ... [few] ... *[many] ... } now selects correctly
```

## Localization with fallback

```go
import (
    "testing/fstest"
    "github.com/hakastein/gofluent/localization"
)

fsys := fstest.MapFS{
    "en/main.ftl": {Data: []byte("greeting = Hello")},
    "de/main.ftl": {Data: []byte("greeting = Hallo")},
}
loader := localization.FSLoader(fsys, "{locale}/{resource}.ftl")

l10n, _ := localization.NewFromLocales(
    []string{"de-DE"}, []string{"de", "en"}, "en",
    []string{"main"}, loader,
)
val, _ := l10n.FormatValue("greeting", nil) // "Hallo", falling back to "en" if missing
```

## Design

See [`docs/superpowers/specs/2026-06-03-gofluent-design.md`](docs/superpowers/specs/2026-06-03-gofluent-design.md)
for the full design and the decisions behind it.

## License

The port follows the structure of fluent.js (Apache-2.0). Add a LICENSE file before
publishing.
