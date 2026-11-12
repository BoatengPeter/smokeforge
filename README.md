# smokeforge

**Scaffold production-ready projects from a versioned template registry.**

`smokeforge` is a Go CLI for generating backend, frontend, mobile, desktop, and
fullstack projects for clients. Pick a stack, compose the features you need
(database, cache, queues, auth, TOTP, Docker, CI), and get a wired, buildable
monorepo — reproducibly, from a pinned template registry.

> **Status:** M1/M2 generation, the M4 lifecycle, and M5 release engineering are
> complete. `smokeforge new`, `add`, and `remove` work for single-target
> projects with drift detection, backups, and rollback. Cross-platform archives,
> checksums, version metadata, and a draft release workflow are prepared.
> Publishing and M3 registry content remain pending; fullstack and `install`
> are planned. See the [roadmap](docs/development/roadmap.md).

---

## Why smokeforge

- **Multi-stack, one workflow** — Go, Rust, Python APIs; Next.js web; native
  SwiftUI iOS; Jetpack Compose Android; GPUI and EGUI desktop.
- **Composable features** — start from a base template, then overlay database,
  cache, queues, auth, TOTP, Docker, and CI without waiting for a new preset.
- **Reproducible** — every project records the registry tag, commit, and
  template checksums in a lockfile. The same choices regenerate the same tree.
- **Recoverable** — generation stages files and commits atomically; `add` and
  `remove` snapshot touched files, detect drift, and roll back on failure.
- **Safe by design** — templates are declarative data. No template shell hooks,
  no arbitrary code execution, checksum-verified downloads.
- **Agent- and CI-friendly** — the interactive wizard has full flag parity.

---

## Supported stacks (v1)

> **Open decision:** the frameworks, libraries, and package choices used
> *inside generated apps* (routers, ORMs, auth/cache/queue clients, migration
> tooling) are **not finalized**. Stack IDs below are provisional names for
> discussion; templates will be built only after those internals are agreed.

| Kind | Stack ID (provisional) | Language / platform |
| --- | --- | --- |
| Backend | `go/chi` | Go — HTTP framework TBD |
| Backend | `rust/axum` | Rust — HTTP framework TBD |
| Backend | `python/fastapi` | Python — HTTP framework TBD |
| Frontend | `next` | Next.js / React |
| Mobile | `ios/swiftui` | Swift — UI approach TBD |
| Mobile | `android/compose` | Kotlin — UI approach TBD |
| Desktop | `desktop/gpui` | Rust — GPUI under consideration |
| Desktop | `desktop/egui` | Rust — EGUI under consideration |
| Workspace | `workspace/fullstack` | Monorepo root for multi-target projects |

Feature implementation internals (database driver/ORM, Redis client, queue
client, JWT/TOTP libraries, migration tool) are also pending discussion.

## Supported features (v1)

| Feature | Variants | Applies to |
| --- | --- | --- |
| `database` | `postgres` | Backends |
| `cache` | `redis` | Backends |
| `queue` | `rabbitmq` | Backends |
| `auth` | `jwt` | Backends |
| `totp` | — (requires `auth:jwt`) | Backends |
| `docker` | — | Repo root |
| `ci` | `github` | Repo root |

Out of v1: SSE, search (Meilisearch/Elasticsearch), storage
(MinIO/imgproxy/nginx or Cloudflare R2), MySQL/SQLite variants, Redis-streams
queues, observability, third-party registries, and three-way template upgrades.

## Platforms and security

- **Platforms:** Linux, macOS, and Windows on `amd64` and `arm64`.
- **CI:** unit + integration tests on all three operating systems; race and lint
  on Linux; cross-compilation of all six release targets; `govulncheck`.
- **Security:** templates are declarative and never executed; remote registries
  require HTTPS, SHA-256 verification, and validated redirects; archives are
  extracted with `os.Root`; secrets stay local in `.env`.

See [cross-platform compatibility](docs/development/cross-platform.md) and the
[security policy](SECURITY.md).

---

## Quick start

Requires Go 1.26.11+ and `git`.

### Install

After the first tagged release:

```bash
go install github.com/BoatengPeter/smokeforge/cmd/smokeforge@v0.1.0
smokeforge version
```

