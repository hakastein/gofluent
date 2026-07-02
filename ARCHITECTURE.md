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
2. **Intl-faithful defaults.** The root `fluent` package wires CLDR-backed
   formatting into every bundle by default, matching ECMA-402 `Intl.*` (and
   therefore fluent.js) out of the box. To do so it depends directly on
   `github.com/hakastein/gocldr`; the `langneg` subpackage remains
   standalone (stdlib-only), and testify is a test-only dependency.
3. **Fault tolerance.** The resolver returns the collected errors and renders
   fluent.js-style placeholders so a best-effort string is always produced;
   only genuine runtime faults propagate (see the panic policy below).
4. **Pluggable formatting.** The core formats against small interfaces
   (`PluralRules`, `NumberFormatter`, `DateTimeFormatter`). The default
   implementations are CLDR-backed — they adapt the external `gocldr` module
   in `format_cldr.go` and are installed by `NewBundle` — but a caller can
   replace any of them per bundle with `WithPluralRules` / `WithNumberFormatter`
   / `WithDateTimeFormatter`.

## Package layout

| Package | Role |
| --- | --- |
| `.` (`fluent`) | Runtime: the optimized FTL parser (`NewResource`), the fault-tolerant resolver, `Bundle` (one locale), builtins (`NUMBER`/`DATETIME`), the formatting interfaces, and their default CLDR-backed implementations (`format_cldr.go`) adapting `gocldr`. |
| `syntax` (+ `syntax/ast`) | Full AST, recursive-descent parser, serializer, and visitor — for tooling and the conformance suite. |
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

Formatting a pattern (`Bundle.FormatPattern`) walks the runtime AST and
resolves placeables. It is **fault-tolerant**: rather than returning early or
panicking, it records an error, renders a fluent.js-style placeholder for the
failed part (`{$var}`, `{-term}`, `{FUNC()}`), and continues. A best-effort
string is always returned alongside the collected errors.

The runtime AST is sealed: `Pattern` is an opaque interface whose
implementations live in the root package, and the resolver dispatches over
typed unions (`expression`, `literal`, `patternElement`) rather than `any`.
Users only ever hold a `Pattern` (from `Message.Value` or an attribute) and
hand it back to `FormatPattern`.

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
  is recovered into the returned errors. A genuine runtime fault (for example a
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

The CLDR tables themselves are **not** in this repository. They live in the
separate module [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr),
which gofluent depends on and which exposes `gocldr/plural`, `gocldr/number`, and
`gocldr/datetime`. The adaptation layer is in this repo: `format_cldr.go` holds
the default `PluralRules` / `NumberFormatter` / `DateTimeFormatter`
implementations, which map the core `fluent.*Options` structs onto the matching
`gocldr` options. `NewBundle` installs them automatically.

That module ships Go tables generated from CLDR data and validated against Node's
`Intl.*` (rather than delegating to `golang.org/x/text`, whose number and date
output diverges from ECMA-402 `Intl.*` — the behavior fluent.js produces). The
generation pipeline, CLDR-version pinning, and the `Intl.*` golden fixtures all
live in `gocldr`; see that module for the mechanics.

CLDR plural-category selection (one/few/many/…) is **always available**: the
plural rule tables are compiled into `gocldr`'s engine independently of the
`locales/*` data packages, so category selection is correct with nothing
blank-imported. Only number/date **rendering** data is **opt-in**: a program
links only the locale packages it blank-imports (`gocldr/locales/<tag>`, a
per-domain `.../locales/all`, or the cross-domain `gocldr/locales/all`). With
nothing imported, rendering degrades gracefully — dates render as RFC 3339,
numbers as the ASCII root locale (grouped decimals) — while plural selection
still works. gofluent's own tests/examples that format import
`gocldr/locales/all`.

## Verification

The guard against generated-code rot is verification, not trust:

- **Syntax** is checked against the upstream Project Fluent conformance fixtures
  (`internal/conformance`), under `go test ./...`.
- **CLDR formatting** (plural rules, number, and date/time) is checked against
  `Intl.PluralRules`, `Intl.NumberFormat`, and `Intl.DateTimeFormat` golden
  fixtures in the `gocldr` module's own test suite.
