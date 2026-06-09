# Contributing to gofluent

Thanks for your interest in improving gofluent. This document covers the
mechanics; [ARCHITECTURE.md](ARCHITECTURE.md) explains how the codebase is
organized and why.

By participating you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## Requirements

- Go **1.23** or newer.

CLDR formatting comes from the external
[`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr) module
(pulled in as a dependency), so there is nothing to generate in this repository.

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

Or run all of them at once with `make lint`.

CI runs the same checks — plus `govulncheck` — on the floor Go version and the
latest stable release.

## Conventions

### Match the reference implementation

gofluent follows fluent.js and ECMA-402 `Intl.*`. When you change parsing
behavior it must still agree with the reference. CLDR formatting (and its
`Intl.*` golden fixtures) lives in the external
[`github.com/hakastein/gocldr`](https://github.com/hakastein/gocldr) module;
formatting fixes and fixture regeneration belong there, not in this repository.

### Tests

The suite is black-box by discipline (follow the surrounding tests for the
pattern):

- External `_test` packages — exercise the **exported API only**. No white-box
  access, and no test-only seams in production code.
- testify: `require` for preconditions, `assert` for checks.
- Table-driven with `t.Run` subtests.

(`internal/conformance` stays `package conformance`: it has no base package and
is already black-box against `syntax`.)

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
