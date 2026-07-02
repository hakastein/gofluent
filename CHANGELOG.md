# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the project is pre-1.0, the public API may change between minor versions.

## [Unreleased]

A hardening pass driven by a production-readiness audit: no crash should be
reachable through the public API, even from misbehaving extension points or
pathological input.

### Fixed — robustness

- **Both parsers bound placeable nesting depth (100 levels).** A pathological
  source with enough nested `{` used to exhaust the stack — a fatal,
  unrecoverable crash in Go (fluent.js survives because JS throws a catchable
  `RangeError`). Past the limit the runtime parser skips the entry and the
  syntax parser emits Junk, like any other syntax error.
- **A custom `Function` returning `(nil, nil)` no longer panics.** The nil
  result is reported as a type error and renders the `{NAME()}` fallback,
  honoring the always-return-a-string contract.
- **Panics from injected `WithNumberFormatter`/`WithDateTimeFormatter`/
  `WithPluralRules` implementations are recovered** with the same discipline
  as custom functions: the error is collected, formatting falls back
  (plain rendering for numbers, default CLDR for datetimes, default variant
  for selects), and only genuine `runtime.Error` programming bugs re-panic.
- `syntax`: parsing a large block pattern is linear again; merging adjacent
  text lines was quadratic (an 800k-line value took ~88s, now well under a
  second).
- `ast`: `StringLiteral.Parse` no longer drops the character immediately
  following a `\u`/`\U` escape.
- `Scope.Locale` is nil-receiver-safe and returns `""` without a bundle, so
  custom `Value` implementations can follow the documented "tolerate a nil
  scope" contract without panicking.

### Fixed — error reporting

- A `NUMBER()` carrying an invalid option reports its range error when used
  purely as a select selector; previously the error surfaced only when the
  number was formatted.
- With several invalid `NUMBER()` options the reported error is deterministic
  (options are validated in sorted order, not map order).

### Changed — breaking API

A coordinated pre-1.0 pass toward an idiomatic-Go 1.0 surface, guided by
"idiomatic Go over JS-mirroring". No formatting or parsing behavior changed.

- **`[]error` returns collapse to a single `error` built with `errors.Join`.**
  `Bundle.FormatPattern`, `Bundle.AddResource`, `Bundle.AddResourceOverriding`,
  and `localization`'s `FormatValue`, `FormatMessage`, and `NewFromLocales` now
  return `error`. `if err != nil` and `errors.Is(err, fluent.ErrReference)`
  work directly; the full list stays recoverable via the joined error's
  `Unwrap() []error`. Formatting is still fault-tolerant — a non-nil error
  comes with a usable best-effort string (partial output plus the problems,
  not failure).
- **`Message` is opaque.** Its fields are unexported; reach its state through
  `ID()`, `Value()` (nil when the message has no value), `Attribute(name)`, and
  `AttributeNames()` (a sorted copy). A caller can no longer race a Bundle by
  mutating the attributes map it shares across FormatPattern calls.
- **Closed-enum options are typed.** `NumberOptions` and `DateTimeOptions`
  fields with a fixed value set use named string types — `NumberStyle`,
  `CurrencyDisplay`, `UnitDisplay`, `PluralType`, `DateTimeStyle`, `TextWidth`,
  `NumericStyle`, `MonthStyle`, `TimeZoneNameStyle` — with exported constants
  (`StyleCurrency`, `Ordinal`, `MonthShort`, `WidthLong`, ...), so a mistyped
  option no longer compiles. Genuinely open-ended values (currency, unit,
  timeZone, calendar, numberingSystem) stay plain strings.
- **`fluent.Bool` and `fluent.Int`** are exported pointer helpers for the
  pointer-typed option fields (precedent: `aws.Bool`/`aws.Int`).
- **`localization.Message` → `localization.FormattedMessage`**, disambiguating
  the formatted result from the compiled `fluent.Message`.
- **`localization.New` is variadic**: `New(b)` for one bundle,
  `New(bundles...)` for a slice. `localization` is the official high-level
  entry point even for a single bundle; `Bundle.FormatPattern` stays the
  low-level API.

### Removed

- **`Bundle.AddFunction`.** Runtime functions are constructor-only through
  `WithFunctions`, matching fluent.js (whose `FluentBundle` has no
  `addFunction`). The functions map is immutable after construction and no
  longer needs the bundle mutex.

