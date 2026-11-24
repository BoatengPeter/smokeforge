# Changelog

All notable changes to smokeforge are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
semantic versioning with pre-1.0 releases using `v0.x.y`.

## [Unreleased]

### Changed

- Desktop next steps: Electron explains that `make dev` hot reloads; egui and
  GPUI now point to `make dev-tools` and the cargo-watch `make dev` loop
  instead of `make run`.

## [0.3.4] - 2026-11-19

### Fixed

- Publish the GPUI release workflow from a normal commit.

## [0.3.3] - 2026-11-19

### Fixed

- Final Go lint in the GPUI next-step switch.

## [0.3.2] - 2026-11-19

### Fixed

- Invalid Go switch expression in GPUI next-step output.

## [0.3.1] - 2026-11-19

### Fixed

- Go lint in fullstack GPUI next-step output.

## [0.3.0] - 2026-11-19

### Added

- `desktop/gpui` stack and `desktop-gpui` preset.
- Fullstack `--desktop gpui` composition.
- GPUI Kit skill and design-guide installation support.
- Pinned GPUI Kit 0.10.0 / GPUI 0.3.12 workflow.

## [0.2.9] - 2026-11-19

### Changed

- Fullstack feature choices now identify whether they belong to the API or the
  web app, so Next.js Zustand/BFF choices are not mistaken for desktop egui
  features.

## [0.2.8] - 2026-11-19

### Fixed

- Presets now declare supported stacks. The Electron desktop preset applies
  only to `desktop/electron`; `desktop-egui` applies only to `desktop/egui`, so
  the preset and feature pickers stay aligned.

## [0.2.7] - 2026-11-19

### Fixed

- The wizard now filters presets by the selected stack. Choosing `desktop/egui`
  no longer offers the Electron desktop preset and selects `desktop-egui`
  automatically.

## [0.2.6] - 2026-11-19

### Fixed

- Go lint issues in fullstack egui resolution and wizard validation.

## [0.2.5] - 2026-11-19

### Added

- Fullstack `--desktop egui` composition without Electron-only overlays.
- Egui next-step guidance and workspace desktop Makefile targets.

## [0.2.4] - 2026-11-19

### Fixed

- The feature picker now rejects selecting two variants of one feature, such as
  Electron auth and OAuth PKCE, instead of failing later with `duplicate feature
  "auth"`.
- The resolver error names both conflicting variants and tells the user to choose
  one.

## [0.2.3] - 2026-11-18

### Added

- Fullstack desktop offline Library sync with idempotent mutations, version
  checks, changefeed cursors, retries, manual conflicts, and per-user outbox
  rows.
- Go and Rust Library sync APIs with changefeeds, tombstones, and idempotency
  records.

## [0.2.2] - 2026-11-17

### Added

- Optional `--desktop electron` target for fullstack projects, producing
  `apps/desktop` alongside `apps/api` and `apps/web`.
- Root desktop wiring: Make targets, README/CONTEXT-MAP sections, env
  documentation, and a conditional desktop CI job.
- Electron main-process API auth with safeStorage refresh tokens and a typed
  preload auth bridge.
- Fullstack desktop SQLite local storage and TanStack Query auth hooks.
- Offline outbox design documentation for future fullstack synchronization.

### Changed

- Fullstack next-step output includes `make desktop-dev` when a desktop target is
  present.
- Fullstack CI setup now honors the selected pnpm/npm/bun package manager for
  web and desktop jobs.

## [0.2.1] - 2026-11-17

### Added

- `desktop` project kind and `desktop/electron` stack for standalone Electron apps.
- Secure TypeScript Electron base with React/Vite, typed preload IPC, sandboxed
  renderer, custom `app://` production protocol, CSP/navigation guards, and
  electron-builder packaging/fuses.
- Desktop `desktop` preset with TanStack Query, Zustand, and SQLite local data.
- Typed atomic JSON settings store for non-sensitive preferences.
- `storage:sqlite` overlay with WAL migrations, validated local-data IPC, and
  native-module rebuild support.
- Optional Playwright Electron E2E and GitHub Actions overlays.
- Selectable pnpm, npm, or bun package manager for desktop projects.
- `electron-best-practices`, `zod`, React, Zustand, Tailwind, and frontend skills
  vendored into generated desktop projects with lock metadata.

### Changed

- `new` help and next-step output now describe and support desktop generation.

## [0.2.0] - 2026-11-15

### Added

- Multi-target fullstack generation for Go/Gin or Rust/Axum APIs plus Next.js.
- `--kind fullstack --backend <stack>` with an optional `--frontend` (default
  `next`), producing `apps/api`, `apps/web`, and a root workspace.
- `workspace/fullstack` root template with `CONTEXT-MAP.md`, Makefile, env docs,
  Docker Compose aggregation, and GitHub Actions aggregation.
- Fullstack registry preset `fullstack-api`: Postgres/JWT API, Next.js BFF,
  TanStack Query, Docker, and CI.
- Next.js BFF auth routes with HttpOnly cookies, refresh rotation, origin
  checks, and Go/Rust response normalization.
- Multi-target engine state, lockfile hashes, and root feature tracking.
- Fullstack tests for resolution, multi-target rendering, CLI flags, BFF token
  helpers, and generated Go/Rust projects.

### Changed

- Root operations now run after target overlays so workspace overlays can
  aggregate generated services and workflows.
- `apiFetch` handles `204 No Content` responses for logout and other empty
  mutations.

## [0.1.5] - 2026-11-14

### Fixed

- Registry cache version markers now use the resolved build version, so
  `go install` builds invalidate stale `latest` caches across upgrades.

## [0.1.4] - 2026-11-14

### Changed

- Backend presets now default to Postgres + JWT. `api-custom` starts from the
  base API so users can choose database, auth, TOTP, Docker, and CI features.
- Removed the `api-minimal`, `api-production`, and frontend `production`
  presets, along with production-labelled preset names. Users can add the
  optional deployment features to the default presets.

## [0.1.3] - 2026-11-13

### Fixed

- Preset/kind mismatch errors now list the presets valid for the selected
  kind.
- Frontend and Go production presets are labelled distinctly in help output.

## [0.1.2] - 2026-11-13

### Fixed

- `latest` registry caches now expire and are invalidated when the CLI version
  changes, so a new binary no longer reuses a stale registry.
- The interactive wizard now uses one form with grouped steps; `Shift+Tab`
  navigates back to change a previous answer.

## [0.1.1] - 2026-11-13

### Added

- Kind-scoped presets with one default preset per project kind.
- Go/Gin backend presets in the `new` help and wizard: `api`, `api-minimal`,
  and `api-production`.
- Tests for kind-scoped preset selection and preset/kind mismatch errors.

### Changed

- Preset validation now rejects unknown kinds and duplicate defaults per kind.
- The wizard filters presets to the selected kind.
- Documentation and stack catalog now cover the Go/Gin backend templates.

## [0.1.6] - 2026-11-15

### Added

- `smokeforge update` command with `--check`, `--version`, `--force`, and
  `--json`; downloads the matching release archive, verifies `checksums.txt`,
  and atomically replaces the binary.

## [0.1.7] - 2026-11-15

### Changed

- Backend presets are now labeled stack-neutrally, so Rust and Go stacks both see
  `Backend API` and `Custom backend API` instead of Go-specific names.

## [0.1.8] - 2026-11-15

### Changed

- The wizard now presents media providers as one dropdown and only shows queue
  compression after the RabbitMQ queue is selected.

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
