# AGENTS.md

Mindmap for any agent working on this repository. Read this first, then follow
the linked documents for depth.

## What this repo is

`smokeforge` is a Go CLI that scaffolds backend, frontend, mobile, desktop, and
fullstack projects for clients from a **versioned remote template registry**.
Users pick stacks and compose features (database, cache, queues, auth, TOTP,
Docker, CI) as declarative overlays.

- **Module:** `github.com/BoatengPeter/smokeforge`
- **Binary:** `smokeforge`
- **Language:** Go 1.26.11+
- **Template registry:** separate repo,
  `github.com/BoatengPeter/smokeforge-registry` (planned)
- **Status:** M1, M2, the M4 lifecycle, and M5 release engineering are complete. M3 is in progress, starting with the Next.js template; publishing is pending.

## Read order (agents)

1. This file.
2. [STACKS.md](STACKS.md) — stacks, versions, defaults, and status.
3. [docs/README.md](docs/README.md) — documentation index and status legend.
4. [docs/architecture.md](docs/architecture.md) — visual architecture and Mermaid diagrams.
5. [docs/architecture/overview.md](docs/architecture/overview.md) — components and invariants.
6. [docs/architecture/cli.md](docs/architecture/cli.md) — command surface.
7. [docs/registry/manifest.md](docs/registry/manifest.md) and
   [docs/registry/overlays.md](docs/registry/overlays.md) — registry contract.
8. [docs/development/setup.md](docs/development/setup.md) and
   [docs/development/testing.md](docs/development/testing.md) — how to work here.
9. [docs/development/release.md](docs/development/release.md) — versioning and publishing.
10. [docs/development/security.md](docs/development/security.md) — threat model and controls.
11. [docs/development/cross-platform.md](docs/development/cross-platform.md) — platform matrix.
12. [docs/development/roadmap.md](docs/development/roadmap.md) — what is next.
13. Design record: [docs/superpowers/specs/2026-11-12-smokeforge-design.md](docs/superpowers/specs/2026-11-12-smokeforge-design.md).
14. Plans: [release](docs/superpowers/plans/2026-11-12-smokeforge-release.md), [lifecycle](docs/superpowers/plans/2026-11-12-smokeforge-lifecycle.md), [generation](docs/superpowers/plans/2026-11-12-smokeforge-generation.md), [foundation](docs/superpowers/plans/2026-11-12-smokeforge-foundation.md).

## Project skills — use them

Project-scoped Go skills are installed in this CLI repository under
`.agents/skills/` (OpenCode and AGENTS-compatible agents) and `.claude/skills/`
(Claude Code). Frontend skills are **not** installed here; they are scoped to
generated Next apps by the registry template
(`smokeforge-registry/templates/frontend/next/files`). Apply the relevant Go
skill whenever writing or reviewing Go code:

`golang-code-style`, `golang-testing`, `golang-error-handling`,
`golang-security`, `golang-safety`, `golang-context`, `golang-naming`,
`golang-documentation`, `golang-design-patterns`, `golang-structs-interfaces`,
`golang-performance`, `golang-concurrency`, `golang-lint`, `golang-benchmark`,
`golang-troubleshooting`, `golang-spf13-viper`, `golang-swagger`.

Baseline expectations derived from them: test-first, exported doc comments,
wrapped lowercase errors, `errors.Is` sentinels, context first and propagated,
no unchecked errors, path-traversal-safe I/O, bounded resources, `t.Parallel`
for independent tests, named subtests, and `golangci-lint` clean.

## Non-negotiable architecture rules

1. **Templates are data.** Never execute shell commands, scripts, plugins, or
   code from registry/template content. The CLI performs all side effects.
2. **Validate before writing.** Resolve and preflight the entire plan before the
   target is touched. On error, the target must remain unchanged.
3. **Atomic creation.** Render into a sibling staging directory on the same
   filesystem, then rename into place. No partial targets.
4. **Transactional mutation.** `add`/`remove` snapshot impacted files to
   `.smokeforge/backups/<timestamp>/`, verify each op, roll back on failure.
