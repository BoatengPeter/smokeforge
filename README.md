# smokeforge

**Scaffold production-ready projects from a versioned template registry.**

[**Download the latest release →**](https://github.com/BoatengPeter/smokeforge/releases/latest)

`smokeforge` is a Go CLI for generating backend, frontend, mobile, desktop, and
fullstack projects for clients. Pick a stack, compose the features you need
(database, auth, client state, analytics, Docker, CI), and get a wired,
buildable project — reproducibly, from a pinned template registry.

> **Status:** v0.2.5 is published. CLI generation, lifecycle, fullstack composition, native egui base generation, and release engineering are complete.
> The Next.js template, Go/Gin backend (four architectures), Rust/Axum backend,
> and Electron desktop base are implemented. Mobile and remaining native desktop
> templates remain provisional.
> `smokeforge install` and Homebrew are not available yet. See the [stack catalog](STACKS.md) and
> [roadmap](docs/development/roadmap.md).

---

## Install

> **Do I need Go? No.** Download the prebuilt binary for your operating system.
> Go is only needed if you choose to run `go install` or build from source.

You do **not** need Go to use a prebuilt release.

### 1. Check what you need

| You want to... | You need |
| --- | --- |
| Create a project | smokeforge only |
| Run a generated Next.js app | Node.js 24+ and pnpm 11+ (npm and bun also work) |
| Use the fullstack preset | Postgres 17; Docker is the easiest way to run it |
| Use Docker/CI/E2E overlays | Docker and/or a GitHub repository |

### 2. Download smokeforge

Open the [latest release](https://github.com/BoatengPeter/smokeforge/releases/latest)
and download the file for your computer:

| Computer | File |
| --- | --- |
| macOS Apple Silicon | `smokeforge_<version>_darwin_arm64.tar.gz` |
| macOS Intel | `smokeforge_<version>_darwin_amd64.tar.gz` |
| Linux Intel/AMD | `smokeforge_<version>_linux_amd64.tar.gz` |
| Linux ARM | `smokeforge_<version>_linux_arm64.tar.gz` |
| Windows | `smokeforge_<version>_windows_amd64.zip` |

`amd64` means a standard Intel/AMD computer. `arm64` means Apple Silicon or an
ARM server.

#### macOS and Linux

```bash
tar -xzf smokeforge_*.tar.gz
sudo install -m 0755 smokeforge /usr/local/bin/smokeforge
smokeforge version
```

If `sudo` is not available, put the binary somewhere on your `PATH`:

```bash
mkdir -p ~/.local/bin
mv smokeforge ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
smokeforge version
```

Add the `export PATH=...` line to `~/.zshrc` or `~/.bashrc` to make it permanent.

#### Windows

1. Download the `.zip` file.
2. Right-click and choose **Extract All**.
3. Open the extracted folder in PowerShell.
4. Run:

```powershell
.\smokeforge.exe version
```

To run `smokeforge` from any folder, add the extracted folder to your Windows
`PATH`.

#### Verify the download (recommended)

The release page includes `checksums.txt`. Compute the hash of the archive you
downloaded:

```bash
# Linux
sha256sum smokeforge_*.tar.gz

# macOS
shasum -a 256 smokeforge_*.tar.gz
```

Compare it with the matching line in `checksums.txt`. If the values differ, do
not run the binary.

### 3. Create your first project

Open a terminal and run:

```bash
smokeforge new my-first-app --preset frontend
```

The command creates a complete Next.js app in `my-first-app` with the official
registry. Then:

```bash
cd my-first-app
pnpm install
pnpm dev
```

Open <http://localhost:3000>.

If you prefer the guided experience, run `smokeforge new` with no arguments and
answer the prompts.

## What happens when you run `smokeforge new`

1. The CLI contacts the official registry release on GitHub.
2. It downloads the versioned `registry.tar.gz` and verifies it before use.
3. It caches the registry locally for faster and offline runs.
4. It validates the registry manifest.
5. It resolves your preset, features, and required companions.
6. It renders the template and overlays into a staging directory.
7. It generates local secrets such as `BETTER_AUTH_SECRET`.
8. It writes `.smokeforge.json`, `.smokeforge.lock.json`, `.env`, and
   `.env.example`.
9. It initializes Git unless `--no-git` is used.
10. It atomically moves the finished project into place.

The CLI does not install Node.js, pnpm, Postgres, or Docker for you. Those are
needed only when you run the generated app.

## Where templates come from

smokeforge has two separate releases:

- The **CLI release** contains the smokeforge binary for your operating system.
- The **registry release** contains the templates, presets, overlays, skills, and
  `manifest.yaml` that the CLI reads.

The CLI downloads the latest registry release automatically. Projects record the
registry version in `.smokeforge.lock.json`, so repeated generation can use the
same template data. For local registry development, pass `--registry <dir>`.

## Only if you already have Go

If you have Go 1.26.11+:

```bash
go install github.com/BoatengPeter/smokeforge/cmd/smokeforge@v0.2.5
smokeforge version
```

From source:

```bash
git clone https://github.com/BoatengPeter/smokeforge.git
cd smokeforge
make build
./bin/smokeforge version
```

Homebrew is not available yet.

## Choose a preset

Presets are starting points. Required companions are added automatically.

| Preset | Use it for | Includes |
| --- | --- | --- |
| `frontend` | A polished frontend or landing page | Next.js, shadcn/ui, SEO, API helpers, Jest |
| `fullstack` | A Next.js app with its own auth/database | Frontend + Postgres + Better Auth + Zustand + TanStack Query |
| `fullstack-api` | Next.js + a Go or Rust API | Next.js BFF + API + Postgres/JWT + TanStack Query + Docker + CI |
| `desktop` | A secure Electron desktop app | Electron + React Router + shadcn/ui + TanStack Query + Zustand + SQLite + Drizzle |
| `api` | A backend API with persistence and accounts | Backend + Postgres + JWT auth |
| `api-custom` | Choosing your own backend features | Select database, cache, queue, scheduler, dedupe, compression, auth, TOTP, media, search, search sync, pagination, Docker, and CI |

Defaults are kind-scoped: `fullstack` for frontend, `fullstack-api` for fullstack, `api` for backend.

```bash
smokeforge new my-frontend --preset frontend
smokeforge new my-saas --preset fullstack

# Fullstack API + Next.js (Go or Rust)
smokeforge new my-saas --kind fullstack --backend go/gin-layered
smokeforge new my-saas --kind fullstack --backend rust/axum
smokeforge new my-saas --kind fullstack --backend go/gin-layered --desktop electron

# Standalone Electron desktop app
smokeforge new my-desktop --kind desktop --stack desktop/electron

# Standalone native Rust egui desktop app
smokeforge new my-egui --kind desktop --stack desktop/egui --preset desktop-egui

# Go API: choose an architecture and start from the backend preset
smokeforge new my-api --kind backend --stack go/gin-layered --preset api

# Custom Go API: choose the features yourself
smokeforge new my-api --kind backend --stack go/gin-hexagonal --preset api-custom --feature database:postgres --feature auth:jwt
```

Select individual features instead:

```bash
smokeforge new my-app --preset frontend --feature docker --feature ci:github
```

Feature dependencies are automatic:

- `auth` adds `database:postgres`.
- `ci` adds `testing:e2e`.
- Removing a required feature is blocked unless you pass `--cascade`.

## Run a fullstack app locally

There are two fullstack shapes:

- `fullstack`: Next.js owns auth and the database through Route Handlers.
- `fullstack-api`: Next.js is the BFF and a Go or Rust API owns JWT auth and
  persistence.

Add `--desktop electron` or `--desktop egui` to fullstack-api to generate `apps/desktop`. The
desktop main process owns the API session; Drizzle-backed SQLite is local
storage; TanStack Query and Zustand are added automatically. Optional
`shell:tray` and `auth:oauth-pkce` features are available. Offline Library sync
is included with idempotency, version checks, retries, and manual conflicts.

### Fullstack API + Next.js

```bash
smokeforge new my-product --kind fullstack --backend go/gin-layered
cd my-product
make install
make api-dev        # API on :8080
make web-dev        # Next.js on :3000
```

Or run the whole stack with Docker:

```bash
make docker-up
```

The browser calls same-origin `/api/auth/*` BFF routes. Raw JWT access and
refresh tokens stay in HttpOnly cookies and never reach browser JavaScript.
Read `CONTEXT-MAP.md` first, then the API and web `CONTEXT.md` files.

The Next.js-owned fullstack preset needs Postgres.

### With Docker

Install Docker Desktop or Docker Engine first:
<https://docs.docker.com/get-docker/>.

```bash
smokeforge new my-saas --preset fullstack
cd my-saas
make db-up
make install
make db-migrate
make dev
```

`make db-up` checks Docker and explains what to do if Docker is missing or not
running. Open <http://localhost:3000/signup>. The verification code is printed
in the terminal during local development.

If port `5432` is already used, change the host port in `compose.db.yaml` and
update `DATABASE_URL` in `.env` to match.

If you do not want to install Docker, point `DATABASE_URL` at an existing
Postgres instance and run `make db-migrate` instead.

### With Docker and CI

Add Docker Compose and CI to the default fullstack preset when you want them:

```bash
smokeforge new my-app --preset fullstack --feature docker --feature ci:github
cd my-app
make install
make db-up
make db-migrate
make dev
```

To run the full stack in containers:

```bash
docker compose up --build
```

## Everyday commands

```bash
# See available stacks and features
smokeforge list stacks
smokeforge list features

# Add auth to an existing frontend project
smokeforge add auth --project-dir my-app

# Remove a feature (fails if another active feature needs it)
smokeforge remove auth --project-dir my-app

# Check tooling, registry, and project drift
smokeforge doctor --project-dir my-app
```

Inside a generated project, use the Makefile:

```bash
make dev
make lint
make typecheck
make test
make build
make ci
```

Docker, E2E, and database targets appear only when their overlays are selected.

## Help and shell completion

```bash
smokeforge --help
smokeforge help new
smokeforge new --help
smokeforge add --help
smokeforge remove --help
```

Enable completion in your shell:

```bash
# Bash
source <(smokeforge completion bash)

# Zsh
smokeforge completion zsh > "${fpath[1]}/_smokeforge"

# Fish
smokeforge completion fish | source

# PowerShell
smokeforge completion powershell | Out-String | Invoke-Expression
```

Restart your shell after installing completion.

## Why smokeforge

- **One command, production-shaped output** — auth, database, API conventions,
  logging, tests, and agent instructions arrive together.
- **Composable overlays** — add or remove features later without regenerating
  the project.
- **Reproducible** — projects pin the registry tag, commit, and template
  checksums.
- **Recoverable** — generation is atomic; `add`/`remove` take backups and roll
  back on failure.
- **Safe by design** — templates are declarative data, downloads are
  checksum-verified, and secrets stay local in `.env`.
- **Agent-ready** — generated apps include `AGENTS.md`, `CLAUDE.md`,
  `CONTEXT.md`, and project-scoped skills.

## Supported stacks

| Kind | Stack ID | Status |
| --- | --- | --- |
| Frontend | `next` | Implemented |
| Backend | `go/gin-flat`, `go/gin-layered`, `go/gin-hexagonal`, `go/gin-ddd` | Implemented |
| Backend | `rust/axum` (Axum + SQLx + JWT) | Implemented |
| Backend | `python/fastapi` | Provisional |
| Mobile | `ios/swiftui`, `android/compose` | Provisional |
| Desktop | `desktop/electron` (Electron + React + TypeScript) | Implemented |
| Desktop | `desktop/egui` (Rust + eframe/egui + SQLite) | Implemented |
| Desktop | `desktop/gpui` | Deferred |
| Workspace | `workspace/fullstack` | Implemented for API + Next.js |

## Supported features

| Feature | Variants | Next.js status |
| --- | --- | --- |
| `database` | `postgres` | Implemented |
| `auth` | `session`, `bff` | Frontend session or API JWT BFF |
| `data` | `tanstack-query` | Implemented |
| `state` | `zustand` | Implemented |
| `testing` | `e2e` | Implemented |
| `analytics` | `posthog` | Implemented; opt-in |
| `docker` | — | Implemented |
| `ci` | `github` | Implemented; frontend requires E2E |
| `cache` | `redis` | Backend only, deferred |
| `queue` | `rabbitmq` | Backend only, deferred |
| `totp` | — | Implemented for Go (`auth:jwt` required) |
| `sync` | `library` | Go/Rust versioned API + desktop offline outbox; fullstack desktop default |

## Requirements and troubleshooting

**`smokeforge: command not found`**
The binary is not on your `PATH`. Move it to `/usr/local/bin` or add its folder
to `PATH`, then reopen the terminal.

**Registry download fails**
Check your internet connection. If you are working on the registry itself, use
a local checkout instead:

```bash
smokeforge new my-app --registry /path/to/smokeforge-registry
```

**`pnpm: command not found`**
Install Node.js 24 from <https://nodejs.org>, then run:

```bash
corepack enable
corepack prepare pnpm@11.13.1 --activate
```

**Docker is not installed**
`make db-up` checks for Docker and prints an installation link. Install Docker
Desktop or Docker Engine from <https://docs.docker.com/get-docker/>. If you
already have Postgres, set `DATABASE_URL` in `.env` and run `make db-migrate`
instead.

**Docker is installed but not running**
Start Docker Desktop (macOS/Windows) or the Docker service (Linux), then run
`make db-up` again.

**Fullstack app cannot connect to the database**
Make sure Postgres is running on the host and port in `.env` (`DATABASE_URL`),
then run `pnpm db:migrate` or `make db-migrate`.

**Port 3000 is already in use**

```bash
pnpm dev -- -p 3001
```

Use the matching URL in `NEXT_PUBLIC_APP_URL` and `BETTER_AUTH_URL`.

**No verification email arrives locally**
That is expected. During local development the OTP is printed in the terminal
running `pnpm dev`. Production requires `RESEND_API_KEY` and `EMAIL_FROM`.

**E2E tests say the browser is missing**

```bash
pnpm test:e2e:install
pnpm test:e2e
```

**Removing a feature fails**
The overlay may change files that cannot be reversed automatically, such as
package dependencies. The command refuses rather than corrupting the project;
follow the printed manual-removal instructions.

## Uninstall smokeforge

There is no `smokeforge uninstall` command. `smokeforge remove` removes project
features, not the CLI.

macOS/Linux:

```bash
rm -f /usr/local/bin/smokeforge
# or, if installed there:
rm -f ~/.local/bin/smokeforge

rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/smokeforge"   # Linux
rm -rf "$HOME/Library/Caches/smokeforge"              # macOS
```

Windows:

```powershell
Remove-Item "$env:LOCALAPPDATA\smokeforge" -Recurse -Force
Remove-Item "C:\path\to\smokeforge.exe"
```

If installed with Go:

```bash
rm -f "$(go env GOPATH)/bin/smokeforge"
```

Also remove any `PATH` entry you added. Generated projects are not deleted.

## Platforms and security

- **Platforms:** Linux, macOS, and Windows on `amd64` and `arm64`.
- **CI:** unit + integration tests on all three operating systems; race and lint
  on Linux; cross-compilation of all six release targets; `govulncheck`.
- **Security:** templates are declarative and never executed; remote registries
  require HTTPS, SHA-256 verification, and validated redirects; archives are
  extracted with `os.Root`; secrets stay local in `.env`.

See [cross-platform compatibility](docs/development/cross-platform.md) and the
[security policy](SECURITY.md).

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
| `smokeforge remove <feature>...` | Revert reversible overlays safely; `--cascade` removes dependents | Implemented |
| `smokeforge install` | Install dependencies for recorded targets | Planned |
| `smokeforge list [stacks\|features]` | Inspect registry contents | Implemented |
| `smokeforge registry info` | Show registry source/tag/schema/counts | Implemented |
| `smokeforge update` | Update the smokeforge binary from GitHub releases | Implemented |
| `smokeforge registry update` | Refresh the local registry cache | Implemented |
| `smokeforge doctor` | Check environment, registry, and project drift | Implemented |
| `smokeforge completion <shell>` | Generate shell completion | Implemented |
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
- [Stack catalog and versions](STACKS.md)
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
  - [Release guide](docs/development/release.md)
  - [Publishing checklist](PUBLISHING.md)
  - [Security](SECURITY.md)
  - [Cross-platform](docs/development/cross-platform.md)
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
make lint             # golangci-lint run ./...
make fmt              # golangci-lint fmt ./...
make vulncheck        # govulncheck
make release-check    # goreleaser check
make release-snapshot # build dist/ archives for all platforms
```

CI tests Linux, macOS, and Windows, cross-compiles all six release targets, and
runs `govulncheck`.
Production code is written test-first. See
[development/setup.md](docs/development/setup.md) and
[development/testing.md](docs/development/testing.md).

### Repository layout

```text
cmd/smokeforge/        entrypoint
internal/cli/          Cobra commands and flags
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
STACKS.md              stack catalog, versions, defaults, status
docs/                  design, architecture, registry, and development docs
# separate registry repo: github.com/BoatengPeter/smokeforge-registry
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
   Next.js base, fullstack presets, overlays, and project-scoped skills. _(Next.js implemented; other stacks provisional)_
5. **Full matrix** — fullstack composition and client wiring complete; `install` remains pending.
6. **Release** — goreleaser, checksums, release workflow, and install docs.
   _(v0.2.5 published)_

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
