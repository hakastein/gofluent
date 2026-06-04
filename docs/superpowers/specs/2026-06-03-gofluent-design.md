# gofluent — Project Fluent for Go (design)

Date: 2026-06-03
Module: `github.com/hakastein/gofluent`
Status: approved, in implementation

## Goal

A production-grade, spec-complete implementation of [Project Fluent](https://projectfluent.org)
for Go. It must be usable in company services **and** serve as a reference implementation,
verified against the upstream conformance fixtures.

The implementation is a **port of fluent.js** (`@fluent/syntax` + `@fluent/bundle`) to idiomatic
Go. fluent.js is chosen as the port source because the agreed runtime architecture (two parsers)
and the fallback-placeholder convention both match fluent.js exactly. Reference checkouts live in
`/.reference/` (gitignored): `fluent.js` (port source) and `fluent-spec` (`projectfluent/fluent`,
conformance fixtures).

## Decisions

| Topic | Decision |
| --- | --- |
| Goal | Production lib + reference-complete |
| Locale formatting | Pluggable interfaces in core; `x/text` adapter in a separate subpackage |
| API style | Hybrid: idiomatic Go names, fluent.js/rs layering & fault-tolerant semantics |
| Runtime | **Variant B** — two parsers (full-AST syntax parser + optimized runtime parser), like fluent.js |
| Fallback layer | In v1 (`localization` package) |
| Conformance | Upstream `structure/` + `reference/` AST fixtures from `projectfluent/fluent` |
| Module path | `github.com/hakastein/gofluent` |
| Missing-ref placeholder | fluent.js convention: `{$var}`, `{message}`, `{-term}`, `{FUNC()}` via `FluentNone` |
| BiDi isolation | `useIsolating` option, default **true** (FSI U+2068 / PDI U+2069) |

## Architecture (packages)

```
github.com/hakastein/gofluent
  /syntax            full AST + parser + serializer  (port of @fluent/syntax)
    /syntax/ast      AST node types (Resource, Message, Term, Pattern, Placeable,
                     SelectExpression, *Reference, *Literal, Comment×3, Junk, Span, Annotation)
    (parser + serializer + errors live in package syntax)
  /                  package fluent — runtime  (port of @fluent/bundle)
                     resource.go  : optimized RUNTIME parser -> compact runtime AST
                     bundle.go    : Bundle (one locale): AddResource, HasMessage, GetMessage,
                                    FormatPattern, AddFunction
                     resolver.go  : fault-tolerant resolver over runtime AST + Scope
                     value.go     : Value, None, String, Number, DateTime (+ wrapper options)
                     scope.go     : Scope (args, errors, dirty/cyclic guard, placeable counter)
                     builtins.go  : NUMBER, DATETIME
                     format.go    : PluralRules / NumberFormatter / DateTimeFormatter interfaces,
                                    NumberOptions, DateTimeOptions; no-op default formatters
                     errors.go    : resolver + bundle error types
  /fluentx           x/text-backed implementation of the formatting interfaces
                     (PluralRules via x/text/feature/plural, numbers via x/text/number+message,
                      dates via stdlib time + x/text). Isolates the CLDR dependency from core.
  /localization      high-level fallback layer: ordered bundles per locale, resource loader,
                     FormatValue/FormatMessage with locale fallback; language negotiation via
                     golang.org/x/text/language
  /internal/conformance  runs upstream structure/ + reference/ fixtures against syntax parser
                         & serializer; ported resolver-behavior tests
```

The runtime resolver lives in the root `fluent` package next to `Bundle` (not a separate package)
to avoid import cycles and keep the runtime cohesive. `fluentx` imports `fluent` (to implement its
interfaces), never the reverse, so the core has **no** `x/text` dependency.

### Two parsers (variant B), mirroring fluent.js

- **Syntax parser** (`/syntax`, port of `@fluent/syntax/parser.js`): produces the full AST with
  spans/comments/junk. Used by tooling and to pass the `structure/` & `reference/` fixtures.
  Comes with a **serializer** (`ast.Resource -> ftl`) and `ParseError` codes (E0001…).
- **Runtime parser** (`/resource.go`, port of `@fluent/bundle/resource.js`): a separate,
  allocation-light parser that compiles FTL directly into a compact runtime representation
  (no spans/comments). This is what `Bundle.AddResource` uses for formatting.

## Formatting interfaces (the pluggable core)

```go
type PluralRules interface {
    Cardinal(t language.Tag, n Number) string // "zero"|"one"|"two"|"few"|"many"|"other"
    Ordinal(t language.Tag, n Number) string
}
type NumberFormatter interface {
    FormatNumber(t language.Tag, n Number, opts NumberOptions) string
}
type DateTimeFormatter interface {
    FormatDateTime(t language.Tag, d DateTime, opts DateTimeOptions) string
}
```

Without formatters the core degrades gracefully: numbers via `strconv`, selectors match by exact
value + the `*` default only. Wiring in `fluentx` enables full CLDR behavior. `NUMBER`/`DATETIME`
builtins produce `Number`/`DateTime` values carrying options; the formatter is invoked when the
value is coerced to a string.

## Data flow

`syntax.Parse(ftl)` → `ast.Resource` (tooling) — and independently —
`Bundle.AddResource(ftl)` runs the runtime parser → compact runtime AST →
`Bundle.FormatPattern(msg.Value, args, &errs)` → resolver walks the runtime pattern with a
`Scope{bundle, args, &errs, formatters, tag}`. Selectors evaluate the selector value; numeric
selectors fall back to the bundle's `PluralRules` category match, then exact match, then `*`.

## Cross-cutting principles

- **Fault-tolerant**: the resolver never panics. Missing variable/message/term/attribute, cyclic
  references (guarded via `Scope`), and the `MAX_PLACEABLES` (100) limit append to `[]error` and
  emit a `FluentNone` fallback rendered as `{$var}` / `{message}` / `{FUNC()}` (fluent.js style).
- **BiDi isolation**: `useIsolating` wraps placeables in FSI/PDI; default true.
- **Concurrency**: `Bundle` is immutable after setup, so `FormatPattern` is safe for concurrent
  use. `AddResource`/`AddFunction` are setup-phase and not concurrency-safe (documented).
  Formatters must be safe for concurrent use.

## Testing strategy (TDD)

- Unit tests per package, table-driven, written before/with implementation.
- `internal/conformance` mounts the upstream `structure/` and `reference/` fixtures
  (FTL + JSON AST) and asserts the syntax parser & serializer match.
- Resolver-behavior tests ported from fluent.js (`@fluent/bundle/test`).
- Definition of done: `go vet ./...` clean and `go test ./...` green.

## Grammar coverage (v1)

Messages, terms, attributes; three comment levels; select expressions; variable / message / term /
attribute references; function references (positional + named args); string & number literals;
placeables; multiline patterns with the FTL indentation/dedent rules; junk recovery. Built-in
functions: `NUMBER`, `DATETIME`.

## Out of scope (later sub-projects)

- Runtime pattern pre-compilation to closures/codegen (variant C) — only if profiling demands it.
- DOM/React-style bindings.
- A CLI / linter on top of the syntax package.

## Addendum (2026-06-04): self-contained CLDR instead of x/text

The first cut implemented the formatting interfaces with a `golang.org/x/text` adapter.
That dependency has been **removed entirely**. The pluggable interfaces stay exactly as
designed, but the CLDR-backed implementations are now generated from CLDR data into the
repository and validated against Node's `Intl.*` (the same engine fluent.js uses), giving
closer fluent.js parity with zero runtime dependencies.

New packages (each stdlib-only, usable standalone):

- `cldr/plural` — cardinal + ordinal rules generated from `cldr-core` (219 / 104 locales).
  100% parity with `Intl.PluralRules` and with CLDR's own `@integer`/`@decimal` samples.
- `cldr/number` — decimal / percent / currency formatting generated from
  `cldr-numbers-full` + `cldr-core` currency data (725 locales). **100% parity** with
  `Intl.NumberFormat` over the fixture matrix.
- `cldr/datetime` — date / time formatting generated from `cldr-dates-full` (725 locales):
  dateStyle/timeStyle from CLDR patterns plus skeleton best-match for component options.
  `timeStyle` 100% vs `Intl.DateTimeFormat`; dateStyle/component ~93–94% (remaining gap is
  structural — non-Gregorian default calendars for `fa`/`th` and skeleton best-match edge
  cases — not a data-version issue).

Each package ships committed `tables_gen.go` plus a build-ignored generator and Node-based
golden-fixture producers. `fluentx` is now a thin adapter mapping the core option structs
onto these packages. The whole module depends only on the Go standard library
(`go.mod` has no `require` block).

### Pinned generation toolchain (2026-06-04)

Generation is **hermetic and version-locked**, not dependent on the host's Node. The CLDR
release is pinned in two places that must agree: the Node image (its bundled ICU fixes the
`Intl.*` golden fixtures) and the `cldr-*` npm packages (the JSON the Go generators read).
Both are CLDR 46 (`node:22.15.0` → ICU 76.1 → CLDR 46.0; `cldr-*@46.0.0`), so generated
tables and Intl fixtures agree by construction. `make gen` builds `gen/Dockerfile` and runs
`go generate ./cldr/...` + tests inside it; the host never runs the generators. This closed
the earlier CLDR-45-vs-host-ICU-46 number divergence (99.8% → 100%). The committed fixtures
make `go test` itself host-independent; only regeneration needs the pinned image.
