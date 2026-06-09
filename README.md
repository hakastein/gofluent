# gofluent

[![Go Reference](https://pkg.go.dev/badge/github.com/hakastein/gofluent.svg)](https://pkg.go.dev/github.com/hakastein/gofluent)
[![CI](https://github.com/hakastein/gofluent/actions/workflows/ci.yml/badge.svg)](https://github.com/hakastein/gofluent/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hakastein/gofluent)](https://goreportcard.com/report/github.com/hakastein/gofluent)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A Go implementation of [Project Fluent](https://projectfluent.org) — a
localization system for natural-sounding translations. You write `.ftl` files,
load them into a per-locale `Bundle`, and format messages whose plurals,
numbers, and dates follow the rules of each language. gofluent ports the
reference JavaScript implementation
([`@fluent/syntax`](https://github.com/projectfluent/fluent.js) and
`@fluent/bundle`); the locale-aware formatting is wired in from the
CLDR-backed [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr)
module (validated against Node's `Intl.*`) through the `fluentx` adapter.

> **Status:** pre-1.0. The library is feature-complete and tested against the
> upstream conformance and `Intl.*` suites, but the public API may still change
> between minor versions until 1.0.

## Install

```sh
go get github.com/hakastein/gofluent
```

Requires Go 1.23 or newer.

## Quickstart

This example renders one Russian message across the plural categories Russian
actually uses — **one** (1, 21), **few** (2), **many** (5) — plus a grouped
number and a localized date. It is verified by a runnable example
(`ExampleBundle_pluralRussian` in [`example_test.go`](example_test.go)), so the
output below is exactly what `go test` asserts.

```go
package main

import (
	"fmt"
	"time"

	_ "github.com/hakastein/gocldr/locales/ru" // Russian number + date data (see "Locale data")

	fluent "github.com/hakastein/gofluent"
	"github.com/hakastein/gofluent/fluentx"
)

const src = `
apples =
    { $n ->
        [one] { $n } яблоко
        [few] { $n } яблока
       *[many] { $n } яблок
    }
total = Итого: { NUMBER($total) }
updated = Обновлено { DATETIME($at, dateStyle: "long") }
`

func main() {
	res, _ := fluent.NewResource(src)

	// fluentx.Options() injects the CLDR plural rules, number, and date
	// formatters. useIsolating is disabled here so the output is plain text;
	// the default (true) wraps placeables in Unicode bidi isolation marks.
	b := fluent.NewBundle("ru", append(fluentx.Options(), fluent.WithUseIsolating(false))...)
	b.AddResource(res)

	apples, _ := b.GetMessage("apples")
	for _, n := range []int{1, 2, 5, 21} {
		fmt.Println(b.FormatPatternAny(apples.Value, map[string]any{"n": n}, nil))
	}

	total, _ := b.GetMessage("total")
	fmt.Println(b.FormatPatternAny(total.Value, map[string]any{"total": 1234567}, nil))

	updated, _ := b.GetMessage("updated")
	at := time.Date(2023, 1, 5, 14, 9, 7, 0, time.UTC)
	fmt.Println(b.FormatPatternAny(updated.Value, map[string]any{"at": at}, nil))
}
```

Output:

```text
1 яблоко
2 яблока
5 яблок
21 яблоко
Итого: 1 234 567
Обновлено 5 января 2023 г.
```

The number `1 234 567` is grouped with no-break spaces and the date reads
`5 января 2023 г.` — both Russian conventions, matching `Intl.*`. The plural
select picks `[one]` for 1 and 21, `[few]` for 2, and the default `*[many]` for
5, following CLDR's Russian cardinal rules.

The same shape works for English — `fluent.NewBundle("en", fluentx.Options()...)`
with an FTL whose select uses English's `[one]`/`*[other]` categories.

## How it works

1. A **`Bundle`** holds the translations for **one locale** (`fluent.NewBundle("ru")`).
   The locale string drives every locale-aware decision below.
2. **FTL source** is parsed once with `fluent.NewResource` and added with
   `b.AddResource`. A resource is a set of messages; each message has a value
   (a `Pattern`) and optional `.attributes`.
3. A `{ $n -> [one] … [few] … *[many] … }` **select expression** asks the
   bundle's `PluralRules` for the CLDR plural **category** of `$n` in this
   locale, then renders the matching variant (falling back to the `*` default).
4. **`NUMBER($n)`** and **`DATETIME($d)`** format their argument through the
   bundle's `NumberFormatter` / `DateTimeFormatter`, honoring options such as
   `dateStyle: "long"` or `useGrouping`.
5. `fluentx.Options()` is what supplies those three formatters — it wires the
   CLDR-backed implementations (`fluentx.NewPluralRules()`,
   `NewNumberFormatter()`, `NewDateTimeFormatter()`) into the bundle via
   `fluent.WithPluralRules` / `WithNumberFormatter` / `WithDateTimeFormatter`.

The core `fluent` package is dependency-free and ships **no-op** formatters by
default; without `fluentx` a numeric select matches by exact value and
`NUMBER`/`DATETIME` pass values through untouched. Importing `fluentx` is the
opt-in that turns on real locale behavior.

The resolver is **fault-tolerant**: pass a `*[]error` sink to `FormatPattern` /
`FormatPatternAny` (or `nil` to panic on the first error). In collect mode it
never panics — missing references and other problems are appended to the slice
and rendered as fluent.js-style placeholders (for example `{$name}`), and a
best-effort string is always returned. A `Bundle` is also safe for concurrent
use across all of its read and `Add*` methods.

## Locale data

Plural-category selection (the `[one]`/`[few]`/`[many]` choice) uses CLDR rules
that are **always linked** — Russian plurals are correct with no extra import.

Number and date **formatting** data is **opt-in**: a program links only the
locales it blank-imports. For each locale you format, import its data:

```go
import _ "github.com/hakastein/gocldr/locales/ru" // numbers + dates for ru
import _ "github.com/hakastein/gocldr/locales/en" // numbers + dates for en
```

Each `locales/<lang>` package registers both the number and the date data for
that language. (If you only ever format numbers, `gocldr/number/locales/ru`
alone is enough; for dates, `gocldr/datetime/locales/ru`.) With **no** locale
data imported, formatting degrades gracefully: dates render as RFC3339 and
numbers use the ASCII root (e.g. `1,234,567`), while plural selection still
works.

The `gocldr` formatters are also usable on their own, independent of Fluent
(`gocldr/number`, `gocldr/plural`, `gocldr/datetime`); see that module's docs.

## Loading `.ftl` files

In a real app the FTL lives in files, one directory per locale, loaded through
the `localization` package. `localization.FSLoader` accepts any `fs.FS` —
typically an `embed.FS` (translations compiled into the binary) or
`os.DirFS("./locales")` (read from disk at runtime).

Directory layout (the path template tells the loader where to look):

```text
locales/
  en/
    main.ftl   # apples = { $n -> [one] … *[other] … }
  ru/
    main.ftl   # apples = { $n -> [one] … [few] … *[many] … }
```

```go
import (
	"embed"

	_ "github.com/hakastein/gocldr/locales/en"
	_ "github.com/hakastein/gocldr/locales/ru"

	"github.com/hakastein/gofluent/fluentx"
	"github.com/hakastein/gofluent/localization"
)

//go:embed locales
var localesFS embed.FS

// "{locale}" and "{resource}" are substituted per (locale, resource) pair:
// e.g. "locales/ru/main.ftl".
loader := localization.FSLoader(localesFS, "locales/{locale}/{resource}.ftl")

l10n, _ := localization.NewFromLocales(
	[]string{"ru-RU"},     // requested locales (e.g. from Accept-Language)
	[]string{"ru", "en"},  // locales you ship
	"en",                  // default / ultimate fallback
	[]string{"main"},      // resource ids (file basenames)
	loader,
	fluentx.Options()...,  // forwarded to every per-locale bundle
)

// Walks the negotiated chain (ru, then en) and returns the first match.
val, _ := l10n.FormatValue("apples", map[string]any{"n": 5}) // "5 яблок"
```

`NewFromLocales` negotiates the requested locales against the ones you ship,
builds one `Bundle` per negotiated locale (forwarding the `fluentx` options to
each), and resolves a message from the first bundle in the chain that defines
it. Missing files and parse errors are non-fatal: the failing resource is
skipped and the rest of the chain still works.

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/hakastein/gofluent` | Runtime: fast FTL parser, fault-tolerant resolver, `Bundle` (one locale). |
| `.../syntax` (+ `.../syntax/ast`) | Full AST, recursive-descent parser, serializer, visitor — for tooling. |
| `.../fluentx` | Wires the [`gocldr`](https://github.com/hakastein/gocldr) formatters into a `Bundle` via `fluentx.Options()`. |
| `.../langneg` | Language negotiation (port of `@fluent/langneg`). |
| `.../localization` | High-level fallback layer that loads `.ftl` files and formats across an ordered chain of locale bundles. |

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
