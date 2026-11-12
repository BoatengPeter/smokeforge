# Changelog

All notable changes to smokeforge are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
semantic versioning with pre-1.0 releases using `v0.x.y`.

## [Unreleased]

### Added

- Registry loading and validation:
  - strict YAML manifest schema with unknown-field rejection
  - local and remote registry sources
  - SHA-256 verification, safe `os.Root` extraction, and offline cache
  - `registry info/update`, `list`, and `doctor`
- Single-target generation:
  - `smokeforge new` with flags and a Charm-style interactive wizard
  - strict template rendering, path templating, and a variable/function map
  - feature overlays: file additions, marker insertions, JSON/YAML/TOML
    patches, dependency adapters, and env entries
  - atomic staging, `.env`/`.env.example`, optional `git init`
  - `.smokeforge.json` state and `.smokeforge.lock.json` pinning
- Project lifecycle:
  - `smokeforge add` / `smokeforge remove` for reversible overlays
  - drift detection with `--force` for adds and strict removal
  - timestamped backups with automatic restore on failure
  - project-aware `doctor` drift reporting
- Release engineering:
  - version metadata from ldflags with Go module build-info fallback
  - goreleaser config for Linux/macOS/Windows on amd64/arm64
  - tag-triggered draft release workflow with checksums
  - install, release, and registry-release documentation
  - cross-platform CI matrix (Linux/macOS/Windows) and six-target cross-build
  - `govulncheck`, registry URL credential/host validation, and redirect validation
- Neutral `testdata/registry` fixture and integration tests covering
  generate → build → add → build → remove → build.

### Deferred

- M3 registry content: official stack templates and feature overlays.
- `smokeforge install`, fullstack composition, and client API wiring.
- Homebrew tap publishing and the `v1.0.0` release.
- Mutation dry-runs, `--json` output, and arbitrary template upgrades.
