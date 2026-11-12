# Roadmap

**Status:** M1, M2, the M4 lifecycle, and M5 release engineering are complete. M3 registry content is deferred; publishing and v1.0.0 remain pending.

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

### M2 — Single-target generation _(complete)_

- [x] `internal/resolve`: kinds, limits, variants, requires/conflicts, plan order
- [x] `internal/render`: strict templates, variables, file modes, function map
- [x] `internal/patch`: anchors, markers, JSON/YAML/TOML, dependency adapters, env
- [x] `internal/project`: `.smokeforge.json`, lockfile, file hashes
- [x] `internal/engine`: preflight, staging, atomic commit, dry-run, env, git init
- [x] `new` flags + interactive Charm-style wizard
- [x] Integration test: generate from `testdata/registry` and run `go build ./...`

**Acceptance met:** a single-target project is generated atomically with state,
lockfile, `.env`, optional git initialization, and a buildable Go fixture.

### M3 — Registry content and overlays _(deferred)_

Deferred by decision on 2026-11-12; work continues on unblocked milestones and
returns here after generated-app internals are agreed.

- [ ] Registry repo governance, layout, release process
- [ ] Base templates: `go/chi`, `rust/axum`, `python/fastapi`, `next`,
      `ios/swiftui`, `android/compose`, `desktop/gpui`, `desktop/egui`
- [ ] Overlays: `database:postgres`, `cache:redis`, `queue:rabbitmq`,
      `auth:jwt`, `totp`, `docker`, `ci:github`
- [ ] Contract tests for every overlay×base pair
- [ ] Native build matrix in registry CI

**Acceptance:** every declared stack builds standalone; every feature overlay
applies to every supported base and the result builds.

### M4 — Lifecycle and fullstack _(lifecycle complete)_

- [x] `add` / `remove` with drift detection, backups, automatic rollback
- [x] `doctor` project-state drift checks
- [ ] `install` per-stack idempotent setup
- [ ] Toolchain checks in `doctor`
- [ ] `workspace/fullstack` root base
- [ ] Multi-target monorepo generation (`apps/api`, `apps/web`, `apps/ios`,
      `apps/android`, `apps/desktop`)
- [ ] Client API-URL wiring
- [ ] Root compose/CI aggregation
- [ ] End-to-end matrix for representative fullstack combos

**Acceptance (lifecycle):** generate, add, and remove reversible features;
drifted files block mutation; failures restore backups and leave state unchanged.
Fullstack/install acceptance remains pending.

### M5 — Release _(engineering complete; publishing pending)_

- [x] goreleaser: Linux/macOS/Windows, amd64/arm64
- [x] Tag-triggered draft release workflow with checksums
- [x] Version metadata via ldflags plus `go install` build-info fallback
- [x] `go install`, prebuilt archive, and release documentation
- [ ] Homebrew tap creation and publishing
- [ ] Registry release assets (deferred with M3)
- [ ] v1.0.0 release

**Acceptance (prepared):** snapshot builds produce six archives and checksums;
the version command reports injected commit/date. Actual publishing is pending a
tag, release review, Homebrew tap, and the remaining v1 scope.

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
