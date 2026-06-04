# gofluent

A Go implementation of [Project Fluent](https://projectfluent.org) — a localization
system for natural-sounding translations.

> **This is an LLM-generated codebase** (written with Claude, ported from fluent.js).
> We don't hide that — we lean on it: correctness is pinned to the upstream Project
> Fluent conformance fixtures and to Node's `Intl.*` golden fixtures, both checked in
> CI-style by `go test ./...`. Read the code and the tests, not just the prose.

gofluent is a port of the reference JavaScript implementation
([`@fluent/syntax`](https://github.com/projectfluent/fluent.js) and `@fluent/bundle`).
Locale-aware formatting (plural rules, numbers, dates) is exposed through **pluggable
interfaces**. The CLDR-backed implementations live in separate `cldr/*` packages
**generated from CLDR data** and validated against Node's `Intl.*`.

They are generated rather than delegated to `golang.org/x/text` for a specific reason:
`x/text`'s number/date output diverges from ECMA-402 `Intl.*` (the engine fluent.js uses),
so matching fluent.js means producing our own Intl-validated tables. The generation
pipeline itself leans on existing tooling — the CLDR JSON comes from the `cldr-*` npm
packages and the golden fixtures from Node's `Intl.*`, run in a pinned Docker image.

The syntax parser and serializer are verified against the upstream Project Fluent
conformance fixtures (62/62 structure, 35/36 reference — the one skip matches fluent.js).
The CLDR formatters are verified against `Intl.PluralRules` (100% parity),
`Intl.NumberFormat` (100%), and `Intl.DateTimeFormat` (dateStyle/timeStyle from CLDR
patterns; ~94% on component options) using golden fixtures dumped from Node.

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
nothing is fetched at build time. Regeneration runs in a **pinned Docker toolchain**
(`gen/`) — never on the host — so the generated tables and the golden `Intl.*` fixtures
always describe the **same CLDR release**:

```sh
make gen
```

This builds `gen/Dockerfile` (a digest-pinned `node:22.15.0` → ICU 76 → **CLDR 46**, plus
`cldr-*@46` JSON from `gen/package.json` and the Go toolchain) and runs
`go generate ./cldr/...` followed by the tests inside it. Both the Go generators (reading
the CLDR JSON) and the Node `Intl.*` fixture dumps see one CLDR version, so there is no
host-Node version skew. To move to a newer CLDR, bump the Node image and the `cldr-*`
versions together and re-run `make gen`.

## Contributing

Architecture, conventions, and how to regenerate the CLDR data live in
[`CLAUDE.md`](CLAUDE.md).

## License

The port follows the structure of fluent.js (Apache-2.0). Add a LICENSE file before
publishing.
