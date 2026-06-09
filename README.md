# gofluent

[![Go Reference](https://pkg.go.dev/badge/github.com/hakastein/gofluent.svg)](https://pkg.go.dev/github.com/hakastein/gofluent)
[![CI](https://github.com/hakastein/gofluent/actions/workflows/ci.yml/badge.svg)](https://github.com/hakastein/gofluent/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hakastein/gofluent)](https://goreportcard.com/report/github.com/hakastein/gofluent)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A Go implementation of [Project Fluent](https://projectfluent.org) — a
localization system for natural-sounding translations.

gofluent is a port of the reference JavaScript implementation
([`@fluent/syntax`](https://github.com/projectfluent/fluent.js) and
`@fluent/bundle`). Locale-aware formatting (plural rules, numbers, dates) is
exposed through **pluggable interfaces**; the CLDR-backed implementations live in
separate `cldr/*` packages, generated from CLDR data and validated against Node's
`Intl.*`.

> **Status:** pre-1.0. The library is feature-complete and tested against the
> upstream conformance and `Intl.*` suites, but the public API may still change
> between minor versions until 1.0.

## Install

```sh
go get github.com/hakastein/gofluent
```

Requires Go 1.26 or newer.

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

The resolver is **fault-tolerant**: it never panics when given an error sink.
Missing references and other problems are appended to `errs` and rendered as
fluent.js-style placeholders (for example `{$name}`); a best-effort string is
always returned.

By default placeables are wrapped in Unicode bidirectional isolation marks
(FSI/PDI). Disable with `fluent.NewBundle("en", fluent.WithUseIsolating(false))`.

A `Bundle` is safe for concurrent use: `FormatPattern` / `FormatPatternAny`,
`HasMessage`, `GetMessage`, and the `AddResource` / `AddResourceOverriding` /
`AddFunction` mutators may be called from multiple goroutines simultaneously.

## Locale-aware formatting

Wire the CLDR-backed formatters from `fluentx` to get correct plurals, number
grouping, currency, and dates:

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

number.Format("de", 1234.5, number.Options{})                                  // "1.234,5"
number.Format("en", 1234, number.Options{Style: "currency", Currency: "USD"})  // "$1,234.00"
plural.CardinalFor("ru", 2, 0, 0)                                              // plural.Few
```

## Localization with fallback

`FSLoader` accepts any `fs.FS` — typically an `embed.FS` (translations compiled
into the binary) or `os.DirFS("./locales")` (read from disk at runtime):

```go
import (
    "embed"
    "github.com/hakastein/gofluent/localization"
)

// e.g. locales/en/main.ftl ("greeting = Hello"), locales/de/main.ftl ("greeting = Hallo")
//go:embed locales
var locales embed.FS

loader := localization.FSLoader(locales, "locales/{locale}/{resource}.ftl")

l10n, _ := localization.NewFromLocales(
    []string{"de-DE"}, []string{"de", "en"}, "en",
    []string{"main"}, loader,
)
val, _ := l10n.FormatValue("greeting", nil) // "Hallo", falling back to "en" if missing
```

## Provenance & verification

gofluent is generated code — it was ported from fluent.js with the assistance of
large language models — and that is stated plainly, because the project's
credibility rests on verification rather than authorship. Correctness is pinned
to executable references, all run under `go test ./...`:

- The **syntax** parser and serializer are checked against the upstream Project
  Fluent conformance fixtures (62/62 structure, 35/36 reference — the single skip
  matches fluent.js).
- The **CLDR formatters** are checked against Node's `Intl.*`: `Intl.PluralRules`
  (full parity), `Intl.NumberFormat` (full parity), and `Intl.DateTimeFormat`
  via golden fixtures covering dateStyle/timeStyle, component options, flexible
  day periods (the `dayPeriod` option / `B` field — "in the afternoon"), and
  time-zone names (specific `EDT` / `Eastern Daylight Time`, generic
  `ET` / `Eastern Time`, location `United Kingdom Time` / `Sydney Time`, and
  numeric `GMT±HH:mm` offsets).

The one standing gap is the two matrix locales `fa` and `th`, which default to
non-Gregorian calendars (Persian, Buddhist) that this package does not implement;
excluding those, day-period and zone-name resolution match `Intl` essentially
exactly.

Read the code and the tests, not just the prose — [ARCHITECTURE.md](ARCHITECTURE.md)
explains the design and where each guarantee is enforced.

## Regenerating CLDR data

The `cldr/*` packages ship generated tables (`tables_gen.go`) committed to the
repo, so nothing is fetched at build time. Regeneration runs in a **pinned Docker
toolchain** (`gen/`) — never on the host — so the generated tables and the golden
`Intl.*` fixtures always describe the **same CLDR release**:

```sh
make gen
```

This builds `gen/Dockerfile` (a digest-pinned `node:22.15.0` → ICU 76 →
**CLDR 46**, plus `cldr-*@46` JSON from `gen/package.json` and the Go toolchain)
and runs `go generate ./cldr/...` followed by the tests inside it. To move to a
newer CLDR, bump the Node image and the `cldr-*` versions together and re-run
`make gen`. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for build,
test, and linting mechanics, and [ARCHITECTURE.md](ARCHITECTURE.md) for how the
codebase is organized and why. By participating you agree to the
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

Licensed under the [Apache License, Version 2.0](LICENSE). See [NOTICE](NOTICE)
for attribution of the fluent.js port lineage and the CLDR data.
