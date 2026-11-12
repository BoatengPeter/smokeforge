# Roadmap

**Status:** Milestone 1 (foundation) is in progress.

## Milestones

### M1 — Foundation _(complete)_

Deliver a working CLI that can load, validate, cache, and inspect a registry.

- [x] Repository scaffold, module, Makefile, CI, docs
- [x] Cobra entrypoint and `version` command
- [x] Manifest schema, strict loading, validation
- [x] Local registry source
- [x] Remote registry tarball: checksum verification, safe extraction, cache
- [x] `registry info`, `registry update`
- [x] `list stacks`, `list features`
- [x] `doctor`
- [x] Unit tests for all of the above; lint clean

**Acceptance:** `smokeforge list stacks --registry <dir>` prints a validated
manifest; a remote registry served by a test server is verified, cached, and
usable offline.

### M2 — Single-target generation

- [ ] `internal/resolve`: kinds, limits, variants, requires/conflicts, plan order
- [ ] `internal/render`: strict templates, variables, file modes, secrets
- [ ] `internal/patch`: anchors, markers, JSON/YAML/TOML, dependency adapters
- [ ] `internal/engine`: preflight, staging, atomic commit, rollback, dry-run
- [ ] `internal/project`: `.smokeforge.json`, lockfile, file hashes
- [ ] `.env` generation, `.gitignore` handling, optional `git init`
- [ ] `new` flags + wizard for single-target kinds
- [ ] Integration test: generate `go/chi`, run `go build ./...`

**Acceptance:** `smokeforge new --kind backend --backend go/chi ...` produces a
buildable project atomically, records state/lock, and fails safely.

### M3 — Registry content and overlays

- [ ] Registry repo governance, layout, release process
- [ ] Base templates: `go/chi`, `rust/axum`, `python/fastapi`, `next`,
      `ios/swiftui`, `android/compose`, `desktop/gpui`, `desktop/egui`
- [ ] Overlays: `database:postgres`, `cache:redis`, `queue:rabbitmq`,
      `auth:jwt`, `totp`, `docker`, `ci:github`
- [ ] Contract tests for every overlay×base pair
- [ ] Native build matrix in registry CI

**Acceptance:** every declared stack builds standalone; every feature overlay
applies to every supported base and the result builds.

### M4 — Lifecycle and fullstack

- [ ] `add` / `remove` with drift detection, backups, automatic rollback
- [ ] `install` per-stack idempotent setup
- [ ] `doctor` project-state and toolchain checks
- [ ] `workspace/fullstack` root base
- [ ] Multi-target monorepo generation (`apps/api`, `apps/web`, `apps/ios`,
      `apps/android`, `apps/desktop`)
- [ ] Client API-URL wiring
- [ ] Root compose/CI aggregation
- [ ] End-to-end matrix for representative fullstack combos

**Acceptance:** generate, add, remove, and install a fullstack project; all
targets build; interrupted mutations recover automatically.

### M5 — Release

- [ ] goreleaser: Linux/macOS/Windows, amd64/arm64
- [ ] `go install` instructions, Homebrew tap
- [ ] Version metadata via ldflags
- [ ] Registry `v1.0.0` tag + checksums
- [ ] User docs for common workflows
- [ ] v1.0.0 release

**Acceptance:** installable from two channels; a documented fullstack example
works end-to-end from a clean machine.

## v1 scope summary

| Area | Included |
| --- | --- |
| Kinds | backend, frontend, mobile, desktop, fullstack |
| Backends | Go (chi), Rust (axum), Python (FastAPI) |
| Frontend | Next.js |
| Mobile | SwiftUI iOS, Jetpack Compose Android |
| Desktop | GPUI, EGUI |
| Features | Postgres, Redis cache, RabbitMQ queue, JWT auth, TOTP, Docker, GitHub Actions CI |
| Lifecycle | new, add, remove, install, list, registry, doctor |

## Pending decisions (blocking M3 templates)

These are deliberately not decided yet and must be discussed before any
generated-app template or overlay is written:

- Backend HTTP frameworks and project layout (Go, Rust, Python).
- Database access: driver vs ORM/query builder, migration tooling.
- Redis client and cache abstraction.
- Queue client and worker model (RabbitMQ).
- JWT/session implementation and TOTP library.
- Native mobile UI approach (SwiftUI vs UIKit; Compose vs Views) and project
  scaffolding tool.
- Desktop UI framework selection (GPUI vs EGUI) and app architecture.
- Dependency management and formatting conventions inside generated apps.

Stack IDs in the manifest are provisional until these discussions conclude.
The CLI foundation does not depend on their outcome.

## Out of v1

- SSE helper overlays
- Search (Meilisearch, Elasticsearch)
- Storage (MinIO, imgproxy, nginx) and Cloudflare R2
- MySQL and SQLite database variants
- Redis Streams queue variant
- Observability (structured logs/metrics/tracing presets)
- Third-party registries and template signing
- Three-way template upgrades / `smokeforge update`
- Adding new client targets after creation (`add` of stacks)
- `smokeforge recreate --from-lock`

These are candidates for v1.1+; none are designed yet.

## Guiding constraints

- Do not trade reproducibility for convenience.
- Do not execute template-provided code.
- Prefer composable overlays over preset matrices.
- Keep the CLI single-binary and cross-platform.
- Every new behavior ships with tests and docs.