## [0.5.0] - 2026-06-15

A fluent.js-parity and cleanup pass: backend and frontend rendering the same
FTL file must produce identical output.

### Fixed — fluent.js parity

- **`NUMBER()` and `DATETIME()` honor the fluent.js FTL option allowlists.**
  `style`, `currency` and `unit` (NUMBER) and `timeZone`, `calendar` and
  `numberingSystem` (DATETIME) are now ignored when set from FTL, matching
  fluent.js: a translation cannot change what kind of quantity a value is.
  Set them on a `Number`/`DateTime` argument built in code instead.
- Integer options are validated against their Intl ranges (fraction digits
  0–100, integer/significant digits 1–21, `fractionalSecondDigits` 1–3);
  out-of-range values report a range error and fall back, like
  `Intl.NumberFormat`/`Intl.DateTimeFormat` throwing.
- `DATETIME()` rejects NaN, infinities, and timestamps outside ECMA-262's
  ±8.64e15 ms range with a range error instead of rendering an
  implementation-defined string.
- The runtime parser accepts the same whitespace inside placeables as
  fluent.js (JavaScript `\s`: tabs, NBSP, ...), not just spaces and newlines.
- `langneg` no longer lower-cases the script tail and variant of a locale id,
  matching @fluent/langneg's case handling (affects negotiation winners when
  available locales differ only in variant case).
- `syntax`: comments containing astral characters (emoji) survive parsing
  intact; `\u`/`\U` escapes with a sign (e.g. `\u+123`) stay verbatim; the
  E0025 diagnostic at EOF reports `\undefined` like fluent.js.

### Removed

- `fluent.WithLocales` — the fallback list was stored but never read; real
  fallback lives in the `localization` package.
- `None.Fallback()` — the fallback string is internal to rendering.
- `langneg.Locale`, `langneg.NewLocale` — internal parsing types;
  `Strategy.String()` is also gone.
- `ast.Resource.MarshalJSON` — use `ast.Marshal`.

## [0.4.0] - 2026-06-10

A breaking API cleanup toward an idiomatic-Go 1.0 surface. No formatting or
parsing behavior changed: the conformance fixtures and the behavior tests pass
unmodified (only their call sites were updated).

### Changed — core `fluent`

- **`FormatPattern(pattern, args map[string]any) (string, []error)`** replaces
  the `errs *[]error` out-parameter and `FormatPatternAny`. Formatting is
  always fault-tolerant; the "throw mode" (panic on a nil sink) is gone.
  Argument maps accept raw Go values and `fluent.Value` instances
  interchangeably.
- **`Bundle.Message(id) (*Message, bool)`** replaces `GetMessage`;
  `HasMessage` is removed (use the second return value).
- **`Pattern` is now an opaque, sealed interface.** The runtime AST node types
  (`SelectExpression`, `VariableReference`, `TermReference`,
  `MessageReference`, `FunctionReference`, `StringLiteral`, `NumberLiteral`,
  `NamedArgument`, `Variant`, `ComplexPattern`, `Expression`, `Literal`,
  `PatternElement`, `Term`) are no longer exported.
- **`NewResource(source) *Resource`** — the `[]error` return was always nil by
  design (the runtime parser silently skips broken entries, matching
  fluent.js); use the `syntax` package to diagnose malformed sources.
  `Resource` is now opaque.
- `BundleOption` → `Option`; `FluentString` → `String`.
- `Number` and `DateTime` expose `Value`/`Options` and `Time`/`Options` as
  fields instead of `Value()`/`Opts()` getters; `None.Value()` →
  `None.Fallback()`.
- `Scope` is opaque with a single `Locale()` accessor for custom `Value`
  implementations.
- `MemoizerForLocales` is removed (an unused port artifact).

### Changed — `syntax`

- `FluentParser` / `FluentSerializer` → `Parser` / `Serializer` (constructors
  `NewParser` / `NewSerializer`).
- `ParseEntry` returns `ast.Entry` without the error (it was documented as
  always nil).
- `SerializeExpression` / `SerializeVariantKey` are no longer exported.
- `ast.Comments` → `ast.BaseComment`; `ast.Node` no longer embeds the JSON
  marshaling detail and is documented as a closed interface.
- `Resource.MarshalJSON` omits spans by default; use `ast.Marshal(r, true)` to
  include them.

### Changed — `langneg`

