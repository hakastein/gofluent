# CLAUDE.md — gofluent

Guidance for Claude Code and other AI assistants working in this repository. The
durable documentation lives in the standard places — read those first:

- **[ARCHITECTURE.md](ARCHITECTURE.md)** — package layout, the resolver's
  fault-tolerance and error model, BiDi isolation, and how CLDR formatting is
  delegated to the external `github.com/hakastein/gocldr` module.
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — build/test, linting, and the testing
  discipline.
- **[README.md](README.md)** — what the library is and how to use it.

## Working agreements specific to this repo

- **Provenance is stated, not hidden.** This codebase is LLM-generated. The guard
  against generated-code rot is verification, not trust: the syntax layer is
  checked against the upstream Project Fluent conformance fixtures via
  `go test ./...`. When in doubt, trust the tests over the prose.
- **Match fluent.js / `Intl.*`.** Behavior follows the reference implementation.
  Changes to parsing/resolving must keep the conformance fixtures green.
- **CLDR formatting lives in `github.com/hakastein/gocldr`.** CLDR formatting
  comes from the external `github.com/hakastein/gocldr` module. The core
  `fluent` package adapts its formatters in `format_cldr.go` and installs them
  as the `Bundle` defaults (overridable via `WithNumberFormatter`,
  `WithDateTimeFormatter`, `WithPluralRules`). The CLDR tables, their generation
  toolchain, and the `Intl.*` golden fixtures all live in that repo.
  Applications opt into locale data via `gocldr/locales/<tag>` or
  `gocldr/locales/all`.
- **Tests are black-box.** External `_test` packages, exported API only, testify,
  table-driven via `t.Run`. No test-only seams in production code.
- **Keep checks green.** `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, and
  `go test -race ./...` must all pass before a change is done.
