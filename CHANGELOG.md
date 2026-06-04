# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the project is pre-1.0, the public API may change between minor versions.

## [Unreleased]

### Changed

- Lowered the minimum supported Go version from 1.26 to **1.23**.

### Fixed

- Resolved all `staticcheck` diagnostics across the module.

### Added

- Apache-2.0 `LICENSE` and `NOTICE`.
- Project governance and documentation: `CONTRIBUTING.md`, `ARCHITECTURE.md`,
  `CODE_OF_CONDUCT.md`, `SECURITY.md`, issue and pull-request templates,
  Dependabot configuration, and a hardened CI workflow (vet, build, race tests,
  `gofmt`, `staticcheck`, `govulncheck`).

[Unreleased]: https://github.com/hakastein/gofluent/commits/main
