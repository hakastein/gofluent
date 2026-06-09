# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the project is pre-1.0, the public API may change between minor versions.

## [Unreleased]

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

[Unreleased]: https://github.com/hakastein/gofluent/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hakastein/gofluent/releases/tag/v0.1.0
