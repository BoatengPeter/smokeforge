# Overlay and patch model

**Status:** planned. This document is the authoring contract for feature
overlays and the implementation contract for the patch engine.

## Why overlays

A base template provides a working project for one stack. Features are layered on
top as overlays. This avoids a combinatorial preset matrix while keeping every
generated project buildable and reviewable.

Overlays are **declarative data**. They cannot execute commands. The CLI is the
only component that writes files.

## Base template contract

A base template must:

1. Compile/build and pass its own tests before any overlay is applied.
2. Include anchors at every extension point overlays may need.
3. Keep dependency declarations in standard manifests (`go.mod`, `Cargo.toml`,
   `pyproject.toml`, `package.json`, Gradle files).
4. Avoid generated secrets in committed files.
5. Use lowercase, slash-namespaced stack IDs.
6. Document variables in `template.yaml`.

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

- Anchor names are unique per file, not globally.
- An anchor must match exactly once in the rendered base file.
- Anchor lines are preserved in generated output (they are part of the template).
- Overlays never edit anchor lines; they insert around them.

## Overlay manifest

```yaml
id: cache/redis
name: Redis cache
supports:
  kinds: [backend]
  stacks: [go/chi, rust/axum, python/fastapi]
requires: []
conflicts: []
removable: true
files: files/
ops:
  - addFiles:
      - path: internal/cache/redis.go.tmpl
        dest: internal/cache/redis.go
  - addDependency:
      file: go.mod
      module: github.com/redis/go-redis/v9
      version: v9.7.0
  - insertAfter:
      file: internal/app/app.go
      anchor: imports
      marker: cache:redis
      content: |
        "github.com/acme/app/internal/cache"
  - patchJSON:
      file: package.json
      path: dependencies.ioredis
      value: "^5.4.1"
  - env:
      key: REDIS_URL
      example: redis://localhost:6379
```

### Fields

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | yes | Registry-unique overlay ID (usually `feature:variant`). |
| `name` | yes | Human-readable label. |
| `supports` | yes | Kinds and/or stacks this overlay can apply to. |
| `requires` | no | Other overlays/features that must be present. |
| `conflicts` | no | Overlays that must not coexist. |
| `removable` | no | Defaults to `true`; `false` for creation-only overlays. |
| `files` | no | Directory of files referenced by `addFiles`. |
| `ops` | yes | Ordered declarative operations. |

## Operation types

### `addFiles`

Copies rendered files to destinations. Existing destinations cause a preflight
conflict unless the overlay declares `overwrite: true` (only allowed at
creation; non-removable).

```yaml
- addFiles:
    - {path: internal/db/postgres.go.tmpl, dest: internal/db/postgres.go}
    - {path: migrations/0001_init.sql, dest: migrations/0001_init.sql}
```

### `insertBefore` / `insertAfter`

Inserts a block adjacent to an anchor. The block is wrapped in markers:

```go
// smokeforge:begin cache:redis
import "github.com/acme/app/internal/cache"
// smokeforge:end cache:redis
```

- `marker` defaults to the overlay ID.
- Applying the same marker twice is a no-op (idempotent).
- Multiple overlays may insert at the same anchor; order is deterministic
  (dependency order, then registry order).
- Removal excises the exact marked block.

### `replaceBlock`

Replaces content between existing begin/end markers. Allowed only at creation
time or when the overlay declares itself non-removable. Used sparingly (for
example, changing a default in a generated config that has no anchor).

### `patchJSON`

Set, merge, append, or delete a value at a JSON path:

```yaml
- patchJSON:
    file: package.json
    op: set            # set | merge | append | delete
    path: dependencies.ioredis
    value: "^5.4.1"
```

Paths use dot notation with array indexes (`scripts.build`). The document is
validated before and after the edit.

### `patchYAML`

Node-preserving edits, used for compose files and workflows:

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

Comments and key order are preserved where the YAML node tree allows.

### `patchTOML`

TOML edits for `Cargo.toml` / `pyproject.toml`:

