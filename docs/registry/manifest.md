# Registry manifest reference

**Status:** schema is planned; manifest loading/validation is the current
foundation milestone.

## Registry repository layout

```text
smokeforge-registry/
├── manifest.yaml
├── templates/
│   ├── backend/
│   │   ├── go/chi/
│   │   ├── rust/axum/
│   │   └── python/fastapi/
│   ├── frontend/next/
│   ├── mobile/
│   │   ├── ios/swiftui/
│   │   └── android/compose/
│   ├── desktop/rust/
│   │   ├── gpui/
│   │   └── egui/
│   └── workspace/fullstack/
├── overlays/
│   ├── backend/
│   │   ├── go/
│   │   │   ├── database/postgres/
│   │   │   ├── cache/redis/
│   │   │   ├── queue/rabbitmq/
│   │   │   ├── auth/jwt/
│   │   │   └── totp/
│   │   ├── rust/...
│   │   └── python/...
│   └── shared/
│       ├── docker/
│       └── ci/github/
└── tests/
```

Each base template directory contains `template.yaml` and a `files/` tree. Each
overlay directory contains `overlay.yaml` and optional `files/`.

## Distribution and versioning

- Registry releases are git tags (`v1.0.0`, `v1.1.0`, ...).
- Each release publishes two assets:
  - `registry.tar.gz` — the registry tree at the tag;
  - `registry.sha256` — `<sha256>  registry.tar.gz`.
- `smokeforge` downloads from
  `<source>/download/<tag>/registry.tar.gz` (GitHub releases layout).
- `latest` resolves to the release marked latest and is cached by source+tag.
- The tarball must contain a top-level `registry/` directory; extraction strips
  it into the cache entry.
- No symlinks or hard links are allowed in the archive.
- Templates are versioned with the registry; the CLI does not maintain a
  separate compatibility matrix beyond `schemaVersion`.

## Top-level manifest

```yaml
schemaVersion: 1
name: smokeforge-registry
kinds:
  backend:  {max: 1}
  frontend: {max: 1}
  mobile:   {max: 2}
  desktop:  {max: 1}
stacks:
  - id: go/chi
    kind: backend
    name: Go (chi)
    template: templates/backend/go/chi
    variables: [module_path, port]
  - id: rust/axum
    kind: backend
    name: Rust (axum)
    template: templates/backend/rust/axum
    variables: [crate_name, port]
  - id: python/fastapi
    kind: backend
    name: Python (FastAPI)
    template: templates/backend/python/fastapi
    variables: [package_name, port]
  - id: next
    kind: frontend
    name: Next.js
    template: templates/frontend/next
    variables: [package_name, port]
  - id: ios/swiftui
    kind: mobile
    name: iOS (SwiftUI)
    template: templates/mobile/ios/swiftui
    variables: [bundle_id, team_id]
  - id: android/compose
    kind: mobile
    name: Android (Compose)
    template: templates/mobile/android/compose
    variables: [application_id, package_name]
  - id: desktop/gpui
    kind: desktop
    name: Desktop (GPUI)
    template: templates/desktop/rust/gpui
    variables: [crate_name]
  - id: desktop/egui
    kind: desktop
    name: Desktop (EGUI)
    template: templates/desktop/rust/egui
    variables: [crate_name]
features:
  - id: database
    name: Database
    kinds: [backend]
    stacks: [go/chi, rust/axum, python/fastapi]
    variants:
      - {id: postgres, name: PostgreSQL, default: true}
  - id: cache
    name: Cache
    kinds: [backend]
    variants:
      - {id: redis, name: Redis, default: true}
  - id: queue
    name: Queue
    kinds: [backend]
    variants:
      - {id: rabbitmq, name: RabbitMQ, default: true}
  - id: auth
    name: Authentication
    kinds: [backend]
    variants:
      - {id: jwt, name: JWT, default: true}
  - id: totp
    name: TOTP two-factor authentication
    kinds: [backend]
    requires: [auth:jwt]
  - id: docker
    name: Docker and dev compose
    kinds: [backend, frontend, desktop]
  - id: ci
    name: CI pipeline
    variants:
      - {id: github, name: GitHub Actions, default: true}
```

