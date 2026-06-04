# Contributing to gofluent

Thanks for your interest in improving gofluent. This document covers the
mechanics; [ARCHITECTURE.md](ARCHITECTURE.md) explains how the codebase is
organized and why.

By participating you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## Requirements

- Go **1.23** or newer.
- Docker — only for regenerating CLDR data. You do **not** need it for normal
  development; the generated tables and fixtures are committed.

## Building and testing

```sh
go test ./...           # the whole suite (pure Go, zero runtime deps)
go test -race ./...      # recommended for concurrency-sensitive changes
make test                # same as go test ./...
```

Before sending a change, make sure these are clean:

```sh
gofmt -l .               # must print nothing
go vet ./...
staticcheck ./...        # https://staticcheck.dev
```

CI runs the same checks — plus `govulncheck` — on the floor Go version and the
latest stable release.

## Conventions

### Match the reference implementation

gofluent follows fluent.js and ECMA-402 `Intl.*`. When you change parsing or
formatting behavior it must still agree with the reference. For formatting,
**regenerate fixtures with `make gen`** instead of hand-editing expected values:
if a golden value looks wrong, the fix usually belongs in the generator, not in
the fixture.

### Tests

The suite is black-box by discipline (follow the surrounding tests for the
pattern):

- External `_test` packages — exercise the **exported API only**. No white-box
  access, and no test-only seams in production code.
- testify: `require` for preconditions, `assert` for checks.
- Table-driven with `t.Run` subtests.

(`internal/conformance` stays `package conformance`: it has no base package and
is already black-box against `syntax`.)

### Generated code

Files named `tables_gen.go` and everything under `cldr/*/testdata/` are
**generated** — do not edit them by hand. Regenerate with:

```sh
make gen
```

This builds the pinned image in `gen/` (a digest-pinned Node image → a fixed ICU
→ a fixed CLDR release, plus the Go toolchain) and runs `go generate ./cldr/...`
followed by the tests inside it. Because both the Go generators and the Node
`Intl.*` fixture dumps see one CLDR release, the tables and fixtures stay
consistent. To move CLDR versions, bump the Node image and the `cldr-*` versions
together in `gen/`, then re-run `make gen`. Never run the generators or their
Node scripts on the host.

## Pull requests

- Keep each PR focused on a single concern.
- Update documentation and `CHANGELOG.md` for user-facing changes.
- Ensure `gofmt`, `go vet`, `staticcheck`, and the tests pass. The PR template
  checklist mirrors this.

## Licensing of contributions

gofluent is licensed under [Apache-2.0](LICENSE). Per section 5 of that license,
any contribution you intentionally submit for inclusion is provided under the
same terms, with no additional conditions. Please submit only work you have the
right to license this way.

## A note on provenance

This codebase was generated with the assistance of large language models and is
held to the verification suite described in [ARCHITECTURE.md](ARCHITECTURE.md).
Contributions — human or tool-assisted — are held to the same bar: they must keep
the conformance and `Intl` checks green.