5. **Reproducibility.** Registry tarballs are SHA-256 verified; projects pin
   tag, commit, and template/overlay checksums in `.smokeforge.lock.json`.
6. **Composable overlays, not preset matrices.** Base template + feature
   overlays using anchors and structured patches.
7. **Hermetic tests.** No network or real user cache in unit tests; inject
   sources and use `t.TempDir()`.
8. **Strict templates.** Missing variables are errors; never emit `<no value>`.
9. **Determinism.** Same registry pin + selections = byte-identical output,
   except locally generated `.env` secrets.
10. **Secrets stay local.** Generate with `crypto/rand`, write only to `.env`,
    never commit them or bake them into templates.

Full details: [architecture overview](docs/architecture/overview.md).

## Current state

| Area | State |
| --- | --- |
| Repo scaffold, Makefile, CI, docs | Done |
| CLI entrypoint + `version` | Done |
| Manifest schema + validation | Done |
| Registry local + remote/cache/checksum/extraction | Done |
| `list`, `registry`, `doctor` | Done |
| `new`, resolver, renderer, patches, engine, state, wizard | Done (M2) |
| `add`, `remove`, drift, backups, rollback, project doctor | Done (M4 lifecycle) |
| Version metadata, goreleaser, release workflow, release docs | Done (M5 engineering) |
| Cross-platform CI matrix, cross-build, govulncheck, security policy | Done |
| Registry templates/overlays | In progress (M3, Next first) |
| `install`, fullstack | Planned (M4 remainder) |
| Release/distribution | Planned (M5) |

If the table and code disagree, the code wins — update the table.

## Commands

```bash
make build                    # bin/smokeforge
make test                     # go test ./...
make lint                     # golangci-lint run (v2.x, CI pins v2.11.0)
make fmt                      # golangci-lint fmt ./...
make vulncheck                # govulncheck
make release-check            # goreleaser check
make release-snapshot         # dist/ archives for all platforms

go run ./cmd/smokeforge version
go run ./cmd/smokeforge list stacks --registry ./testdata/registry
go run ./cmd/smokeforge new /tmp/demo --registry ./testdata/registry \
  --kind backend --stack test/plain --name "Demo" --feature logging \
  --var module_path=example.com/demo --no-input --no-git
go test ./internal/cli/ -run TestVersionCommand -v
go test ./... -race
go test -tags=integration ./...
go run ./cmd/smokeforge add metrics --project-dir /tmp/demo
go run ./cmd/smokeforge remove metrics --project-dir /tmp/demo
```

## Repository map

```text
cmd/smokeforge/        main() only
.goreleaser.yaml       cross-platform release build config
.github/workflows/     CI and tag-triggered release workflow
internal/cli/          Cobra commands, flags, output, exit codes
internal/tui/          interactive wizard and Charm styling
internal/manifest/     schema types, strict YAML load + Validate()
internal/registry/     source dispatch, local dirs, HTTP tarball, cache
internal/resolve/      selection -> ordered generation plan
internal/render/       strict text/template rendering
internal/patch/        anchors, markers, JSON/YAML/TOML, deps, env
internal/engine/       staging, apply, add/remove, backups, atomic commit
internal/project/      state, lockfile, hashes, drift, backups
internal/install/      per-stack install adapters (planned)
internal/version/      build metadata
docs/architecture/     system design and engine internals
docs/registry/         manifest + overlay authoring contract
docs/development/      setup, testing, release, security, cross-platform, roadmap
docs/superpowers/      design specs and implementation plans
testdata/registry/     neutral local registry used by integration tests/manual runs
```

Dependency direction: `cli` → libraries. Libraries never import `cli`.
`engine` is the only package that mutates project targets. `manifest` never
touches the network. `registry` depends on `manifest`.

## How to make a change (workflow)

1. **Pick the next task** from the active plan or roadmap. Keep scope small.
2. **Write a failing test first.** Run it and confirm it fails for the expected
   reason. See [testing](docs/development/testing.md).
3. **Implement the minimum** to pass. No speculative features.
4. **Refactor** with tests green.
5. **Run the gate:** `go test ./...`, `golangci-lint run`, `gofmt`.
6. **Update docs** when you change commands, flags, manifest fields, operation
   types, or behavior. Update the README command/status tables if needed.
