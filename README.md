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
exposed through **pluggable interfaces**; the CLDR-backed implementations come
from the separate [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr)
module (generated from CLDR data and validated against Node's `Intl.*`), wired in
through the `fluentx` adapter.

> **Status:** pre-1.0. The library is feature-complete and tested against the
> upstream conformance and `Intl.*` suites, but the public API may still change
> between minor versions until 1.0.

## Install

```sh
go get github.com/hakastein/gofluent
```

Requires Go 1.23 or newer.

CLDR-backed formatting (plurals, numbers, dates) comes from the
[`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr) module (a
dependency). Its locale data is **opt-in**: an application that formats numbers
or dates must blank-import the locale data it needs —
`import _ "github.com/hakastein/gocldr/locales/en"` for a single locale, or
`import _ "github.com/hakastein/gocldr/locales/all"` for every locale. With no
locale data imported, formatting degrades gracefully (dates render as RFC3339,
numbers as ASCII root).

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/hakastein/gofluent` | Runtime: fast FTL parser, fault-tolerant resolver, `Bundle` (one locale). |
| `.../syntax` (+ `.../syntax/ast`) | Full AST, recursive-descent parser, serializer, visitor — for tooling. |
| `.../fluentx` | Wires the [`gocldr`](https://github.com/hakastein/gocldr) formatters into a `Bundle` via `fluentx.Options()`. |
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
grouping, currency, and dates. The CLDR data is opt-in, so blank-import the
locales you need (here, every locale via `.../locales/all`):

```go
import (
    fluent "github.com/hakastein/gofluent"
    "github.com/hakastein/gofluent/fluentx"

    _ "github.com/hakastein/gocldr/locales/all" // or .../locales/ru for just Russian
)

b := fluent.NewBundle("ru", fluentx.Options()...)
b.AddResource(res) // { $n -> [one] ... [few] ... *[many] ... } selects correctly
```

The underlying [`gocldr`](https://github.com/hakastein/gocldr) formatters are also
usable on their own, independent of Fluent (`gocldr/number`, `gocldr/plural`,
`gocldr/datetime`); see that module's documentation.

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
- The **CLDR formatters** live in
  [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr) and are
  checked there against Node's `Intl.*` (`Intl.PluralRules`, `Intl.NumberFormat`,
  and `Intl.DateTimeFormat` golden fixtures).

Read the code and the tests, not just the prose — [ARCHITECTURE.md](ARCHITECTURE.md)
explains the design and where each guarantee is enforced.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for build,
test, and linting mechanics, and [ARCHITECTURE.md](ARCHITECTURE.md) for how the
codebase is organized and why. By participating you agree to the
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

Licensed under the [Apache License, Version 2.0](LICENSE). See [NOTICE](NOTICE)
for attribution of the fluent.js port lineage and the CLDR data.
