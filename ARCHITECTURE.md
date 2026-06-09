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
   conformance fixtures; the CLDR formatters (in the external `gocldr` module)
   match Node's `Intl.*`. Where the reference implementation and intuition
   disagree, the reference wins.
2. **Dependency-free core.** The root `fluent` package and the runtime resolver
   pull in no third-party packages at all (testify is a test-only dependency).
   Only the `fluentx` adapter brings in a dependency — `github.com/hakastein/gocldr`
   — and only for callers that opt into CLDR formatting.
3. **Fault tolerance.** Given an error sink, the resolver never panics; it
   collects errors and renders fluent.js-style placeholders so a best-effort
   string is always produced.
4. **Pluggable formatting.** The core depends on small interfaces
   (`PluralRules`, `NumberFormatter`, `DateTimeFormatter`), not on a concrete
   CLDR implementation. The CLDR-backed implementations live in the external
   `gocldr` module and are wired in through `fluentx`.

## Package layout

| Package | Role |
| --- | --- |
| `.` (`fluent`) | Runtime: the optimized FTL parser (`NewResource`), the fault-tolerant resolver, `Bundle` (one locale), builtins (`NUMBER`/`DATETIME`), and the formatting interfaces. |
| `syntax` (+ `syntax/ast`) | Full AST, recursive-descent parser, serializer, and visitor — for tooling and the conformance suite. |
| `fluentx` | Thin adapter wiring the external `gocldr` formatters into a `Bundle` (`fluentx.Options()`). |
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

## CLDR formatting lives in `gocldr`

The CLDR-backed formatters are **not** in this repository. They live in the
separate module [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr),
which gofluent depends on and which exposes `gocldr/plural`, `gocldr/number`, and
`gocldr/datetime`. `fluentx` is a thin adapter that maps the core
`fluent.*Options` structs onto the matching `gocldr` options and implements the
`PluralRules` / `NumberFormatter` / `DateTimeFormatter` interfaces.

That module ships Go tables generated from CLDR data and validated against Node's
`Intl.*` (rather than delegating to `golang.org/x/text`, whose number and date
output diverges from ECMA-402 `Intl.*` — the behavior fluent.js produces). The
generation pipeline, CLDR-version pinning, and the `Intl.*` golden fixtures all
live in `gocldr`; see that module for the mechanics.

`gocldr` locale data is **opt-in**: a program links only the locale packages it
blank-imports (`gocldr/locales/<tag>`, a per-domain `.../locales/all`, or the
cross-domain `gocldr/locales/all`). With nothing imported, `fluentx` formatting
degrades gracefully — dates render as RFC3339, numbers as the ASCII root locale.
gofluent's own tests/examples that format import `gocldr/locales/all`.

## Verification

The guard against generated-code rot is verification, not trust:

- **Syntax** is checked against the upstream Project Fluent conformance fixtures
  (`internal/conformance`), under `go test ./...`.
- **CLDR formatting** (plural rules, number, and date/time) is checked against
  `Intl.PluralRules`, `Intl.NumberFormat`, and `Intl.DateTimeFormat` golden
  fixtures in the `gocldr` module's own test suite.
