# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the project is pre-1.0, the public API may change between minor versions.

## [Unreleased]

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

[Unreleased]: https://github.com/hakastein/gofluent/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/hakastein/gofluent/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/hakastein/gofluent/releases/tag/v0.1.0
