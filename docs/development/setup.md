# Local development setup

## Prerequisites

**Required**

| Tool | Version | Notes |
| --- | --- | --- |
| Go | 1.25+ | `go version` |
| git | 2.30+ | Used by `smokeforge` and for development |
| golangci-lint | v2.x (CI pins v2.11.0) | `golangci-lint version` |

**Optional** — only needed when testing generated projects locally:

| Toolchain | For |
| --- | --- |
| Rust (`cargo`) | `rust/axum`, GPUI, EGUI bases |
| Python 3.12+ and `uv` | `python/fastapi` base |
| Node 20+ and a package manager | `next` base |
| Xcode | `ios/swiftui` base |
| Android SDK + Gradle | `android/compose` base |
| Docker | compose-based feature tests |

The CLI itself does not require these to build or run; `smokeforge install`
invokes them only for the targets a project contains.

## Build and run

```bash
git clone https://github.com/BoatengPeter/smokeforge.git
cd smokeforge
make build
./bin/smokeforge version
```

Without `make`:

```bash
go build -o bin/smokeforge ./cmd/smokeforge
go run ./cmd/smokeforge version
```

## Test and lint

```bash
make test              # go test ./...
make lint              # golangci-lint run
make fmt               # gofmt -w .
```

Focused runs:

```bash
go test ./internal/cli/ -run TestVersionCommand -v
go test ./internal/manifest/ -v
go test ./... -race
```

## Working with a local registry

During development, point the CLI at a local directory instead of the network:

```bash
go run ./cmd/smokeforge list stacks --registry ./testdata/registry
```

A local registry is any directory containing a valid `manifest.yaml`. Create
fixtures under `testdata/` or per-package `testdata/` directories. The tag is
reported as `local`; no checksums or network access are involved.

## Repository layout

```text
cmd/smokeforge/        entrypoint (main.go only)
internal/cli/          Cobra commands, flags, output
internal/tui/          interactive wizard (planned)
internal/manifest/     schema types, strict load/validation
internal/registry/     source dispatch, HTTP tarball, checksum, cache
internal/resolve/      selection -> plan (planned)
internal/render/       strict templates (planned)
internal/patch/        anchors, JSON/YAML/TOML, dependency adapters (planned)
internal/engine/       preflight, staging, commit, rollback (planned)
internal/project/      state, lockfile, drift, add/remove (planned)
internal/install/      per-stack install steps (planned)
internal/fsx/          atomic writes and path safety (planned)
internal/gitx/         git init (planned)
internal/version/      build metadata
docs/                  architecture, registry, development docs
docs/superpowers/      design records and implementation plans
```

Package boundaries:

- `cli` depends on libraries; libraries never import `cli`.
- `manifest` has no filesystem side effects beyond reading.
- `registry` depends on `manifest` for validation.
- `engine` orchestrates `render`/`patch`; it is the only package that writes to
  the target filesystem.
- `project` owns state/lockfile serialization.
- No registry content may influence shell execution.

## Conventions

- **TDD is mandatory** for production code: write the failing test, watch it
  fail, implement the minimum, then refactor. No production code without a
  failing test first.
- Wrap errors with context: `fmt.Errorf("load manifest: %w", err)`.
- Prefer small, focused files with one responsibility.
- No comments unless a non-obvious workaround needs explanation.
- Do not add dependencies without checking the existing stack first (Cobra for
  CLI, `yaml.v3` for YAML).
- User-facing errors must say what failed, why, and what to do next.
- Never execute commands from registry/template content.
- Keep generated output deterministic; secrets are the only exception.

## Commits and branches

- Conventional commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`, `test:`,
  `refactor:`.
- Keep commits scoped and buildable. Run `go test ./...` before committing.
- Do not commit generated binaries (`bin/` is gitignored).

## CI

`.github/workflows/ci.yml` runs on pushes to `main` and on pull requests:

1. `go test ./...`
2. `golangci-lint run` (v2.11.0)

The registry repository has its own contract/build matrix CI; changes to
template contracts must keep both repositories green.

## Common tasks

### Add a CLI command

1. Create `internal/cli/<name>.go` with a `new<Name>Cmd(opts *options)`.
2. Register it in `internal/cli/root.go`.
3. Write `internal/cli/<name>_test.go` using `NewRootCmd` with a `bytes.Buffer`.
4. Update `docs/architecture/cli.md` and the command table in `README.md`.

### Add a manifest field

1. Add the field to the relevant struct in `internal/manifest/manifest.go`.
2. Decide whether it is required; add validation in `validate.go`.
3. Extend `manifest_test.go` / `validate_test.go` for accept and reject cases.
4. Update `docs/registry/manifest.md` field tables.

### Add a feature overlay contract

1. Define the manifest entry in the registry repo, not this repo.
2. Extend `internal/patch` only if a new operation type is needed.
3. Add contract tests that apply the overlay to every supported base.
4. Document the operation in `docs/registry/overlays.md`.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `no required module provides package ...` | Run `go mod tidy`; the IDE may lag behind. |
| Registry offline error | No cache entry yet; run once online or pass `--registry <dir>`. |
| Checksum mismatch | Registry asset and `.sha256` are out of sync; do not bypass. |
| `golangci-lint` version errors | Use v2.x; CI pins v2.11.0. |
| Tests touch the network | Inject a local source / `httptest` server; unit tests must be hermetic. |
```
