# Generation engine

**Status:** planned. This document specifies the behavior the engine must
implement in the next milestones.

## Responsibilities

The engine turns a validated selection into a committed project tree. It is
split into four focused packages:

| Package | Responsibility |
| --- | --- |
| `internal/resolve` | Validate the selection and produce an ordered plan. |
| `internal/render` | Render template files with strict variables. |
| `internal/patch` | Apply anchor and structured operations. |
| `internal/engine` | Preflight, stage, apply, commit, roll back. |

## Plan model

```go
type Plan struct {
    Registry  RegistryPin
    Variables map[string]string
    Targets   []TargetPlan
    RootOps   []patch.Op
}

type TargetPlan struct {
    Path     string        // relative to project root, e.g. "apps/api"
    Stack    string        // e.g. "go/chi"
    Template TemplateRef
    Overlays []OverlayRef  // dependency order
}
```

Resolution rules:

1. Normalize selections (`feature` defaults to its declared default variant).
2. Enforce per-kind limits from the registry manifest.
3. Reject features whose kind/stack is unsupported.
4. Enforce `requires` and `conflicts`; report the full dependency chain.
5. Topologically sort overlays by `requires`, then by registry-declared order.
6. Detect duplicate operations on the same file/anchor that cannot compose.
7. Assign variables, applying precedence: flags → wizard → stack defaults.

The plan is deterministic: same registry, selections, and variables produce the
same operation order and output bytes.

## Rendering

- Go `text/template`, with a small function map (`lower`, `upper`, `snake`,
  `kebab`, `camel`, `pascal`, `quote`, `default`).
- **Strict variables:** rendering fails on any missing key. Templates never see
  `<no value>`.
- Paths are rendered too, so file names can contain the project slug.
- File modes come from the registry archive; executable bits are preserved.
- Binary files are copied verbatim (no template parse) based on manifest
  declarations or NUL-byte detection.
- Generated secrets use `crypto/rand` and are written to `.env` only; they are
  not part of deterministic snapshots (tests seed or mask them).

## Patch model

Overlays operate through a closed set of declarative operations:

| Operation | Purpose |
| --- | --- |
| `addFiles` | Copy rendered files into the target. Never overwrites silently. |
| `insertBefore` / `insertAfter` | Insert a marked block adjacent to a named anchor. |
| `replaceBlock` | Replace the content between begin/end markers (creation only). |
| `patchJSON` | Set/merge/append values in JSON documents. |
| `patchYAML` | Node-preserving YAML edits (including compose files). |
| `patchTOML` | TOML edits, verified by re-parsing after the write. |
| `addDependency` | Language-adapter dependency edits. |
| `env` | Append `.env.example` entries. |

### Anchors

Base templates include anchor lines:

```go
// smokeforge:anchor imports
// smokeforge:anchor routes
```

```python
# smokeforge:anchor routers
```

```yaml
# smokeforge:anchor services
```

Each operation must match its anchor exactly once. Zero or multiple matches abort
the plan during preflight — before any file is written.

### Inserted blocks

Insertions are wrapped in markers:

```go
// smokeforge:begin cache:redis
// ... generated code ...
// smokeforge:end cache:redis
```

Markers make removal exact and idempotent. Applying the same overlay twice is a
no-op (detected by markers/state), not a duplicated block.

### Structured patches

- JSON: path-based set/merge/append. Object key order is preserved where the
  underlying writer allows; formatting is normalized and validated.
- YAML: node-level edits preserve comments and key order (`yaml.v3` nodes).
- TOML: edited then re-parsed; invalid output aborts the operation.
- Dependency adapters:
  - Go: `golang.org/x/mod/modfile` for `go.mod`.
  - Node: JSON dependency maps in `package.json`.
  - Python: `pyproject.toml` dependency arrays.
  - Rust: dependency tables in `Cargo.toml`.

Each adapter exposes `Add`, `Has`, and `Remove` and must leave a parseable file.

## Staging and commit

```text
parent/
├── .smokeforge-staging-<random>/   render here
└── <target>/                       created by atomic rename on success
```

- Staging lives on the same filesystem as the target, so the final step is a
  rename, not a copy.
- Preflight runs against rendered content in staging, but before the target is
  touched.
- On success: write state/lock/env inside staging, optionally `git init`, then
  rename to the target path.
- On failure: remove staging (or retain it with `--keep-staging`) and return an
  error. The target path is never partially written.

For `add`/`remove`, the engine works in place but is transactional:

1. Validate the full operation set.
2. Copy impacted files to `.smokeforge/backups/<timestamp>/`.
3. Apply operations, verifying each one.
4. Update state/lock last.
5. On any error, restore the backup and leave state unchanged.

## `--dry-run` and `--json`

`--dry-run` performs steps through preflight, applies rendering and operations
to an in-memory/temporary tree, and prints:

- resolved targets and stacks;
- ordered overlay list;
- file additions/modifications per target;
- anchors and structured paths that will be modified;
- variables in use (secrets redacted).

`--json` emits the same information as a stable JSON document for tooling.

## Determinism guarantees

Given identical inputs and registry pin:

- operation order is identical;
- file contents are byte-identical (except generated secrets in `.env`);
- file modes and paths are identical;
- `.smokeforge.json` hashes are identical (secrets excluded).

Tests enforce this with golden trees.

## Error taxonomy

| Error class | Example | Behavior |
| --- | --- | --- |
| Selection | `totp` without `auth` | List the unmet requirement and valid choices. |
| Target | target non-empty | Refuse; suggest a new directory or clearing it. |
| Anchor | anchor missing/twice | Abort preflight; name file, anchor, overlay. |
| Patch | TOML edit produced invalid file | Abort; restore/leave target untouched. |
| Drift | user edited a generated file before `add` | Show summary; refuse or require `--force`. |
| Commit | rename fails | Keep staging; report path and recovery steps. |
| Install | missing toolchain | Rerunnable failure with install guidance. |

## Related documents

- [Overlay and patch model](../registry/overlays.md)
- [Project state](project-state.md)
- [Testing strategy](../development/testing.md)
```
