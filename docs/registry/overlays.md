# Overlay and patch model

**Status:** implemented for project creation. Removal reversal, drift checks, and
`add`/`remove` are planned for M4.

## Why overlays

A base template provides a working project for one stack. Features are layered on
top as overlays. This avoids a combinatorial preset matrix while keeping every
generated project buildable and reviewable.

Overlays are **declarative data**. They cannot execute commands. The CLI is the
only component that writes files.

## Base template contract

A base template must:

1. Build and pass its own tests before any overlay is applied.
2. Include anchors at every extension point overlays may need.
3. Keep dependency declarations in standard manifests (`go.mod`, `Cargo.toml`,
   `pyproject.toml`, `package.json`).
4. Avoid generated secrets in committed files.
5. Declare variables in `template.yaml`.
6. Put renderable content under `files/`; `.tmpl` files are rendered, other files
   are copied verbatim.

Anchors are comments that survive formatting:

```go
// smokeforge:anchor imports
import (
    "net/http"
)
```

```python
# smokeforge:anchor routers
```

```yaml
# smokeforge:anchor services
services: {}
```

Rules:

- An anchor must match exactly once per file. Zero or multiple matches abort
  generation.
- Anchor lines are preserved in generated output.
- Overlays never edit anchor lines; they insert marked blocks before or after
  them.

## Overlay manifest

```yaml
id: logging/basic
name: Basic logging
files: files
ops:
  - addFiles:
      - path: logger.go.tmpl
        dest: internal/log/logger.go
  - insertAfter:
      file: main.go
      anchor: imports
      marker: logging:basic
      content: '"example.com/app/internal/log"'
  - env:
      key: LOG_LEVEL
      example: info
```

### Fields

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | yes | Human-readable overlay ID (typically `feature:variant`). |
| `name` | no | Human-readable label. |
| `files` | no | Source directory for `addFiles`; defaults to `files`. |
| `ops` | yes | Ordered declarative operations. |

Feature applicability, variants, dependency ordering, and conflicts live in the
registry manifest (`feature.kinds`, `feature.stacks`, `requires`, `conflicts`,
and the feature/variant `overlays` maps), not in the overlay file.

## Operation types

Every item in `ops` must set exactly one operation.

### `addFiles`

Copies rendered files from the overlay directory into the project.

```yaml
- addFiles:
    - {path: internal/db/db.go.tmpl, dest: internal/db/db.go}
    - {path: migrations/0001_init.sql, dest: migrations/0001_init.sql}
```

Fields: `path` (relative to the overlay's files directory), `dest` (rendered
destination path), and optional `overwrite` (default `false`). Destinations must
not already exist unless `overwrite: true`. Source `.tmpl` files are rendered.

### `insertBefore` / `insertAfter`

Inserts a rendered block adjacent to an anchor, wrapped in markers:

```go
// smokeforge:begin logging:basic
import "example.com/app/internal/log"
// smokeforge:end logging:basic
```

```yaml
- insertAfter:
    file: main.go
    anchor: imports
    marker: logging:basic
    content: '"example.com/app/internal/log"'
```

Fields: `file`, `anchor`, optional `marker` (defaults to the overlay ID), and
`content` (rendered as a template). The block inherits the anchor line's
indentation. Comment syntax is chosen from the file extension (`//`, `#`, `--`,
or `<!-- -->`).

Applying the same marker twice is a no-op. Multiple overlays may insert at the
same anchor; each keeps its own marker.

### `patchJSON`

Set, merge, append, or delete a value at a dot-separated path:

```yaml
- patchJSON:
    file: package.json
    op: set            # set | merge | append | delete
    path: dependencies.ioredis
    value: "^5.4.1"
```

Objects are created along missing `set` paths. `merge` combines nested objects.
`append` requires an array target. The document is re-marshaled with two-space
indentation and validated.

### `patchYAML`

Node-preserving YAML edits for compose files and workflows:

```yaml
- patchYAML:
    file: deploy/docker-compose.yml
    op: set
    path: services.redis
    value:
      image: redis:7-alpine
      ports: ["6379:6379"]
      volumes: ["redis-data:/data"]
```

Comments and key order are preserved by editing the YAML node tree. Supports
`set`, `merge`, `append`, and `delete`.

### `patchTOML`

TOML edits for `Cargo.toml` and `pyproject.toml`:

```yaml
- patchTOML:
    file: Cargo.toml
    op: set
    path: dependencies.redis
    value: {version: "0.27", features: ["tokio-comp"]}
```

The file is decoded, patched, re-encoded, and re-parsed. Invalid output aborts
the operation.

### `addDependency`

Language-aware dependency edits:

```yaml
- addDependency:
    file: go.mod
    module: github.com/redis/go-redis/v9
    version: v9.7.0
```

```yaml
- addDependency:
    file: Cargo.toml
    module: redis
    version: "0.27"
    features: [tokio-comp]
```

Adapters:

| File | Behavior |
| --- | --- |
| `go.mod` | `golang.org/x/mod/modfile`; adds a require entry. |
| `package.json` | Adds or updates `dependencies[module]`. |
| `pyproject.toml` | Appends `module==version` to `project.dependencies`. |
| `Cargo.toml` | Adds a version string or a table with `features`. |

An existing dependency with a different version is an error; an identical entry
is a no-op.

### `env`

Records an environment variable consumed by the generated project:

```yaml
- env:
    key: REDIS_URL
    example: redis://localhost:6379
    secret: false
```

The engine aggregates entries into `.env.example` and `.env`. `secret: true`
generates a random value for `.env` while keeping the example placeholder. `.env`
and `.smokeforge/` are added to `.gitignore`.

## Ordering

1. Overlays are topologically sorted by feature `requires`; ties keep the user's
   selection order.
2. Operations within an overlay run in declared order.
3. All paths, anchors, and operations are validated as they are applied. Any
   failure aborts the staged generation and leaves the target untouched.

## Path and archive safety

- Overlay and destination paths must be relative, clean, forward-slash paths that
  do not escape their root.
- Template directories reject symlinks and non-regular files.
- Registry archives reject traversal, links, devices, and oversized entries.

## Removal and mutation (planned)

`smokeforge remove` will excise marked blocks and reverse supported operations
when the affected content is unchanged. M2 records per-file hashes and markers in
`.smokeforge.json` so M4 can implement drift detection and reversal safely.

## Authoring checklist

Before submitting an overlay to the registry:

- [ ] It applies cleanly to every stack declared in the manifest overlay map.
- [ ] Every anchor used exists exactly once in each supported base.
- [ ] Dependency additions use `addDependency` where possible.
- [ ] Inserted code blocks are marked and compile.
- [ ] `.env.example` entries are documented and referenced by generated code.
- [ ] Contract tests pass for each supported base.
- [ ] A representative generated project builds and passes tests.

## Worked example: Go + logging

Base `test/plain` exposes:

```go
// main.go
import (
    "fmt"
    // smokeforge:anchor imports
)
```

Overlay `logging/basic` adds `internal/log/logger.go`, inserts `"os"` after the
`imports` anchor, inserts `_ = os.Args` after an `after-print` anchor, and adds
`LOG_LEVEL` to the environment files. The generated `main.go` contains:

```go
// smokeforge:begin logging:basic
"os"
// smokeforge:end logging:basic
```

The full fixture lives in `testdata/registry/` and is exercised by the
integration test.

## Related documents

- [Registry manifest](manifest.md)
- [Generation engine](../architecture/generation-engine.md)
- [Project state](../architecture/project-state.md)
