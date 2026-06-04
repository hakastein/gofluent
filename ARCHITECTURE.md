# Architecture

gofluent is a Go port of [Project Fluent](https://projectfluent.org)'s reference
JavaScript implementation
([fluent.js](https://github.com/projectfluent/fluent.js): `@fluent/syntax`,
`@fluent/bundle`, `@fluent/langneg`). Behavior follows fluent.js; locale-aware
formatting is validated against ECMA-402 `Intl.*`.

This document describes how the module is laid out and the design decisions that
are not obvious from the code. For build, test, and contribution mechanics, see
[CONTRIBUTING.md](CONTRIBUTING.md).

## Design goals

1. **Spec fidelity.** The syntax layer matches the upstream Project Fluent
   conformance fixtures; the CLDR formatters match Node's `Intl.*`. Where the
   reference implementation and intuition disagree, the reference wins.
2. **Dependency-free runtime.** Importing gofluent pulls in no third-party
   packages (testify is a test-only dependency). This is a means, not an end —
   see [Why generated CLDR tables](#why-generated-cldr-tables).
3. **Fault tolerance.** Given an error sink, the resolver never panics; it
   collects errors and renders fluent.js-style placeholders so a best-effort
   string is always produced.
4. **Pluggable formatting.** The core depends on small interfaces
   (`PluralRules`, `NumberFormatter`, `DateTimeFormatter`), not on a concrete
   CLDR implementation. The CLDR-backed implementations live in separate,
   independently usable packages and are wired in through `fluentx`.

## Package layout

| Package | Role |
| --- | --- |
| `.` (`fluent`) | Runtime: the optimized FTL parser (`NewResource`), the fault-tolerant resolver, `Bundle` (one locale), builtins (`NUMBER`/`DATETIME`), and the formatting interfaces. |
| `syntax` (+ `syntax/ast`) | Full AST, recursive-descent parser, serializer, and visitor — for tooling and the conformance suite. |
| `cldr/plural` | CLDR cardinal and ordinal plural rules. Usable standalone. |
| `cldr/number` | CLDR number / percent / currency formatting. Usable standalone. |
| `cldr/datetime` | CLDR date / time formatting. Usable standalone. |
| `fluentx` | Thin adapter wiring the `cldr/*` formatters into a `Bundle` (`fluentx.Options()`). |
| `langneg` | Language negotiation (port of `@fluent/langneg`). |
| `localization` | Fallback layer over an ordered chain of locale bundles. |
| `internal/conformance` | Runs the upstream Project Fluent fixtures. |

### Two parsers, on purpose

There are two FTL parsers, and that is deliberate:

- `syntax` parses to a **full AST** with spans, comments, and junk. It is the
  basis for tooling and the conformance suite, and it round-trips through
  `syntax.Serialize`.
- The root `fluent` package has a separate, **optimized runtime parser**
  (`NewResource`) that produces a compact representation holding only what the
  resolver needs. It is faster and allocates less, at the cost of discarding
  information (spans, comments) the runtime does not use.

## The resolver

Formatting a pattern (`Bundle.FormatPattern` / `FormatPatternAny`) walks the
runtime AST and resolves placeables. It is **fault-tolerant**: rather than
returning early or panicking, it appends an error to the caller-supplied sink,
renders a fluent.js-style placeholder for the failed part (`{$var}`, `{-term}`,
`{FUNC()}`), and continues. A best-effort string is always returned.

Errors are classified by sentinel wrapping, mirroring the JS error classes
fluent.js reports:

- `ErrReference` — an unknown message, term, variable, function, or attribute
  (JS `ReferenceError`).
- `ErrRange` — e.g. a selector that matched no variant (JS `RangeError`).
- `ErrType` — a value used in a position its type does not support
  (JS `TypeError`).

Callers classify a failure with `errors.Is(err, fluent.ErrReference)` and so on.

Two safety properties matter:

- **Panic policy.** A panic from a user-supplied function that carries an `error`
  is recovered into the error sink. A genuine runtime fault (for example a
  nil-map write, which panics with a `runtime.Error`) is *not* swallowed: it
  propagates, so a real bug in a custom function surfaces rather than hiding.
- **Expansion bound.** `MaxPlaceables` caps how many placeables a single
  `FormatPattern` call may expand, defending against Billion-Laughs and
  quadratic-blowup style inputs.

## BiDi isolation

By default the resolver wraps each placeable in Unicode bidirectional isolation
marks — U+2068 FIRST STRONG ISOLATE and U+2069 POP DIRECTIONAL ISOLATE — so that
values of one directionality embedded in text of another render correctly. This
matches fluent.js. Disable it per bundle with `WithUseIsolating(false)`; tests
and examples that assert plain output use that option.

Because isolation characters are part of this library's domain, U+2068/U+2069
appear intentionally in source and test data, and the `staticcheck` check ST1018
(which flags Unicode format characters in string literals) is disabled in
`staticcheck.conf`.

## Why generated CLDR tables

The `cldr/*` packages ship Go tables (`tables_gen.go`) generated from CLDR data
rather than delegating to `golang.org/x/text`. This is the one place the project
hand-rolls something a mature dependency already provides, and it is for a
concrete reason: `x/text`'s number and date output **diverges from ECMA-402
`Intl.*`**, which is exactly the behavior fluent.js produces. Matching fluent.js
therefore means producing Intl-validated tables of our own.

"Doesn't fit" here means "exists but not to the required correctness," not "no
package exists." Everywhere else the project prefers mature dependencies — the
generation pipeline itself leans on the `cldr-*` npm packages (CLDR JSON), Node's
`Intl.*` (golden fixtures), and Docker (a pinned toolchain).

### Generation pipeline

`make gen` builds a pinned Docker image (`gen/`) and runs the generators inside
it; they are never run on the host. A single image pins one CLDR release for both
halves of the process:

- The Go generators read the CLDR JSON and emit `tables_gen.go`.
- Node's `Intl.*` dumps the golden fixtures under `testdata/`.

Because both see the same CLDR version, the tables and the fixtures agree by
construction. The committed `tables_gen.go` and `testdata/` keep `go test` itself
host-independent. See [CONTRIBUTING.md](CONTRIBUTING.md) for the mechanics and
CLDR-version bumps.

## Verification

The guard against generated-code rot is verification, not trust:

- **Syntax** is checked against the upstream Project Fluent conformance fixtures
  (`internal/conformance`).
- **Plural rules** are checked against `Intl.PluralRules`.
- **Number formatting** is checked against `Intl.NumberFormat`.
- **Date/time formatting** is checked against `Intl.DateTimeFormat` golden
  fixtures dumped from Node, covering dateStyle/timeStyle, component options,
  flexible day periods (the `dayPeriod` / `B` field), and time-zone names.

All of it runs under `go test ./...`.
