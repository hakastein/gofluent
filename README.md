# gofluent

A Go implementation of [Project Fluent](https://projectfluent.org) — a localization
system for natural-sounding translations.

gofluent is a port of the reference JavaScript implementation
([`@fluent/syntax`](https://github.com/projectfluent/fluent.js) and `@fluent/bundle`).
Locale-aware formatting (plural rules, numbers, dates) is exposed through **pluggable
interfaces**; the CLDR-backed implementations live in separate `cldr/*` packages
**generated from CLDR data** and validated against Node's `Intl.*`, so they match
fluent.js without depending on ICU at runtime.

The **entire module has zero external dependencies** — no `golang.org/x/text`, nothing.
All CLDR data is generated into the repository.

The syntax parser and serializer are verified against the upstream Project Fluent
conformance fixtures (62/62 structure, 35/36 reference — the one skip matches fluent.js).
The CLDR formatters are verified against `Intl.PluralRules` (100% parity),
`Intl.NumberFormat` (~99.8%), and `Intl.DateTimeFormat` (dateStyle/timeStyle from CLDR
patterns; common component options) using golden fixtures dumped from Node.

## Install

```sh
go get github.com/hakastein/gofluent
```

Requires Go 1.26+.

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/hakastein/gofluent` | Runtime: fast FTL parser, fault-tolerant resolver, `Bundle` (one locale). |
| `.../syntax` (+ `.../syntax/ast`) | Full AST, recursive-descent parser, serializer, visitor — for tooling. |
| `.../fluentx` | Wires the `cldr/*` formatters into a `Bundle` via `fluentx.Options()`. |
| `.../cldr/plural` | CLDR cardinal & ordinal plural rules (217 / 103 locales). Usable standalone. |
| `.../cldr/number` | CLDR number / percent / currency formatting (710 locales). Usable standalone. |
| `.../cldr/datetime` | CLDR date / time formatting (710 locales). Usable standalone. |
| `.../langneg` | Language negotiation (port of `@fluent/langneg`). |
| `.../localization` | High-level fallback layer over an ordered chain of locale bundles. |

Every package depends only on the Go standard library.

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

The `cldr/*` packages are also usable on their own, independent of Fluent:

```go
import (
    "github.com/hakastein/gofluent/cldr/number"
    "github.com/hakastein/gofluent/cldr/plural"
)

number.Format("de", 1234.5, number.Options{})            // "1.234,5"
number.Format("en", 1234, number.Options{Style: "currency", Currency: "USD"}) // "$1,234.00"
plural.CardinalFor("ru", 2, 0, 0)                        // plural.Few
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

## Regenerating CLDR data

The `cldr/*` packages ship generated tables (`tables_gen.go`) committed to the repo, so
nothing is fetched at build time. To regenerate against a newer CLDR release, fetch the
CLDR JSON (`npm install cldr-core cldr-numbers-full cldr-dates-full`), point the
`//go:generate` directives at it, and run `go generate ./cldr/...`. The golden-fixture
tests are produced from Node's `Intl.*`, so a working Node install reproduces them.

## Design

See [`docs/superpowers/specs/2026-06-03-gofluent-design.md`](docs/superpowers/specs/2026-06-03-gofluent-design.md)
for the full design and the decisions behind it. CLDR formatting is self-contained
(generated from CLDR data, matched to `Intl.*`) rather than delegated to `golang.org/x/text`.

## License

The port follows the structure of fluent.js (Apache-2.0). Add a LICENSE file before
publishing.
