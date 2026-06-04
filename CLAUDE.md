# CLAUDE.md — gofluent

Guidance for Claude Code and other AI assistants working in this repository. The
durable documentation lives in the standard places — read those first:

- **[ARCHITECTURE.md](ARCHITECTURE.md)** — package layout, the resolver's
  fault-tolerance and error model, BiDi isolation, and why the CLDR tables are
  generated.
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — build/test, linting, the testing
  discipline, and how to regenerate CLDR data with `make gen`.
- **[README.md](README.md)** — what the library is and how to use it.

## Working agreements specific to this repo

- **Provenance is stated, not hidden.** This codebase is LLM-generated. The guard
  against generated-code rot is verification, not trust: the syntax layer is
  checked against the upstream Project Fluent conformance fixtures and the CLDR
  formatters against Node `Intl.*` golden fixtures, all via `go test ./...`. When
  in doubt, trust the tests over the prose.
- **Match fluent.js / `Intl.*`.** Behavior follows the reference implementation.
  When changing formatting, regenerate fixtures with `make gen` rather than
  hand-editing expected values.
- **Never run the generators on the host.** Use `make gen` (pinned Docker
  toolchain). Do not edit `tables_gen.go` or `cldr/*/testdata/` by hand.
- **Tests are black-box.** External `_test` packages, exported API only, testify,
  table-driven via `t.Run`. No test-only seams in production code.
- **Keep checks green.** `gofmt -l .`, `go vet ./...`, `staticcheck ./...`, and
  `go test -race ./...` must all pass before a change is done.
