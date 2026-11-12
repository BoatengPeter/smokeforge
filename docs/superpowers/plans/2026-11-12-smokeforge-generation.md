# smokeforge M2 — Single-target generation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development for every production behavior. Steps use checkbox (`- [x]`) syntax for tracking.

**Status:** complete.

**Goal:** `smokeforge new` generates a single-target project from a local or remote registry, applying feature overlays, writing state/lockfiles, and committing atomically.

**Architecture:** Seven focused packages — `manifest` (schemas), `render` (strict templates), `patch` (declarative operations), `resolve` (selection → plan), `project` (state/lock/hashes), `engine` (staging/commit), `tui` + `cli` (wizard/flags). Generated-app framework internals remain deferred; tests use a neutral fixture stack.

**Tech stack:** Go 1.25, Cobra, `yaml.v3`, `huh`, `golang.org/x/mod/modfile`, `pelletier/go-toml/v2`, `golang.org/x/term`.

---

## Scope

In M2:

- Kinds: `backend`, `frontend`, `mobile`, `desktop` (single target).
- One stack per project, multiple feature overlays.
- Overlay operations: `addFiles`, `insertBefore`, `insertAfter`, `patchJSON`, `patchYAML`, `patchTOML`, `addDependency`, `env`.
- Variables: stack-declared with defaults, plus `project_name`, `project_slug`.
- State: `.smokeforge.json`, `.smokeforge.lock.json`, file hashes.
- Output: `.env`/`.env.example`, optional `git init`, atomic staging.
- CLI: `new` flags + interactive wizard, `--dry-run`, `--no-git`, `--keep-staging`.
- Integration: generate a Go fixture project and run `go build ./...`.

Out of M2: fullstack/multi-target, `add`/`remove`, `install`, official stack templates.

## Schema additions

Registry manifest feature/variant gains stack-to-overlay paths:

```yaml
features:
  - id: logging
    kinds: [backend]
    variants:
      - id: basic
        default: true
        overlays:
          test/plain: overlays/logging/basic
  - id: docker
    kinds: [backend]
    overlays:
      test/plain: overlays/docker
```

Rules: overlay paths are relative, local, slash-separated; every feature or
selected variant must provide an overlay for the selected stack.

Template metadata (`<template>/template.yaml`):

```yaml
id: test/plain
variables:
  - name: module_path
    prompt: Go module path
    default: example.com/app
  - name: greeting
    prompt: Greeting
    default: hello
```

## Package contracts

```go
// internal/render
func New(projectName string, variables map[string]string) (*Renderer, error)
func (r *Renderer) RenderPath(path string) (string, error)
func (r *Renderer) RenderBytes(content []byte) ([]byte, error)
func Slug(name string) string

// internal/patch
func LoadOverlayFile(path string) (*Overlay, error)
func (o *Overlay) Validate() error

// internal/resolve
type Selection struct {
    Kind      string
    Stack     string
    Features  []string
    Variables map[string]string
}
func Resolve(m *manifest.Manifest, sel Selection) (*Plan, error)
func ResolveVariables(stack manifest.Stack, tmpl manifest.Template, declared map[string]string) (map[string]string, error)

// internal/project
func NewState(name, kind string, ref RegistryRef, variables map[string]string, targets []Target, applied []AppliedOverlay) *State
func WriteState(dir string, state *State, lock *Lock) error
func LoadState(dir string) (*State, error)

// internal/engine
type Request struct {
    Registry *registry.Result
    Plan     *resolve.Plan
    Target   string
    NoGit    bool
    DryRun   bool
    KeepStaging bool
}
func Generate(ctx context.Context, req Request) (*Result, error)
```

## Tasks

### Task 1 — Manifest schema extensions

- [x] Failing tests: overlay path validation (relative/local/non-empty), feature overlay keys reference declared stacks, variant overlay keys reference declared stacks, template.yaml loading with unknown fields rejected, duplicate/empty variables rejected.
- [x] Extend `manifest.Feature`/`Variant` with `Overlays map[string]string`.
- [x] Add `manifest.Template`/`TemplateVariable` loader + validation.
- [x] Update `internal/manifest/testdata/manifest.yaml`.
- [x] Verify: `go test ./internal/manifest/...`, lint.

### Task 2 — Renderer

- [x] Failing tests: strict missing variable error, path rendering, slug rules, function map (`snake`, `kebab`, `camel`, `pascal`, `quote`, `default`), invalid template syntax.
- [x] Implement renderer with `missingkey=error`.
- [x] Verify: `go test ./internal/render/...`.

### Task 3 — Patch engine

- [x] Failing tests per operation:
  - anchor insertion exactly-once, marker wrapping, indentation, idempotency;
  - unsafe paths rejected;
  - JSON set/merge/append/delete;
  - YAML set/merge/delete with comments preserved;
  - TOML set + reparse validation;
  - dependency adapters: `go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`;
  - env entries.
- [x] Implement overlay schema (`overlay.yaml`) with exactly one op per item.
- [x] Implement ops and validation.
- [x] Verify: `go test ./internal/patch/...`.

### Task 4 — Resolver

- [x] Failing tests: unknown kind/stack, kind/stack mismatch, unsupported feature, variant default/required, requires/conflicts, duplicate feature, overlay path required, deterministic topological ordering, variable resolution (defaults, overrides, missing, unknown).
- [x] Implement `Resolve` and `ResolveVariables`.
- [x] Verify: `go test ./internal/resolve/...`.

### Task 5 — Project state

- [x] Failing tests: state round-trip, lock round-trip, JSON schema stable, hashes recorded, invalid schema rejected.
- [x] Implement `State`, `Lock`, atomic JSON writes.
- [x] Verify: `go test ./internal/project/...`.

### Task 6 — Engine

- [x] Failing tests: renders base + overlays into staging, target untouched on failure, atomic commit, non-empty target refused, dry-run writes nothing, env generation, overlay markers/hashes recorded, optional git init (fixture with `--no-git` for tests).
- [x] Implement engine orchestration and staging.
- [x] Verify: `go test ./internal/engine/...`.

### Task 7 — CLI `new`

- [x] Failing CLI tests with local fixture registry and `--no-input`: flags resolve, unknown stack/feature errors, `--dry-run` output, generated tree, `--no-git`.
- [x] Implement flags and help. Wizard only runs when stdin is a TTY and `--no-input` is absent.
- [x] Verify: `go test ./internal/cli/...`.

### Task 8 — Integration fixture

- [x] Add `testdata/registry/` with neutral `test/plain` Go stack + logging overlay.
- [x] Integration test: generate, assert markers/state, run `go build ./...`.
- [x] Manual: `go run ./cmd/smokeforge new /tmp/demo --registry ./testdata/registry ...`.

### Task 9 — Documentation and verification

- [x] Update README command/status/flags.
- [x] Update docs/architecture (status), generation engine, CLI, manifest, overlays, project state, roadmap, Mermaid status map.
- [x] Update AGENTS.md and CLAUDE.md status.
- [x] `go test ./... -race`, `golangci-lint run ./...`, `go build ./...`.
- [x] Commit in focused commits.

## Acceptance

- A local fixture project is generated atomically with a base template and an
  overlay, contains schema-correct state/lock files, and builds with `go build`.
- Every failed generation leaves the target path untouched.
- Missing variables, invalid selections, unsafe paths, and malformed patches
  fail with actionable errors before commit.
- Documentation reflects the implemented M2 behavior and remaining M3-M5 work.
