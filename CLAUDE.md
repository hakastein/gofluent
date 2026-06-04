# CLAUDE.md — gofluent

Guidance for Claude Code and human contributors. Project-specific; keep it short and current.

## What this is

A production-grade, spec-complete implementation of [Project Fluent](https://projectfluent.org)
for Go, **ported from fluent.js** (`@fluent/syntax` + `@fluent/bundle`).

**This codebase is LLM-generated** (Claude). That's stated plainly, not hidden. The guard
against generated-code rot is verification, not trust: the syntax layer is checked against the
upstream Project Fluent conformance fixtures, and the CLDR formatters against Node's `Intl.*`
golden fixtures — all via `go test ./...`. When in doubt, trust the tests over the prose.

## Layout

- `fluent` (module root) — runtime: the optimized FTL parser (`NewResource`), the
  fault-tolerant resolver, `Bundle` (one locale), builtins (`NUMBER`/`DATETIME`), and the
  pluggable formatting interfaces (`PluralRules`/`NumberFormatter`/`DateTimeFormatter`).
- `syntax` (+ `syntax/ast`) — full AST, recursive-descent parser, serializer (tooling + conformance).
- `cldr/plural`, `cldr/number`, `cldr/datetime` — self-contained CLDR formatting, **generated**
  from CLDR data. Usable standalone. No external deps.
- `fluentx` — thin adapter wiring the `cldr/*` formatters into a `Bundle` (`fluentx.Options()`).
- `langneg` — language negotiation (port of `@fluent/langneg`).
- `localization` — fallback layer over an ordered chain of locale bundles.
- `internal/conformance` — runs the upstream fixtures.

## Build & test

- `make test` / `go test ./...`. Pure Go. **testify is the only dependency, and it is
  test-only** — it does not appear in the library's runtime dependency graph.
- The runtime library has no external dependencies. This is **not a goal in itself** — see below.

## Dependencies: use them, except where they don't fit

Prefer a mature dependency over hand-rolled code; less code is cheaper to maintain. This applies
to dev/build/generation too — we deliberately use the `cldr-*` npm packages (CLDR JSON), Node's
`Intl.*` (golden fixtures), and Docker (pinned toolchain) rather than reinventing them.

The runtime CLDR formatting is the one exception, and for a concrete reason: `golang.org/x/text`'s
number/date output **diverges from ECMA-402 `Intl.*`**, which is exactly what fluent.js matches.
A mature package that produces the wrong output is still the wrong choice, so we generate our own
Intl-validated tables. "Doesn't fit" here means "exists but not to the required correctness," not
"no package exists."

## Regenerating CLDR data

Run **`make gen`** — never run the generators or their Node scripts on the host. It builds the
pinned image in `gen/` (digest-pinned `node:22.15.0` → ICU 76 → **CLDR 46**, plus `cldr-*@46` and
the Go toolchain) and runs `go generate ./cldr/...` + the tests inside it. Both the Go generators
(reading CLDR JSON) and the Node `Intl.*` fixture dumps see one CLDR release, so tables and
fixtures agree by construction. To move CLDR versions, bump the Node image and the `cldr-*`
versions together in `gen/`, then re-run `make gen`. The committed `tables_gen.go` and `testdata/`
keep `go test` itself host-independent.

## Conventions

- **Tests** (see the `writing-go-tests` discipline): external `_test` packages (black-box, exported
  API only), testify (`require` for preconditions, `assert` for checks), table-driven via `t.Run`,
  test the public contract — no white-box, no test-only seams in production. `internal/conformance`
  stays `package conformance` because it has no base package and is already black-box against `syntax`.
- **Resolver is fault-tolerant**: never panics when given an error sink; collects errors and renders
  fluent.js-style placeholders (`{$var}`, `{-term}`, `{FUNC()}`). Classify failures with
  `errors.Is(err, fluent.ErrReference | fluent.ErrRange | fluent.ErrType)` (mirrors fluent.js's
  ReferenceError/RangeError/TypeError).
- **Match fluent.js**: behavior follows the reference implementation; locale formatting is validated
  against Node `Intl.*`. When changing formatting, regenerate fixtures via `make gen` rather than
  hand-editing expected values.
- BiDi isolation (FSI/PDI) defaults on; disable per-bundle with `WithUseIsolating(false)`.
