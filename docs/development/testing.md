# Testing strategy

**Status:** M1 and M2 tests are implemented: manifest/registry unit tests,
renderer/patch/resolver/project/engine tests, CLI flag tests, and a
`integration`-tagged generate-and-build test. Registry contract tests and the
native multi-language build matrix arrive with M3.

## Principles

1. **TDD first.** Every production behavior starts as a failing test. If you did
   not watch it fail, you do not know it tests the right thing.
2. **Hermetic by default.** Unit tests never touch the network or the user's real
   cache. Use `httptest`, `t.TempDir()`, and local registry fixtures.
3. **Real files over mocks.** The engine's job is filesystem mutation; tests
   assert on real rendered trees.
4. **Determinism.** Same inputs and registry pin produce byte-identical trees,
   except locally generated secrets.

## Test pyramid

### 1. Unit tests (fast, hermetic)

| Package | What is tested |
| --- | --- |
| `internal/manifest` | YAML loading, unknown-field rejection, all validation rules |
| `internal/registry` | local loading, tag validation, checksum verify, safe extraction, cache/offline behavior |
| `internal/patch` | every operation type: anchors (0/1/N matches), JSON/YAML/TOML patches, dependency adapters, marker idempotency |
| `internal/resolve` | kind limits, variant defaults, requires/conflicts, topological order, plan determinism |
| `internal/engine` | preflight failures, staging cleanup, atomic commit, rollback |
| `internal/project` | state round-trip, hash recording, drift classification, backup/restore |
| `internal/cli` | command wiring, flags, output, exit errors |

Commands:

```bash
go test ./...
go test ./internal/registry/ -run TestLoadRemote -v
go test ./... -race
go test -tags=integration ./...
```

### 2. Contract tests (registry repository)

The registry is a separate repository. Its CI must prove that declared content
is internally consistent:

- `manifest.yaml` parses and validates against the CLI's schema version.
- Every stack's `template.yaml` and template path exist.
- Every feature×supported-stack overlay exists.
- Every anchor referenced by an overlay exists exactly once in each supported
  base (render the base with fixture variables first).
- Overlay dependency graphs resolve (`requires`/`conflicts`).
- Removal metadata is coherent (`removable: false` explained/expected).

Failing a contract test blocks registry releases.

### 3. Golden tests (deterministic trees)

For representative selections, render a project with a seeded test RNG and
compare the tracked tree against a golden directory:

- file set and modes;
- file contents (`.env` secrets masked);
- `.smokeforge.json` and `.smokeforge.lock.json` (checksums normalized).

Golden updates are explicit and reviewed; never auto-accept in CI.

### 4. Build matrix (native semantics)

Generated projects must actually build/test. Representative matrix:

| Target | Commands |
| --- | --- |
| Go | `go build ./...`, `go test ./...` |
| Rust backends/desktop | `cargo check`, `cargo test` (GPUI/EGUI use `cargo check`) |
| Python | `uv sync`, `pytest` |
| Next.js | install, `npm run build`, `npm run lint` |
| iOS | `xcodebuild` / `swift build` on macOS |
| Android | `./gradlew assembleDebug`, `./gradlew test` |

Runners: Linux for Go/Rust/Python/Node/Android; macOS for iOS and desktop where
required. The registry CI decides the exact combination budget; at minimum,
every base and every overlay×base pair is compiled once.

### 5. CLI end-to-end

Implemented for M2 flags with a local fixture registry; wizard flows and mutation
lifecycle remain planned. Drive the compiled binary (or `NewRootCmd`) through:

- `new` with flags for each kind;
- wizard flows (Charm `teatest`) for representative paths;
- `add`/`remove` round-trips: generate → add → remove → compare to baseline;
- drift detection: edit a generated file, confirm refusal/force behavior;
- failure injection: kill during staging, confirm target untouched;
- `--dry-run` and `--json` output contracts.

## Fixtures

- Local manifest fixtures live in package-level `testdata/`.
- Local registry fixtures live in `testdata/registry/` with full template and
  overlay trees.
- HTTP tests use `httptest.Server` serving a tarball + checksum, built with the
  same helper as extraction tests.
- Never write to `os.UserCacheDir()` in tests; always pass `CacheDir:
  t.TempDir()`.

## Secrets and determinism

`crypto/rand` output is not deterministic. Tests must either:

- seed a test-only RNG via an internal option, or
- mask `.env` values in golden comparisons.

Production code must not expose a secret-generation override publicly.

## Race and platform coverage

- `go test -race ./...` runs in CI for the CLI.
- The CLI binary is built on Linux, macOS, and Windows.
- Filesystem tests avoid assumptions about case sensitivity, symlink support,
  and path separators; use `filepath` exclusively.

## Definition of done

A change is complete when:

- [ ] Failing test written first and observed failing.
- [ ] Minimal implementation makes it pass.
- [ ] `go test ./...` passes (including `-race` where relevant).
- [ ] `golangci-lint run` is clean.
- [ ] Relevant docs updated (command tables, manifest fields, operation types).
- [ ] No network or user-cache access in unit tests.
- [ ] Generated output remains deterministic.
- [ ] If templates changed: registry contract + build matrix pass.

## Related documents

- [Setup](setup.md)
- [Roadmap](roadmap.md)
- [Generation engine](../architecture/generation-engine.md)
```
