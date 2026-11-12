# CLAUDE.md

This file is the quick operating guide for Claude-based agents. Read
[AGENTS.md](AGENTS.md) for the complete mindmap and
[docs/README.md](docs/README.md) for the documentation index.

## Project

`smokeforge` — a Go 1.25+ CLI that scaffolds backend (Go/Rust/Python),
frontend (Next.js), mobile (SwiftUI/Compose), desktop (GPUI/EGUI), and
fullstack projects from a versioned remote template registry with composable
feature overlays.

Module: `github.com/BoatengPeter/smokeforge`
Status: Milestone 1 (foundation) in progress; see
[docs/development/roadmap.md](docs/development/roadmap.md).

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
make lint
make fmt
go run ./cmd/smokeforge list stacks --registry ./testdata/registry
```

## Where things live

- `internal/cli` — Cobra commands, flags, output.
- `internal/manifest` — schema types, strict loading, `Validate()`.
- `internal/registry` — local/HTTP sources, checksums, XDG cache.
- `internal/resolve|render|patch|engine|project|install` — planned generation
  pipeline; each has an owning doc under `docs/architecture/` or
  `docs/registry/`.
- `docs/architecture` — internals (overview, CLI, engine, state, fullstack).
- `docs/registry` — manifest and overlay authoring contract.
- `docs/development` — setup, testing, roadmap.

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