Prebuilt Linux, macOS, and Windows archives (`amd64`/`arm64`) with
`checksums.txt` are attached to GitHub releases. A Homebrew tap is planned; see
the [release guide](docs/development/release.md).

From source:

```bash
git clone https://github.com/BoatengPeter/smokeforge.git
cd smokeforge
make build

./bin/smokeforge version
./bin/smokeforge list stacks
```

Generate a single-target project against the bundled neutral fixture:

```bash
# Interactive wizard
smokeforge new --registry ./testdata/registry

# Non-interactive CI usage
smokeforge new ./acme \
  --registry ./testdata/registry \
  --kind backend \
  --stack test/plain \
  --name "Acme App" \
  --feature logging \
  --var module_path=example.com/acme \
  --no-input --no-git

cd acme
go build ./...

# Evolve the project later
smokeforge add metrics --project-dir ./acme
smokeforge remove metrics --project-dir ./acme
```

Lifecycle commands locate `.smokeforge.json`, resolve overlays against the
locked registry, refuse drifted files (add supports `--force`), and restore a
snapshot automatically if a mutation fails. Removal is limited to overlays made
of added files, marked insertions, and env entries; overlays with structured
config or dependency patches are refused with a manual-removal message.

M2 project layout:

```text
acme/
  main.go                 # rendered from the registry template
  go.mod
  internal/log/logger.go  # added by the logging overlay
  .env.example            # documented environment variables
  .env                    # local values and generated secrets (gitignored)
  .gitignore
  .smokeforge.json        # choices, targets, applied overlays, file hashes
  .smokeforge.lock.json   # registry pin and template/overlay checksums
```