```yaml
- patchTOML:
    file: Cargo.toml
    op: set
    path: dependencies.redis
    value: {version: "0.27", features: ["tokio-comp"]}
```

The file is re-parsed after the edit; invalid output aborts the operation.

### `addDependency`

Language-aware dependency edits. Prefer this over raw `patch*` for dependency
manifests:

| Language | Target file | Adapter behavior |
| --- | --- | --- |
| Go | `go.mod` | `golang.org/x/mod/modfile`; adds require entry |
| Rust | `Cargo.toml` | TOML dependency table edit + reparse |
| Python | `pyproject.toml` | Dependency array edit + reparse |
| Node | `package.json` | JSON dependency map edit |

Each adapter implements `Add`, `Has`, and `Remove` so overlays are reversible.

### `env`

Appends a documented variable to `.env.example` (and to `.env` at creation):

```yaml
- env:
    key: REDIS_URL
    example: redis://localhost:6379
    secret: false
```

`secret: true` generates a random value at creation time and never commits it.

## Ordering and conflicts

1. Overlays are topologically sorted by `requires`.
2. Ties are broken by registry-declared order (`order` field, then ID).
3. Operations within an overlay run in declared order.
4. Preflight rejects:
   - two `replaceBlock` operations on the same range;
   - `addFiles` destination collisions;
   - an anchor referenced by an operation but missing/duplicated;
   - structured patches whose parent path does not exist;
   - dependency edits that would produce a duplicate conflicting constraint.
5. Insertions at the same anchor are allowed and deterministic; each is wrapped
   in its own marker.

## Removal semantics

Removal is supported for overlays marked `removable: true`:

- delete files added by the overlay if their current hash matches the recorded
  post-add hash;
- excise marked blocks by marker name;
- reverse structured patches only if the affected entries are unchanged;
- remove env entries only if unchanged;
- remove dependency entries added by the overlay;
- refuse if user edits make the reverse operation unsafe.

Refusals list the conflicting files and suggest manual steps.

## Authoring checklist

Before submitting an overlay to the registry:

- [ ] It applies cleanly to every stack listed in `supports`.
- [ ] Every anchor used exists exactly once in each supported base.
- [ ] Dependency additions use `addDependency`, not raw text edits.
- [ ] Inserted code blocks are wrapped in markers and compile.
- [ ] `.env.example` entries are documented and referenced by generated code.
- [ ] Removal works (or `removable: false` is justified).
- [ ] Contract tests pass for each supported base.
- [ ] A representative generated project builds and passes tests.

## Worked example: Go + Redis

Base `go/chi` exposes:

```go
// internal/app/app.go
// smokeforge:anchor imports
// smokeforge:anchor clients
```

Overlay `cache/redis`:

```yaml
id: cache/redis
supports: {kinds: [backend], stacks: [go/chi]}
ops:
  - addFiles:
      - {path: internal/cache/redis.go.tmpl, dest: internal/cache/redis.go}
  - addDependency: {file: go.mod, module: github.com/redis/go-redis/v9, version: v9.7.0}
  - insertAfter:
      file: internal/app/app.go
      anchor: imports
      marker: cache:redis
      content: '	"github.com/acme/app/internal/cache"'
  - insertAfter:
      file: internal/app/app.go
      anchor: clients
      marker: cache:redis-client
      content: '	redisClient := cache.NewRedis(cfg.RedisURL)'
  - env: {key: REDIS_URL, example: "redis://localhost:6379"}
```

Generated fragment:

```go
// smokeforge:begin cache:redis
	"github.com/acme/app/internal/cache"
// smokeforge:end cache:redis
```

Removing `cache:redis` deletes `internal/cache/redis.go`, excises both marked
blocks, removes the `go.mod` requirement, and removes `REDIS_URL` from `.env`
(if untouched).

## Related documents

- [Registry manifest](manifest.md)
- [Generation engine](../architecture/generation-engine.md)
- [Project state](../architecture/project-state.md)
```