- **`NegotiateLanguages` returns `([]string, error)`**; the panicking variant
  and `NegotiateLanguagesErr` are merged into it. An empty result is nil.
- `FilterMatches`, `Locale.Matches`, `Locale.AddLikelySubtags`,
  `Locale.ClearVariants`/`ClearRegion`, and `Locale.IsEqual` are no longer
  exported (`Locale` values compare with `==`).

### Changed — `localization`

- **`NewFromLocales(Config)`** replaces the seven-positional-parameter
  constructor pair; `Config.Strategy`'s zero value is `langneg.Filtering`.
  A negotiation error now yields a nil `*Localization`.
- **`FormatMessage` returns `(Message, []error)`** with
  `Message{Value, Attributes}`; the `found` bool is replaced by the
  `*NotFoundError` convention `FormatValue` already used.
- `FormatValues` and `L10nID` are removed — batch with a loop.

## [0.3.0] - 2026-06-09

### Changed

- **CLDR formatting is now the default in the core `fluent` package.**
  `NewBundle` wires the gocldr-backed number, date and plural formatters
  automatically, so a bare `fluent.NewBundle(locale, …)` formats locale-aware
  output (matching `Intl.*`) out of the box. The core package now depends
  directly on `github.com/hakastein/gocldr`. Locale data is still opt-in —
  blank-import the locale data you format
  (`import _ "github.com/hakastein/gocldr/locales/en"`, or `.../locales/all`);
  with none imported, numbers fall back to the CLDR root and dates to RFC 3339.

### Removed

- The `fluentx` package (`fluentx.Options`, `fluentx.NewPluralRules`,
  `fluentx.NewNumberFormatter`, `fluentx.NewDateTimeFormatter`). Its adapter now
  lives inside the core `fluent` package as the default formatters. Migrate
  `fluent.NewBundle("xx", fluentx.Options()...)` to `fluent.NewBundle("xx")`;
  to add other options, drop the `append(fluentx.Options(), …)...` wrapper and
  pass them directly, e.g.
  `fluent.NewBundle("xx", fluent.WithUseIsolating(false))`.

## [0.2.0]

### Changed

- CLDR-backed formatting was extracted into the separate
  [`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr) module,
  which gofluent now depends on. The in-repo `cldr/*` packages and the
  Docker-based generation tooling (`gen/`, `make gen`) have been removed; the
  `fluentx` adapter now wires the `gocldr` formatters into a `Bundle`.
- `gocldr` locale data is **opt-in**: an application that formats numbers or
  dates through `fluentx` must blank-import the locale data it needs
  (`import _ "github.com/hakastein/gocldr/locales/en"` for a single locale, or
  `.../locales/all` for every locale). With no locale data imported, formatting
  degrades gracefully (dates render as RFC3339, numbers as ASCII root).
- Lowered the minimum Go version from 1.26 to **1.23** (the 1.26 floor was only
  required by the large generated CLDR tables, which now live in `gocldr`).

## [0.1.0] - 2026-06-09

Initial public release. Requires Go 1.26 or newer.

### Added

- FTL parser and a fault-tolerant resolver with a per-locale `Bundle`. The
  resolver never panics given an error sink; it collects errors and renders
  fluent.js-style placeholders. Placeables are wrapped in Unicode bidi isolation
  marks by default (`WithUseIsolating`).
- `syntax` package (with `syntax/ast`): full AST, recursive-descent parser,
  serializer, and visitor, verified against the upstream Project Fluent
  conformance fixtures.
- CLDR-backed formatting validated against ECMA-402 `Intl.*`, each package usable
  standalone: `cldr/plural` (cardinal and ordinal rules), `cldr/number` (decimal,
  percent, currency), and `cldr/datetime` (date/time, flexible day periods, and
  time-zone names).
- `fluentx` to wire the CLDR formatters into a `Bundle`; `langneg` for language
  negotiation; and `localization` for message fallback across an ordered chain of
  locales.
- Apache-2.0 license and project governance: contributing guide, architecture
  overview, code of conduct, security policy, and a CI pipeline running vet,
  build, race tests, `gofmt`, `staticcheck`, and `govulncheck`.

[Unreleased]: https://github.com/hakastein/gofluent/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/hakastein/gofluent/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/hakastein/gofluent/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/hakastein/gofluent/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/hakastein/gofluent/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/hakastein/gofluent/releases/tag/v0.1.0