7. **Commit only if the operator asked.** Use conventional commit prefixes.

## Where to change what

| Task | Files |
| --- | --- |
| Add/modify a CLI command | `internal/cli/<cmd>.go`, `internal/cli/root.go`, tests |
| Change generation flow | `internal/engine`, `internal/resolve`, `internal/render`, `internal/patch`, `internal/project`; update Mermaid docs |
| Change lifecycle behavior | `internal/engine/lifecycle.go`, `internal/project` (drift/backup), `internal/cli/add.go`, `remove.go`; update lifecycle docs |
| Add a manifest field/rule | `internal/manifest/manifest.go`, `validate.go`, tests, `docs/registry/manifest.md` |
| Change registry fetching/cache | `internal/registry/*`, tests with `httptest` |
| Add a patch operation | `internal/patch/*`, `docs/registry/overlays.md`, contract fixtures |
| Change generation flow | `internal/resolve`, `render`, `engine`, `project`; `docs/architecture/generation-engine.md` |
| Add a stack or feature | Registry repo first; CLI changes only if a new capability is required |
| Update the plan/roadmap | `docs/superpowers/plans/*`, `docs/development/roadmap.md` |
| Change release behavior | `.goreleaser.yaml`, `.github/workflows/release.yml`, `internal/version`, `docs/development/release.md`, `PUBLISHING.md` |
| Change security behavior | `internal/registry`, `internal/patch`, `internal/engine`, `SECURITY.md`, `docs/development/security.md` |
| Change platform behavior | `internal/engine`, `internal/project`, `.github/workflows/ci.yml`, `docs/development/cross-platform.md` |

## Coding conventions

- Wrap errors with context: `fmt.Errorf("load manifest: %w", err)`.
- Small files with one responsibility; split before a file grows unwieldy.
- Prefer standard library; existing dependencies: Cobra, `yaml.v3`.
- No comments unless explaining a non-obvious workaround.
- Use `filepath` everywhere; tests must pass on case-insensitive and
  case-sensitive filesystems.
- User-facing errors: what failed, why, and the valid next action.
- Keep output machine-parseable under `--json`.

## Common pitfalls

| Pitfall | Guard |
| --- | --- |
| Tests hitting the network | Inject local source/`httptest`; pass `CacheDir: t.TempDir()` |
| Anchor matches zero or multiple times | Fail preflight; never guess |
| Partial project on failure | Stage + rename; test failure injection |
| Overwriting user edits | Compare stored SHA-256; refuse/confirm |
| Path traversal in archives | Reject absolute paths, `..`, symlinks, hard links |
| Tag injection into URLs/paths | Validate tags with a strict pattern |
| Nondeterministic golden tests | Mask/seeded secrets; fixed variable fixtures |
| IDE "missing module" noise | Run `go mod tidy`; ignore stale LSP until reindex |

## Scope discipline

**Generated-app internals are not decided.** Do not choose or implement
frameworks, libraries, ORMs, clients, migration tools, or project layouts for
the apps smokeforge generates. Backend HTTP frameworks, mobile UI approaches,
desktop framework selection (GPUI vs EGUI), and feature library choices are
pending discussion with the operator. Stack IDs are provisional. Work only on
CLI/registry infrastructure until those decisions are recorded.

In v1: Go/Rust/Python backends, Next.js, SwiftUI, Compose, GPUI/EGUI, Postgres,
Redis cache, RabbitMQ, JWT + TOTP, Docker, GitHub Actions, local/remote registry,
new/add/remove/install/list/doctor.

Out of v1: SSE, search, storage, MySQL/SQLite, Redis Streams, observability,
third-party registries, template upgrades. Do not start these without an
approved design.

## When to stop and ask

- A task requires executing template-provided code.
- A change breaks atomicity, reproducibility, or hermetic tests.
- The registry contract must change in a backwards-incompatible way.
- The plan and code disagree in a way that affects user-visible behavior.
- A dependency or toolchain requirement would be added to the CLI itself.
```
