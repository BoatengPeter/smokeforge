# smokeforge M5 — Release engineering

> **For agentic workers:** Use superpowers:test-driven-development for version code. Config files are reviewed and validated.

**Status:** engineering complete; publishing and v1.0.0 remain pending (M3 registry content remains deferred).

**Goal:** Make smokeforge installable and releasable: correct version metadata from
ldflags and `go install`, reproducible cross-platform binaries, checksums,
release CI, install docs, and a release checklist.

## Scope

- Version metadata fallback from Go build info for `go install` builds.
- goreleaser v2 configuration for Linux/macOS/Windows on amd64/arm64.
- GitHub Actions release workflow triggered by `v*` tags.
- Install documentation: `go install`, prebuilt archives, Homebrew status.
- CHANGELOG and release/versioning guide.
- Snapshot build verification (`goreleaser release --snapshot --clean`).

## Not in this pass

- Publishing an actual GitHub release or creating a Homebrew tap.
- Tagging `v1.0.0`: the v1 scope still needs M3 registry content and remaining
  M4 features (install, fullstack).
- Registry release assets: deferred with M3.
- License choice: pending owner decision.

## Versioning policy

- Pre-1.0 tags use `v0.x.y`.
- `v1.0.0` is reserved for the completed v1 scope.
- Version, commit, and date come from goreleaser ldflags; `go install` builds
  fall back to module build info.

## Tasks

- [x] Version build-info fallback with tests
- [x] `.goreleaser.yaml` and snapshot build
- [x] `.github/workflows/release.yml`
- [x] Install docs, release guide, CHANGELOG
- [x] README/AGENTS/CLAUDE/roadmap updates
- [x] Verification and commit
