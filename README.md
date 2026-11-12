# smokeforge

**Scaffold production-ready projects from a versioned template registry.**

`smokeforge` is a Go CLI for generating backend, frontend, mobile, desktop, and
fullstack projects for clients. Pick a stack, compose the features you need
(database, cache, queues, auth, TOTP, Docker, CI), and get a wired, buildable
monorepo — reproducibly, from a pinned template registry.

> **Status:** foundation milestone complete. The CLI provides `version`, `list`,
> `registry info/update`, and `doctor`; registries load from local directories or
> checksum-verified remote tarballs with an offline-capable cache. Project
> generation (`new`, resolver, renderer, patch engine) is the next milestone.
> See the [roadmap](docs/development/roadmap.md) for the exact state.

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

---

## Quick start

Requires Go 1.25+ and `git`.

```bash
git clone https://github.com/BoatengPeter/smokeforge.git
cd smokeforge
make build

./bin/smokeforge version
./bin/smokeforge list stacks
```

Generate a project (planned interface, lands with the generation milestone):

```bash
# Interactive
smokeforge new

# Non-interactive CI usage
smokeforge new ./acme \
  --kind fullstack \
  --backend go/chi \
  --clients next,ios/swiftui \
  --feature database:postgres \
  --feature auth:jwt \
  --feature totp \
  --feature cache:redis \
  --feature queue:rabbitmq \
  --feature docker \
  --feature ci:github \
  --module github.com/acme/acme \
  --no-input

cd acme
smokeforge install
```

Generated fullstack layout:

```text
acme/
  apps/
    api/                 # Go, Rust, or Python service
    web/                 # Next.js app
    ios/                 # SwiftUI app
    android/             # Jetpack Compose app
    desktop/             # GPUI or EGUI app
  deploy/                # dev compose, reverse proxy, ops files
  .github/workflows/     # CI for generated targets
  .smokeforge.json       # choices, targets, applied overlays, file hashes
  .smokeforge.lock.json  # pinned registry tag/commit/checksums
  README.md
```

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
| `smokeforge new [dir]` | Scaffold a project (wizard or flags) | Planned |
| `smokeforge add <feature>...` | Apply overlays to an existing project | Planned |
| `smokeforge remove <feature>...` | Revert overlays safely | Planned |
| `smokeforge install` | Install dependencies for recorded targets | Planned |
| `smokeforge list [stacks\|features]` | Inspect registry contents | Implemented |
| `smokeforge registry info` | Show registry source/tag/schema/counts | Implemented |
| `smokeforge registry update` | Refresh the local registry cache | Implemented |
| `smokeforge doctor` | Check environment and registry health | Implemented |
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

---

## Development

```bash
make build     # build ./bin/smokeforge
make test      # go test ./...
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
internal/resolve/      selection -> ordered generation plan (planned)
internal/render/       strict template rendering (planned)
internal/patch/        anchors, structured patches, dependency adapters (planned)
internal/engine/       staging, apply, rollback (planned)
internal/project/      project state, lockfile, add/remove (planned)
internal/install/      per-stack dependency installation (planned)
internal/fsx/          atomic filesystem helpers (planned)
internal/gitx/         git initialization (planned)
internal/version/      build version metadata
docs/                  design, architecture, registry, and development docs
```

---

## Roadmap

1. **Foundation** — repository, CLI skeleton, manifest validation, registry
   fetch/cache/verify, `list`/`registry`/`doctor`. _(in progress)_
2. **Single-target generation** — resolver, renderer, patch engine, staging,
   state/lock, `new` wizard and flags.
3. **Overlay vertical slice** — Go base + Postgres/auth/TOTP/cache/queue,
   Docker/CI, then `add`/`remove`.
4. **Full matrix** — all bases, fullstack composition, `install`, `doctor`.
5. **Release** — registry contract tests, golden/e2e suites, goreleaser,
   Homebrew, v1.0.0.

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