The fixture is deliberately neutral. Official stack templates wait for the
generated-app framework and package decisions listed under
[pending decisions](docs/development/roadmap.md#pending-decisions-blocking-m3-templates).

The CLI uses Charm-style prompts, boxed summaries, and checkmark progress. Set
`NO_COLOR=1` to disable color.

---

## How it works

```text
flags / wizard
      │
      ▼
resolve selections ──► compatibility matrix (kinds, variants, requires/conflicts)
      │
      ▼
fetch + verify registry (checksum, cache, offline support)
      │
      ▼
render base templates + ordered feature overlays into a staging dir
      │
      ▼
validate every anchor/patch operation
      │
      ▼
write state + lock + .env, git init, atomic rename into place
```

1. **Resolve** — the registry manifest declares stacks, kinds, features,
   variants, and dependency rules. Invalid combinations fail before any file is
   written, with the valid alternatives listed.
2. **Render** — Go `text/template` with strict variables (a missing variable is
   an error, never an empty string).
3. **Patch** — overlays add files, insert at named anchors, and apply structured
   patches to JSON/YAML/TOML and dependency manifests. Every operation must
   match exactly once.
4. **Commit** — generation happens in a sibling staging directory; the target is
   only touched on success. `add`/`remove` snapshot and restore on failure.

Read more in the [architecture docs](docs/architecture/overview.md).

---

## Commands

| Command | Purpose | Status |
| --- | --- | --- |
| `smokeforge new [dir]` | Scaffold a project (wizard or flags) | Implemented |
| `smokeforge add <feature>...` | Apply overlays to an existing project | Implemented |
| `smokeforge remove <feature>...` | Revert reversible overlays safely | Implemented |
| `smokeforge install` | Install dependencies for recorded targets | Planned |
| `smokeforge list [stacks\|features]` | Inspect registry contents | Implemented |
| `smokeforge registry info` | Show registry source/tag/schema/counts | Implemented |
| `smokeforge registry update` | Refresh the local registry cache | Implemented |
| `smokeforge doctor` | Check environment, registry, and project drift | Implemented |
| `smokeforge version` | Print version information | Implemented |

Global flags: `--registry <url|dir>`, `--tag <tag>`, `--cache-dir <dir>`,
`--offline`.

See the [CLI reference](docs/architecture/cli.md) for wizard flows, flag
parity, output conventions, and lifecycle semantics.

---

## Registry and reproducibility

Templates live in a separate, versioned registry repository. `smokeforge`
downloads a tagged release tarball, verifies its SHA-256 checksum, and caches it
under the user cache directory:

```text
$XDG_CACHE_HOME/smokeforge/registry/<hashed-source-tag>/
```

Projects pin the registry tag, resolved commit, and template checksums in
`.smokeforge.lock.json`. `--offline` uses the cache only. A local directory can
be used as a registry for development and tests:

```bash
smokeforge list stacks --registry ./path/to/registry
```

Registry layout, manifest fields, and validation rules are documented in
[registry/manifest.md](docs/registry/manifest.md).

---

## Documentation

- [Documentation index](docs/README.md)
- **Agent guides** — [AGENTS.md](AGENTS.md) (full mindmap) and
  [CLAUDE.md](CLAUDE.md) (quick operating guide)
- **Architecture**
  - [Visual architecture (Mermaid)](docs/architecture.md)
  - [System overview](docs/architecture/overview.md)
  - [CLI and lifecycle](docs/architecture/cli.md)
  - [Generation engine](docs/architecture/generation-engine.md)
  - [Project state and recovery](docs/architecture/project-state.md)
  - [Fullstack composition](docs/architecture/fullstack.md)
- **Registry**
  - [Manifest reference](docs/registry/manifest.md)
  - [Overlay and patch model](docs/registry/overlays.md)
- **Development**
  - [Local setup](docs/development/setup.md)
  - [Testing strategy](docs/development/testing.md)
  - [Roadmap](docs/development/roadmap.md)
- **Design records**
  - [Design spec](docs/superpowers/specs/2026-11-12-smokeforge-design.md)
  - [Foundation plan](docs/superpowers/plans/2026-11-12-smokeforge-foundation.md)
  - [Generation plan](docs/superpowers/plans/2026-11-12-smokeforge-generation.md)
  - [Lifecycle plan](docs/superpowers/plans/2026-11-12-smokeforge-lifecycle.md)

---

## Development

```bash
make build            # build ./bin/smokeforge
make test             # go test ./...
make test-integration # go test -tags=integration ./...
make lint      # golangci-lint run
make fmt       # gofmt -w .
```

CI runs the test suite and golangci-lint on every push and pull request.
Production code is written test-first. See
[development/setup.md](docs/development/setup.md) and
[development/testing.md](docs/development/testing.md).

### Repository layout

```text
cmd/smokeforge/        entrypoint
internal/cli/          Cobra commands and flags
internal/tui/          interactive wizard (planned)
internal/manifest/     registry schema, loading, validation
internal/registry/     local/remote sources, checksums, cache
internal/resolve/      selection -> ordered generation plan
internal/render/       strict template rendering
internal/patch/        anchors, structured patches, dependency adapters
internal/engine/       staging, apply, atomic commit
internal/project/      project state and lockfile
internal/tui/          wizard and terminal styling
internal/install/      per-stack dependency installation (planned)
internal/version/      build version metadata
docs/                  design, architecture, registry, and development docs
```

---

## Roadmap

1. **Foundation** — repository, CLI skeleton, manifest validation, registry
   fetch/cache/verify, `list`/`registry`/`doctor`. _(complete)_
2. **Single-target generation** — resolver, renderer, patch engine, staging,
   state/lock, `new` wizard and flags. _(complete)_
3. **Lifecycle** — `add`/`remove`, drift detection, backups, automatic
   rollback, project-aware `doctor`. _(complete)_
4. **Registry content** — official stack templates and feature overlays.
   _(deferred pending generated-app decisions)_
5. **Full matrix** — fullstack composition, `install`, client wiring.
6. **Release** — goreleaser, checksums, draft release workflow, install/release
   docs. _(engineering complete; publishing pending)_

Details and acceptance criteria: [roadmap](docs/development/roadmap.md).

## Design principles

- **Templates are data.** No hooks, no scripts, no hidden side effects.
- **Fail before writing.** Validate the whole plan first; abort with a fixable
  message.
- **Atomic by default.** Staging + rename for creation; backups + rollback for
  mutation.
- **Pinned and verifiable.** Checksums and lockfiles over "latest".
- **Composable over combinatorial.** Base + overlays instead of thousands of
  preset directories.
