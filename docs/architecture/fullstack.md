# Fullstack composition

**Status:** planned. Defines how multi-target projects are modeled and
generated.

## Kinds and limits

The registry manifest declares per-kind selection limits:

| Kind | Max in a project | Examples |
| --- | --- | --- |
| `backend` | 1 | `go/chi`, `rust/axum`, `python/fastapi` |
| `frontend` | 1 | `next` |
| `mobile` | 2 | `ios/swiftui` + `android/compose` |
| `desktop` | 1 | `desktop/gpui` **or** `desktop/egui` |

A fullstack project is one API target plus any supported clients. A backend-only
project is a fullstack project with no clients; a client-only project is one
without an API.

## Monorepo layout

```text
acme/
  apps/
    api/                  # backend target (required for fullstack)
    web/                  # Next.js target
    ios/                  # SwiftUI target
    android/              # Jetpack Compose target
    desktop/              # GPUI or EGUI target
  deploy/
    docker-compose.yml    # dev services: database, redis, rabbitmq, etc.
    .env.example          # shared local configuration
  .github/
    workflows/
      ci.yml
  .smokeforge.json
  .smokeforge.lock.json
  README.md
  Makefile                # or Taskfile, chosen by the root workspace overlay
```

Target paths are fixed by kind (`apps/api`, `apps/web`, `apps/ios`,
`apps/android`, `apps/desktop`) unless a stack declares an override.

## Composition rules

1. **Root workspace base** — a `workspace/fullstack` template renders the root
   files (README, task runner, `.gitignore`, root env documentation).
2. **One base template per target** — each target is rendered independently
   with its own variables (module path, bundle ID, application ID).
3. **Overlay targeting:**
   - backend features (`database`, `cache`, `queue`, `auth`, `totp`) apply only
     to the API target;
   - repo-wide features (`docker`, `ci`) apply at the root and may read target
     metadata;
   - client-specific overlays (future) apply to their target.
4. **Operation ordering** — base templates first (in target order), then
   overlays sorted by dependency and registry order. Root operations run last so
   they can aggregate generated services and workflows.
5. **No cross-target file writes** — an overlay may not write outside its
   declared target; root overlays use root-relative paths only.

## API-client wiring

Clients must not require manual URL hunting. At generation time:

- The API port is allocated (collision-checked) and recorded in state.
- Web: `.env.example`/`.env.local` receives `API_URL=http://localhost:<port>`.
- iOS: a generated `Config.swift` (or xcconfig) contains the base URL for debug.
- Android: `BuildConfig` field or `local.properties` entry for the base URL.
- Desktop: a config module/env var for the base URL.
- Root `.env.example` documents shared values; compose uses the same names.

This wiring is implemented by small client-specific overlays shipped with each
client base, not by hardcoded CLI logic.

## Docker and CI aggregation

`docker` (root) renders:

- `deploy/docker-compose.yml` with services for the selected backing features
  (Postgres, Redis, RabbitMQ) and the API where applicable;
- `.env.example` entries consumed by compose.

`ci` (root) renders a GitHub Actions workflow that:

- detects generated targets from `.smokeforge.json`;
- runs each ecosystem's standard build/test commands;
- uses path filters so a web-only change does not rebuild native clients.

Overlays patch compose/workflow files through structured YAML operations, not
string concatenation, so multiple features compose deterministically.

## Adding clients later

`smokeforge add next` on a backend-only project is **not** just an overlay: it
introduces a new target. Planned behavior for v1.1:

1. Detect that the feature is actually a stack/kind addition.
2. Resolve the new base + client wiring overlay.
3. Render the new target under `apps/<kind>`.
4. Apply root updates (compose, CI, env docs).
5. Record the new target in state.

v1 supports adding only feature overlays to existing targets.

## Validation

- Fullstack requires an API stack; clients alone use their own kind.
- Multiple clients of the same kind beyond the limit are rejected.
- Desktop GPUI and EGUI are mutually exclusive per project.
- Client/API compatibility is declared in the manifest (`stacks`/`kinds`), not
  assumed by the CLI.

## Related documents

- [CLI lifecycle](cli.md)
- [Registry manifest](../registry/manifest.md)
- [Overlay model](../registry/overlays.md)
```
