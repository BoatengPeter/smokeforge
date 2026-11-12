# CLAUDE.md

This file is the quick operating guide for Claude-based agents. Read
[AGENTS.md](AGENTS.md) for the complete mindmap,
[STACKS.md](STACKS.md) for stack versions/defaults,
[docs/architecture.md](docs/architecture.md) for Mermaid diagrams, and
[docs/README.md](docs/README.md) for the documentation index.

Project-scoped Go skills live in this CLI repo under `.claude/skills/` and
`.agents/skills/`. Apply the relevant skill for every Go change: code style,
testing, error handling, security, safety, context, naming, documentation,
design patterns, structs/interfaces, performance, concurrency, lint, benchmark,
troubleshooting, viper, and swagger.

Frontend skills (Next.js, React, Tailwind, Zod, robots, design) are scoped to
generated Next apps from the registry template, not this CLI repo.

## Project

`smokeforge` — a Go 1.26.11+ CLI that scaffolds backend (Go/Rust/Python),
frontend (Next.js), mobile (SwiftUI/Compose), desktop (GPUI/EGUI), and
fullstack projects from a versioned remote template registry with composable
feature overlays.

Module: `github.com/BoatengPeter/smokeforge`
Status: M1, M2, the M4 lifecycle (add/remove, drift, backups), and M5 release
engineering are complete. M3 is in progress, starting with the Next.js template;
publishing remains pending. See [STACKS.md](STACKS.md),
[docs/development/roadmap.md](docs/development/roadmap.md), and
[docs/development/release.md](docs/development/release.md).

## Non-negotiables

1. **TDD:** write the failing test first, watch it fail, then implement the
   minimum. No production code without a failing test.
2. **No template code execution.** Registry content is declarative data only.
3. **Validate before writing.** Never leave a partially generated target.
4. **Atomic creation:** staging directory + rename. **Transactional mutation:**
   backups + automatic rollback for `add`/`remove`.
5. **Checksums and lockfiles** for reproducibility; never bypass verification.
6. **Hermetic tests:** no network, no real user cache. Use `httptest`,
   `t.TempDir()`, local registry fixtures.
7. **Determinism:** same registry pin + selections = identical output, except
   locally generated `.env` secrets.

## Commands

```bash
make build
make test
make test-integration
make lint
make fmt
make vulncheck
make release-check
make release-snapshot
go run ./cmd/smokeforge list stacks --registry ./testdata/registry
```

## Where things live

- `internal/cli` — Cobra commands, flags, output.
- `internal/manifest` — schema types, strict loading, `Validate()`.
- `internal/registry` — local/HTTP sources, checksums, XDG cache.
- `internal/resolve|render|patch|engine|project|install` — planned generation
  pipeline; each has an owning doc under `docs/architecture/` or
  `docs/registry/`.
- `docs/architecture.md` — visual architecture with Mermaid diagrams.
- `docs/architecture` — internals (overview, CLI, engine, state, fullstack).
- `docs/registry` — manifest and overlay authoring contract.
- `docs/development` — setup, testing, release, security, cross-platform, roadmap.
- `SECURITY.md` — reporting and security model.
- `PUBLISHING.md` — manual maintainer release steps.
- `.goreleaser.yaml`, `.github/workflows/release.yml` — release builds.

## Before finishing a change

- [ ] Failing test was written and observed failing.
- [ ] `go test ./...` passes.
- [ ] `golangci-lint run` is clean.
- [ ] Docs updated for any command/flag/manifest/behavior change.
- [ ] No network or user-cache access in unit tests.
- [ ] No secrets committed; `.env` stays gitignored.
- [ ] Commit only if the operator explicitly asked.

## Scope

Generated-app internals (HTTP frameworks, UI approaches, ORMs, Redis/queue
clients, auth libraries, migration tools) are **not decided** — do not choose or
implement them. Stack IDs are provisional. Work on CLI/registry infrastructure
only until those decisions are discussed and recorded.

Do not add SSE, search, storage, MySQL/SQLite, Redis Streams, observability,
third-party registries, or template upgrades without an approved design. Stop
and ask if a change threatens atomicity, reproducibility, or the declarative
no-hooks model.