## Field reference

### Manifest

| Field | Type | Rules |
| --- | --- | --- |
| `schemaVersion` | int | Must be `1`. |
| `name` | string | Non-empty. |
| `kinds` | map[string]Kind | Non-empty; keys are kind names. |
| `stacks` | []Stack | Non-empty; IDs unique. |
| `features` | []Feature | IDs unique. |

### Kind

| Field | Type | Rules |
| --- | --- | --- |
| `max` | int | `>= 0`; selection limit per project. `0` means not selectable. |

### Stack

| Field | Type | Rules |
| --- | --- | --- |
| `id` | string | Unique; slash-separated namespacing recommended. |
| `kind` | string | Must exist in `kinds`. |
| `name` | string | Human-readable label. |
| `template` | string | Relative path inside the registry. |
| `variables` | []string | Non-empty; unique; names must match template usage. |

### Feature

| Field | Type | Rules |
| --- | --- | --- |
| `id` | string | Unique; referenced by CLI flags and state. |
| `name` | string | Human-readable label. |
| `kinds` | []string | Kind names from `kinds`; at least one of `kinds`/`stacks` required. |
| `stacks` | []string | Stack IDs; restricts the feature to specific stacks. |
| `variants` | []Variant | Unique IDs; optional for boolean features. |
| `requires` | []string | `feature` or `feature:variant`; must reference declared features. |
| `conflicts` | []string | Same reference syntax; mutual exclusion. |

### Variant

| Field | Type | Rules |
| --- | --- | --- |
| `id` | string | Unique within the feature. |
| `name` | string | Human-readable label. |
| `default` | bool | At most one default per feature. |

Variant selection syntax: `--feature cache:redis`. A feature without an explicit
variant uses its default when one is declared; otherwise selecting the feature
without a variant is an error.

## Validation rules

Loading is strict: unknown YAML fields are rejected (`KnownFields(true)`), and
`Validate()` additionally enforces:

1. `schemaVersion == 1`.
2. Non-empty `name`, `kinds`, and `stacks`.
3. Unique stack IDs; stack kinds exist; templates and variables non-empty.
4. Unique feature IDs; `kinds`/`stacks` references exist.
5. Unique variant IDs per feature; at most one default.
6. `requires`/`conflicts` reference declared features; self-references rejected.
7. Variant references in `requires`/`conflicts` must exist on the target
   feature (resolved during selection).
8. Kind `max` values are non-negative.

Validation failures are fatal before any resolution or generation work.

## Template manifests

Each base template carries `template.yaml`:

```yaml
id: go/chi
variables:
  - name: module_path
    prompt: Go module path
    default: example.com/app
  - name: port
    prompt: HTTP port
    default: "8080"
anchors:
  - file: internal/app/app.go
    name: imports
  - file: internal/app/app.go
    name: routes
```

Anchors listed here are documentation for overlay authors; the operator still
validates them against the actual file content at generation time.

## Cache layout

```text
$XDG_CACHE_HOME/smokeforge/registry/
└── <16-hex-prefix-of-sha256(source + NUL + tag)>/
    ├── manifest.yaml
    ├── templates/...
    └── overlays/...
```

Cache commits are atomic: the download extracts into a temp directory, verifies
the checksum, and renames into place. `registry update` removes the entry and
re-downloads it.

## Local registries

Any directory containing `manifest.yaml` can be passed via
`--registry ./path`. The tag is reported as `local`, archives/checksums are not
used, and validation still applies. This is the primary fixture mechanism for
tests and template development.

## Related documents

- [Overlay and patch model](overlays.md)
- [Registry overview in the root README](../../README.md#registry-and-reproducibility)
- [Testing strategy](../development/testing.md)
```
