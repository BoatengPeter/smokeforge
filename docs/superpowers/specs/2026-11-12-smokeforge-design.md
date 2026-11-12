# smokeforge — Design

**Date:** 2026-11-12
**Status:** Approved

## Goal

A Go CLI that scaffolds client projects across backend, frontend, mobile, desktop,
and fullstack targets from a versioned remote template registry. Users pick a stack,
then compose features (database, cache, queues, auth, TOTP, Docker, CI) as overlays.

## Decisions

- Multi-language from v1: Go, Rust, Python backends; Next.js web; SwiftUI iOS;
  Jetpack Compose Android; GPUI and EGUI desktop.
- Templates live in a single remote registry repo, fetched and cached locally.
- Registry is versioned by tags; projects pin registry tag + commit + checksums
  in a lockfile for reproducibility.
- Base templates plus composable feature overlays (not full preset per combo).
- Overlays are declarative data only: files, anchor insertions, structured
  patches, dependency edits, env vars. No shell hooks.
- Interactive wizard (Charm `huh`) with full flag parity for CI.
- Generation writes files, initializes git, and creates `.env`. Dependency
  installation is a separate, idempotent `smokeforge install` step.
- Failures are recoverable: staging directory + atomic rename for `new`;
  file-hash drift detection, backups, and automatic rollback for `add`/`remove`.
- `add`/`remove` supported after creation.

## Architecture

```
cmd/smokeforge        entrypoint
internal/cli          cobra commands
internal/tui          interactive wizard
internal/registry     fetch, cache, checksum verification
internal/manifest     schema types + strict validation
internal/resolve      selection -> ordered generation plan
internal/render       strict text/template rendering
internal/patch        anchor ops, JSON/YAML/TOML patches, dependency adapters
internal/engine       staging, apply, rollback
internal/project      .smokeforge.json, lockfile, drift, add/remove
internal/install      per-stack package-manager steps
internal/fsx          atomic writes, path safety
internal/gitx         git init
internal/version      build version
internal/tui          wizard
```

### Generation flow

1. Parse flags; wizard fills gaps (TTY only).
2. Load registry manifest from cache; auto-fetch if missing; `--offline` requires cache.
3. Resolve selections into an ordered plan: base template per target + overlays sorted by `requires`.
4. Preflight: target empty/safe, anchors valid, ops apply cleanly, no conflicts.
5. Render into a staging dir on the same filesystem.
6. Apply patches in deterministic order; verify each op.
7. Write `.smokeforge.json` and `.smokeforge.lock.json`.
8. Generate `.env` from `.env.example`, gitignore it.
9. `git init` by default, no commit.
10. Atomic rename staging -> target. On failure target is untouched.

### Registry

Single GitHub repo with `manifest.yaml`, `templates/`, `overlays/`, `tests/`.
Fetched as a tagged release tarball with a SHA-256 checksum; cached under
`$XDG_CACHE_HOME/smokeforge/registry/<tag-sha>/`. `--registry` and
`SMOKEFORGE_REGISTRY` override the default URL; `--offline` forbids network.

### Manifest model

Registry manifest declares `schemaVersion`, `stacks` (id, kind, template path,
variables), `features` (id, supported kinds/stacks, variants, requires/conflicts),
and per-kind selection limits. Overlays declare `supports`, `requires`, and ops:

- `addFiles` (never silently overwrite)
- `insertBefore` / `insertAfter` at named anchors, wrapped in begin/end markers
- `replaceBlock` (creation only; non-removable)
- `patchJSON` / `patchYAML` / `patchTOML`
- `addDependency` via per-language adapters (edit then re-parse to verify)
- `env` entries

Anchors appear in base files as `// smokeforge:anchor <name>`. Every op must
match exactly once or generation aborts before writing.

### Fullstack

Multiple targets in one monorepo (`apps/api`, `apps/web`, `apps/ios`,
`apps/android`, `apps/desktop`) plus a root workspace base (README, task runner,
`.gitignore`, dev compose, CI). DB/cache/queue/auth overlays target the API;
Docker and CI target the root; clients receive explicit API base-URL config.

### CLI

Commands: `new`, `add`, `remove`, `install`, `list`, `registry update|info|pin`,
`doctor`, `version`. All wizard choices have flags (`--kind`, `--backend`,
`--clients`, `--feature id:variant`, `--module`, `--no-input`, `--no-git`,
`--dry-run`, `--json`).

### Project state

`.smokeforge.json`: choices, targets, applied overlays, per-file post-generation
SHA-256 hashes. `.smokeforge.lock.json`: registry tag, commit, checksums.
`add`/`remove` refuse silently overwriting drifted files; backups live in
`.smokeforge/backups/<timestamp>/` and are restored automatically on failure.

## v1 scope

- Bases: `go/chi`, `rust/axum`, `python/fastapi`, `next`, `ios/swiftui`,
  `android/compose`, `desktop/gpui`, `desktop/egui`, `workspace/fullstack`.
- Features: `database:postgres`, `cache:redis`, `queue:rabbitmq`, `auth:jwt`,
  `totp`, `docker`, `ci:github`.
- Lifecycle: `new`, `add`, `remove`, `install`, `list`, `doctor`, `registry`.
- Testing: unit + contract + golden + native build matrix.

Out of v1: SSE, search, storage, MySQL/SQLite variants, Redis-streams queue,
observability, third-party registries, three-way template upgrades.

## Non-goals

- Running arbitrary template code.
- Becoming a package manager or build tool.
- Managing deployed infrastructure.
